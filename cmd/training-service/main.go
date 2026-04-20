package main

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"net"
	"os"
	"time"

	pb "github.com/MAMUER/Project/api/gen/training"
	"github.com/MAMUER/Project/internal/db"
	"github.com/MAMUER/Project/internal/logger"
	"github.com/MAMUER/Project/internal/queue"
	"github.com/MAMUER/Project/internal/sanitize"
	"github.com/MAMUER/Project/internal/validator"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type trainingServer struct {
	pb.UnimplementedTrainingServiceServer
	db          *sql.DB
	log         *logger.Logger
	rabbitQueue queue.Publisher // ← ИНТЕРФЕЙС
}

func (s *trainingServer) GeneratePlan(ctx context.Context, req *pb.GeneratePlanRequest) (*pb.GeneratePlanResponse, error) {
	s.log.Info("GeneratePlan request received",
		zap.String("user_id", req.UserId),
		zap.String("class", req.ClassificationClass),
		zap.Int32("duration_weeks", req.DurationWeeks),
		zap.Int("available_days", len(req.AvailableDays)),
	)

	if err := ctx.Err(); err != nil {
		s.log.Warn("Request cancelled", zap.Error(err))
		return nil, status.Error(codes.Canceled, "request cancelled")
	}

	if err := validator.ValidateGeneratePlanRequest(req); err != nil {
		s.log.Warn("Invalid generate plan request", zap.Error(err))
		return nil, err
	}

	// Check if user already has an active plan
	var existingCount int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM training_plans WHERE user_id = $1 AND status = 'active'`, req.UserId).Scan(&existingCount)
	if err != nil {
		s.log.Error("Failed to check existing plans", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}
	if existingCount > 0 {
		s.log.Warn("User already has an active plan", zap.String("user_id", req.UserId))
		return nil, status.Error(codes.AlreadyExists, "user already has an active plan")
	}

	classificationClass := sanitize.String(req.ClassificationClass)
	planID := uuid.New().String()

	planData := map[string]interface{}{
		"name":           "Персонализированная программа",
		"class":          classificationClass,
		"confidence":     req.Confidence,
		"duration_weeks": int(req.DurationWeeks),
	}

	startDate := time.Now()
	endDate := startDate.AddDate(0, 0, int(req.DurationWeeks)*7)

	// Сохраняем план в нормализованную схему
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.log.Error("Failed to begin transaction", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to begin transaction")
	}
	defer func() {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			s.log.Error("Failed to rollback transaction", zap.Error(rbErr))
		}
	}()

	s.log.Info("Inserting into training_plans", zap.String("planID", planID), zap.String("userID", req.UserId), zap.String("class", classificationClass))
	_, err = tx.ExecContext(ctx, `
		INSERT INTO training_plans (id, user_id, name, training_goal, duration_weeks, generated_at, start_date, end_date, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, planID, req.UserId, "Персонализированная программа", classificationClass, int32(req.DurationWeeks), time.Now(), startDate.Truncate(24*time.Hour), endDate.Truncate(24*time.Hour), "active")
	if err != nil {
		s.log.Error("Failed to insert plan", zap.Error(err), zap.String("planID", planID))
		return nil, status.Error(codes.Internal, "failed to save plan")
	}

	workouts, _ := planData["workouts"].([]map[string]interface{})
	for week := int32(1); week <= req.DurationWeeks; week++ {
		weekID := uuid.New().String()
		totalDays := len(workouts)
		totalDuration := 0
		for _, w := range workouts {
			if dur, ok := w["duration"].(int); ok {
				totalDuration += dur
			}
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO training_plan_weeks (id, training_plan_id, week_number, total_training_days, total_duration_minutes)
			VALUES ($1, $2, $3, $4, $5)
		`, weekID, planID, week, totalDays, totalDuration)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to save plan weeks")
		}

		for dayIdx, w := range workouts {
			dayID := uuid.New().String()
			dayOfWeek := dayIdx % 7
			trainingType, _ := w["type"].(string)
			duration, _ := w["duration"].(int)

			_, err = tx.ExecContext(ctx, `
				INSERT INTO training_plan_days (id, week_id, day_of_week, training_date, training_type, is_rest_day, total_duration_minutes)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
			`, dayID, weekID, dayOfWeek, startDate.AddDate(0, 0, int(week-1)*7+dayOfWeek), trainingType, false, duration)
			if err != nil {
				return nil, status.Error(codes.Internal, "failed to save plan days")
			}

			exercises, _ := w["exercises"].([]string)
			for exIdx, exName := range exercises {
				_, err = tx.ExecContext(ctx, `
					INSERT INTO training_exercises (id, day_id, exercise_name, sets, reps, sort_order)
					VALUES ($1, $2, $3, $4, $5, $6)
				`, uuid.New().String(), dayID, exName, 3, 12, exIdx)
				if err != nil {
					return nil, status.Error(codes.Internal, "failed to save exercises")
				}
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, status.Error(codes.Internal, "failed to commit plan")
	}

	event := map[string]interface{}{
		"event": "plan_generated", "user_id": req.UserId, "plan_id": planID,
		"class": classificationClass, "timestamp": time.Now(),
	}
	if s.rabbitQueue != nil {
		if pubErr := s.rabbitQueue.Publish(ctx, event); pubErr != nil {
			s.log.Warn("Failed to publish event", zap.Error(pubErr))
		}
	}

	planStruct, err := structpb.NewStruct(planData)
	if err != nil {
		s.log.Error("Failed to create plan struct", zap.Error(err))
		planStruct = &structpb.Struct{}
	}

	return &pb.GeneratePlanResponse{PlanId: planID, PlanData: planStruct}, nil
}

func (s *trainingServer) GetPlan(ctx context.Context, req *pb.GetPlanRequest) (*pb.TrainingPlan, error) {
	s.log.Debug("GetPlan", zap.String("plan_id", req.PlanId))

	var planID, userID, planName, planStatus string
	var generatedAt, startDate, endDate time.Time

	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, generated_at, start_date, end_date, status
		FROM training_plans
		WHERE id = $1
	`, req.PlanId).Scan(&planID, &userID, &planName, &generatedAt, &startDate, &endDate, &planStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "plan not found")
	}
	if err != nil {
		s.log.Error("Failed to query plan", zap.Error(err), zap.String("plan_id", req.PlanId))
		return nil, status.Error(codes.Internal, "database error")
	}

	weeks, err := populatePlanWeeks(ctx, s.db, planID, s.log)
	if err != nil {
		return nil, err
	}

	weeksList, err := convertWeeksToStructpb(weeks, s.log)
	if err != nil {
		return nil, err
	}

	// Создаем Struct вручную, чтобы избежать проблем с []map[string]interface{}
	planDataOut := &structpb.Struct{
		Fields: make(map[string]*structpb.Value),
	}

	// Добавляем поля
	planDataOut.Fields["name"] = structpb.NewStringValue(planName)
	planDataOut.Fields["training_goal"] = structpb.NewStringValue("recovery")
	planDataOut.Fields["duration_weeks"] = structpb.NewNumberValue(4)
	planDataOut.Fields["weeks"] = structpb.NewListValue(weeksList)

	s.log.Info("Plan data created successfully")

	return &pb.TrainingPlan{
		Id:          planID,
		UserId:      userID,
		PlanData:    planDataOut,
		GeneratedAt: timestamppb.New(generatedAt),
		StartDate:   timestamppb.New(startDate),
		EndDate:     timestamppb.New(endDate),
		Status:      planStatus,
	}, nil
}

func (s *trainingServer) ListPlans(ctx context.Context, req *pb.ListPlansRequest) (*pb.ListPlansResponse, error) {
	if err := validator.ValidateListPlansRequest(req); err != nil {
		s.log.Warn("Invalid list plans request", zap.Error(err))
		return nil, err
	}

	s.log.Debug("ListPlans", zap.String("user_id", req.UserId))

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, name, training_goal, duration_weeks, generated_at, start_date, end_date, status
		FROM training_plans
		WHERE user_id = $1
		ORDER BY generated_at DESC
		LIMIT $2 OFFSET $3
	`, req.UserId, req.PageSize, (req.Page-1)*req.PageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.log.Warn("Failed to close rows", zap.Error(closeErr))
		}
	}()

	var plans []*pb.TrainingPlan
	for rows.Next() {
		var planID, userID, planName, planStatus string
		var trainingGoal sql.NullString
		var durationWeeks sql.NullInt32
		var generatedAt, startDate, endDate time.Time

		if err := rows.Scan(&planID, &userID, &planName, &trainingGoal, &durationWeeks, &generatedAt, &startDate, &endDate, &planStatus); err != nil {
			s.log.Error("Failed to scan plan", zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to read plan data")
		}

		planData := map[string]interface{}{
			"name":           planName,
			"training_goal":  stringValue(trainingGoal),
			"duration_weeks": int32Value(durationWeeks),
		}
		planDataStruct, err := structpb.NewStruct(planData)
		if err != nil {
			s.log.Error("Failed to create plan struct", zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to process plan data")
		}

		plans = append(plans, &pb.TrainingPlan{
			Id:          planID,
			UserId:      userID,
			PlanData:    planDataStruct,
			GeneratedAt: timestamppb.New(generatedAt),
			StartDate:   timestamppb.New(startDate),
			EndDate:     timestamppb.New(endDate),
			Status:      planStatus,
		})
	}

	if err := rows.Err(); err != nil {
		s.log.Error("Row iteration error", zap.Error(err))
		return nil, status.Error(codes.Internal, "error reading plans")
	}

	var total int32
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM training_plans WHERE user_id = $1", req.UserId).Scan(&total); err != nil {
		s.log.Warn("Failed to count plans", zap.Error(err))
	}

	return &pb.ListPlansResponse{Plans: plans, Total: total}, nil
}

func (s *trainingServer) CompleteWorkout(ctx context.Context, req *pb.CompleteWorkoutRequest) (*pb.CompleteWorkoutResponse, error) {
	s.log.Info("CompleteWorkout",
		zap.String("user_id", req.UserId),
		zap.String("plan_id", req.PlanId),
		zap.String("workout_id", req.WorkoutId),
	)

	if err := validator.ValidateCompleteWorkoutRequest(req); err != nil {
		s.log.Warn("Invalid complete workout request", zap.Error(err))
		return nil, err
	}

	feedback := sanitize.String(req.Feedback)

	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM workout_completions
					  WHERE user_id = $1 AND training_plan_id = $2 AND workout_id = $3)
	`, req.UserId, req.PlanId, req.WorkoutId).Scan(&exists)
	if err != nil {
		s.log.Error("Failed to check existing completion", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}

	if exists {
		return &pb.CompleteWorkoutResponse{Success: false}, nil
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO workout_completions (user_id, training_plan_id, workout_id, completed, completed_at, feedback)
		VALUES ($1, $2, $3, true, NOW(), $4)
	`, req.UserId, req.PlanId, req.WorkoutId, feedback)
	if err != nil {
		s.log.Error("Failed to save completion", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to save completion")
	}

	var completedCount int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM workout_completions
		WHERE user_id = $1 AND completed = true
	`, req.UserId).Scan(&completedCount)
	if err != nil {
		s.log.Error("Failed to count completions", zap.Error(err))
		completedCount = 0
	}

	var achievementID string
	switch completedCount {
	case 1:
		achievementID = "first_workout"
	case 10:
		achievementID = "ten_workouts"
	case 50:
		achievementID = "fifty_workouts"
	}

	return &pb.CompleteWorkoutResponse{Success: true, AchievementId: achievementID}, nil
}

func (s *trainingServer) GetProgress(ctx context.Context, req *pb.GetProgressRequest) (*pb.GetProgressResponse, error) {
	if err := validator.ValidateGetProgressRequest(req); err != nil {
		s.log.Warn("Invalid get progress request", zap.Error(err))
		return nil, err
	}

	s.log.Debug("GetProgress", zap.String("user_id", req.UserId))

	var totalWorkouts, completedWorkouts int32
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN completed THEN 1 END) as completed
		FROM workout_completions
		WHERE user_id = $1
	`, req.UserId).Scan(&totalWorkouts, &completedWorkouts)
	if err != nil {
		s.log.Error("Failed to get progress data", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}

	completionRate := 0.0
	if totalWorkouts > 0 {
		completionRate = float64(completedWorkouts) / float64(totalWorkouts) * 100
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT workout_id, scheduled_date, completed_at
		FROM workout_completions
		WHERE user_id = $1 AND completed = true
		ORDER BY completed_at DESC
		LIMIT 20
	`, req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.log.Warn("Failed to close rows", zap.Error(closeErr))
		}
	}()

	var history []*pb.WorkoutCompletion
	for rows.Next() {
		var wc pb.WorkoutCompletion
		var scheduledDate, completedAt time.Time
		if err := rows.Scan(&wc.WorkoutId, &scheduledDate, &completedAt); err != nil {
			s.log.Error("Failed to scan workout completion", zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to read workout data")
		}
		wc.ScheduledDate = timestamppb.New(scheduledDate)
		wc.CompletedAt = timestamppb.New(completedAt)
		history = append(history, &wc)
	}

	if err := rows.Err(); err != nil {
		s.log.Error("Row iteration error", zap.Error(err))
		return nil, status.Error(codes.Internal, "error reading workout history")
	}

	return &pb.GetProgressResponse{
		TotalWorkouts:     totalWorkouts,
		CompletedWorkouts: completedWorkouts,
		CompletionRate:    completionRate,
		History:           history,
	}, nil
}

func populatePlanWeeks(ctx context.Context, db *sql.DB, planID string, log *logger.Logger) ([]map[string]interface{}, error) {
	weeksMap := make(map[int32]map[string]interface{})

	// 1. Получаем все недели
	weekRows, err := db.QueryContext(ctx, `
		SELECT week_number, total_training_days, total_duration_minutes
		FROM training_plan_weeks
		WHERE training_plan_id = $1
		ORDER BY week_number
	`, planID)
	if err != nil {
		log.Error("Failed to query weeks", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}
	defer func() {
		if closeErr := weekRows.Close(); closeErr != nil {
			log.Warn("Failed to close week rows", zap.Error(closeErr))
		}
	}()

	for weekRows.Next() {
		var weekNum, totalDays, duration int32
		scanErr := weekRows.Scan(&weekNum, &totalDays, &duration)
		if scanErr != nil {
			log.Error("Failed to scan week", zap.Error(scanErr))
			return nil, status.Error(codes.Internal, "database error")
		}
		weeksMap[weekNum] = map[string]interface{}{
			"week_number":            weekNum,
			"total_training_days":    totalDays,
			"total_duration_minutes": duration,
			"days":                   []map[string]interface{}{},
		}
	}

	// 2. Получаем дни, для каждого дня отдельно получаем его упражнения
	dayRows, err := db.QueryContext(ctx, `
		SELECT d.id, w.week_number, d.day_of_week, d.training_date, d.training_type, d.is_rest_day, d.total_duration_minutes, d.notes
		FROM training_plan_days d
		JOIN training_plan_weeks w ON d.week_id = w.id
		WHERE w.training_plan_id = $1
		ORDER BY w.week_number, d.day_of_week
	`, planID)
	if err != nil {
		log.Error("Failed to query days", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}
	defer func() {
		if closeErr := dayRows.Close(); closeErr != nil {
			log.Warn("Failed to close day rows", zap.Error(closeErr))
		}
	}()

	for dayRows.Next() {
		var dayID string
		var weekNum, dayOfWeek, duration int32
		var trainingDate sql.NullTime
		var trainingType, notes sql.NullString
		var isRestDay bool

		scanErr := dayRows.Scan(&dayID, &weekNum, &dayOfWeek, &trainingDate, &trainingType, &isRestDay, &duration, &notes)
		if scanErr != nil {
			log.Error("Failed to scan day", zap.Error(scanErr))
			return nil, status.Error(codes.Internal, "database error")
		}

		trainingDateStr := ""
		if trainingDate.Valid {
			trainingDateStr = trainingDate.Time.Format("2006-01-02")
		}

		dayData := map[string]interface{}{
			"day_id":        dayID,
			"day_of_week":   dayOfWeek,
			"training_date": trainingDateStr,
			"training_type": trainingType.String,
			"is_rest_day":   isRestDay,
			"duration":      duration,
			"notes":         notes.String,
			"exercises":     []map[string]interface{}{},
		}

		// Получаем упражнения для этого дня
		exRows, exQueryErr := db.QueryContext(ctx, `
			SELECT exercise_name, duration_minutes, intensity, sets, reps, rest_seconds, description, sort_order
			FROM training_exercises
			WHERE day_id = $1
			ORDER BY sort_order
		`, dayID)
		if exQueryErr != nil {
			log.Error("Failed to query exercises", zap.Error(exQueryErr))
			return nil, status.Error(codes.Internal, "database error")
		}

		exercises := []map[string]interface{}{}
		for exRows.Next() {
			var exName, exDesc sql.NullString
			var exDuration, exSets, exReps, exRest, exSort sql.NullInt32
			var exIntensity sql.NullFloat64

			scanErr := exRows.Scan(&exName, &exDuration, &exIntensity, &exSets, &exReps, &exRest, &exDesc, &exSort)
			if scanErr != nil {
				log.Error("Failed to scan exercise", zap.Error(scanErr))
				if closeErr := exRows.Close(); closeErr != nil {
					log.Warn("Failed to close exercise rows", zap.Error(closeErr))
				}
				return nil, status.Error(codes.Internal, "database error")
			}

			exercise := map[string]interface{}{
				"exercise_name": exName.String,
				"duration":      int32Value(exDuration),
				"intensity":     float64Value(exIntensity),
				"sets":          int32Value(exSets),
				"reps":          int32Value(exReps),
				"rest_seconds":  int32Value(exRest),
				"description":   exDesc.String,
				"sort_order":    int32Value(exSort),
			}
			exercises = append(exercises, exercise)
		}
		if closeErr := exRows.Close(); closeErr != nil {
			log.Warn("Failed to close exercise rows", zap.Error(closeErr))
		}

		dayData["exercises"] = exercises

		// Добавляем день в неделю
		if _, exists := weeksMap[weekNum]; exists {
			days := weeksMap[weekNum]["days"].([]map[string]interface{})
			days = append(days, dayData)
			weeksMap[weekNum]["days"] = days
		}
	}

	// 3. Собираем недели в упорядоченный массив
	var weeks []map[string]interface{}
	maxWeekNum := len(weeksMap)
	if maxWeekNum > math.MaxInt32 {
		log.Error("Too many weeks in plan", zap.Int("maxWeekNum", maxWeekNum))
		return nil, status.Error(codes.Internal, "plan has too many weeks")
	}
	for i := int32(1); i <= int32(maxWeekNum); i++ {
		if w, exists := weeksMap[i]; exists {
			weeks = append(weeks, w)
		}
	}

	return weeks, nil
}

func convertWeeksToStructpb(weeks []map[string]interface{}, log *logger.Logger) (*structpb.ListValue, error) {
	weeksList := &structpb.ListValue{
		Values: make([]*structpb.Value, len(weeks)),
	}
	for i, week := range weeks {
		weekStruct := &structpb.Struct{
			Fields: make(map[string]*structpb.Value),
		}

		if val, ok := week["week_number"].(int32); ok {
			weekStruct.Fields["week_number"] = structpb.NewNumberValue(float64(val))
		} else if val, ok := week["week_number"].(int); ok {
			weekStruct.Fields["week_number"] = structpb.NewNumberValue(float64(val))
		}
		if val, ok := week["total_training_days"].(int32); ok {
			weekStruct.Fields["total_training_days"] = structpb.NewNumberValue(float64(val))
		} else if val, ok := week["total_training_days"].(int); ok {
			weekStruct.Fields["total_training_days"] = structpb.NewNumberValue(float64(val))
		}
		if val, ok := week["total_duration_minutes"].(int32); ok {
			weekStruct.Fields["total_duration_minutes"] = structpb.NewNumberValue(float64(val))
		} else if val, ok := week["total_duration_minutes"].(int); ok {
			weekStruct.Fields["total_duration_minutes"] = structpb.NewNumberValue(float64(val))
		}

		daysSlice := week["days"].([]map[string]interface{})
		daysList := &structpb.ListValue{
			Values: make([]*structpb.Value, len(daysSlice)),
		}

		for dayIdx, day := range daysSlice {
			dayStruct := convertDayToStructpb(day, log)
			daysList.Values[dayIdx] = structpb.NewStructValue(dayStruct)
		}

		weekStruct.Fields["days"] = structpb.NewListValue(daysList)
		weeksList.Values[i] = structpb.NewStructValue(weekStruct)
	}

	return weeksList, nil
}

func convertDayToStructpb(day map[string]interface{}, log *logger.Logger) *structpb.Struct {
	dayStruct := &structpb.Struct{
		Fields: make(map[string]*structpb.Value),
	}

	if val, ok := day["day_id"].(string); ok {
		dayStruct.Fields["day_id"] = structpb.NewStringValue(val)
	}
	if val, ok := day["day_of_week"].(int32); ok {
		dayStruct.Fields["day_of_week"] = structpb.NewNumberValue(float64(val))
	} else if val, ok := day["day_of_week"].(int); ok {
		dayStruct.Fields["day_of_week"] = structpb.NewNumberValue(float64(val))
	}
	if val, ok := day["training_date"].(string); ok {
		dayStruct.Fields["training_date"] = structpb.NewStringValue(val)
	}
	if val, ok := day["training_type"].(string); ok {
		dayStruct.Fields["training_type"] = structpb.NewStringValue(val)
	}
	if val, ok := day["is_rest_day"].(bool); ok {
		dayStruct.Fields["is_rest_day"] = structpb.NewBoolValue(val)
	}
	if val, ok := day["duration"].(int32); ok {
		dayStruct.Fields["duration"] = structpb.NewNumberValue(float64(val))
	} else if val, ok := day["duration"].(int); ok {
		dayStruct.Fields["duration"] = structpb.NewNumberValue(float64(val))
	}
	if val, ok := day["notes"].(string); ok {
		dayStruct.Fields["notes"] = structpb.NewStringValue(val)
	}

	exercisesSlice := day["exercises"].([]map[string]interface{})
	exercisesList := &structpb.ListValue{
		Values: make([]*structpb.Value, len(exercisesSlice)),
	}

	for exIdx, ex := range exercisesSlice {
		exStruct := &structpb.Struct{
			Fields: make(map[string]*structpb.Value),
		}
		if val, ok := ex["exercise_name"].(string); ok {
			exStruct.Fields["exercise_name"] = structpb.NewStringValue(val)
		}
		if val, ok := ex["duration"].(int32); ok {
			exStruct.Fields["duration"] = structpb.NewNumberValue(float64(val))
		} else if val, ok := ex["duration"].(int); ok {
			exStruct.Fields["duration"] = structpb.NewNumberValue(float64(val))
		}
		if val, ok := ex["intensity"].(float64); ok {
			exStruct.Fields["intensity"] = structpb.NewNumberValue(val)
		}
		if val, ok := ex["sets"].(int32); ok {
			exStruct.Fields["sets"] = structpb.NewNumberValue(float64(val))
		} else if val, ok := ex["sets"].(int); ok {
			exStruct.Fields["sets"] = structpb.NewNumberValue(float64(val))
		}
		if val, ok := ex["reps"].(int32); ok {
			exStruct.Fields["reps"] = structpb.NewNumberValue(float64(val))
		} else if val, ok := ex["reps"].(int); ok {
			exStruct.Fields["reps"] = structpb.NewNumberValue(float64(val))
		}
		if val, ok := ex["rest_seconds"].(int32); ok {
			exStruct.Fields["rest_seconds"] = structpb.NewNumberValue(float64(val))
		} else if val, ok := ex["rest_seconds"].(int); ok {
			exStruct.Fields["rest_seconds"] = structpb.NewNumberValue(float64(val))
		}
		if val, ok := ex["description"].(string); ok {
			exStruct.Fields["description"] = structpb.NewStringValue(val)
		}
		if val, ok := ex["sort_order"].(int32); ok {
			exStruct.Fields["sort_order"] = structpb.NewNumberValue(float64(val))
		} else if val, ok := ex["sort_order"].(int); ok {
			exStruct.Fields["sort_order"] = structpb.NewNumberValue(float64(val))
		}
		exercisesList.Values[exIdx] = structpb.NewStructValue(exStruct)
	}

	dayStruct.Fields["exercises"] = structpb.NewListValue(exercisesList)
	return dayStruct
}

func main() {
	log := logger.New("training-service")
	defer func() { _ = log.Sync() }()

	port := os.Getenv("TRAINING_SERVICE_PORT")
	if port == "" {
		port = "50053"
	}

	dbCfg := db.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}
	database, err := db.NewConnection(dbCfg)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer func() {
		if closeErr := database.Close(); closeErr != nil {
			log.Error("Failed to close database", zap.Error(closeErr))
		}
	}()

	rabbitURL := os.Getenv("RABBITMQ_URL")
	queueName := "training_events"
	var rabbitQueue queue.Publisher
	if rabbitURL != "" {
		rabbitQueue, err = queue.NewPublisher(rabbitURL, queueName, zap.NewNop())
		if err != nil {
			log.Warn("Failed to connect to RabbitMQ", zap.Error(err))
		} else {
			defer func() { _ = rabbitQueue.Close() }()
			log.Info("RabbitMQ connected", zap.String("queue", queueName))
		}
	}

	lc := net.ListenConfig{}
	lis, err := lc.Listen(context.Background(), "tcp", ":"+port)
	if err != nil {
		log.Fatal("Failed to listen", zap.Error(err))
	}

	s := grpc.NewServer()
	pb.RegisterTrainingServiceServer(s, &trainingServer{
		db:          database,
		log:         log,
		rabbitQueue: rabbitQueue,
	})

	log.Info("Training service starting", zap.String("port", port))
	if err := s.Serve(lis); err != nil {
		log.Fatal("Failed to serve", zap.Error(err))
	}
}

func stringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func int32Value(ni sql.NullInt32) int32 {
	if ni.Valid {
		return ni.Int32
	}
	return 0
}

func float64Value(nf sql.NullFloat64) float64 {
	if nf.Valid {
		return nf.Float64
	}
	return 0
}
