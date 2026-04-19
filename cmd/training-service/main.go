package main

import (
	"context"
	"database/sql"
	"errors"
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

	classificationClass := sanitize.String(req.ClassificationClass)
	planID := uuid.New().String()

	planData := map[string]interface{}{
		"name":       "Персонализированная программа",
		"class":      classificationClass,
		"confidence": req.Confidence,
		"weeks":      req.DurationWeeks,
		"schedule":   req.AvailableDays,
		"workouts": []map[string]interface{}{
			{"day": 1, "type": "cardio", "duration": 30, "intensity": "medium", "exercises": []string{"бег", "велосипед"}},
			{"day": 3, "type": "strength", "duration": 45, "intensity": "high", "exercises": []string{"приседания", "отжимания", "тяга"}},
			{"day": 5, "type": "recovery", "duration": 20, "intensity": "low", "exercises": []string{"растяжка", "йога"}},
		},
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
		return nil, status.Error(codes.Internal, "database error")
	}

	// Собираем план из нормализованных таблиц в JSON
	rows, err := s.db.QueryContext(ctx, `
		SELECT w.week_number, w.total_training_days, w.total_duration_minutes,
			   d.id as day_id, d.day_of_week, d.training_date, d.training_type, d.is_rest_day, d.total_duration_minutes, d.notes,
			   e.id as exercise_id, e.exercise_name, e.duration_minutes, e.intensity, e.sets, e.reps, e.rest_seconds, e.description, e.sort_order
		FROM training_plan_weeks w
		LEFT JOIN training_plan_days d ON d.week_id = w.id
		LEFT JOIN training_exercises e ON e.day_id = d.id
		WHERE w.training_plan_id = $1
		ORDER BY w.week_number, d.day_of_week, e.sort_order
	`, req.PlanId)
	if err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.log.Warn("Failed to close rows", zap.Error(closeErr))
		}
	}()

	weeks := make(map[int32]map[string]interface{})
	dayExercises := make(map[string][]map[string]interface{})

	for rows.Next() {
		var weekNum, totalDays, totalDurMinutes int32
		var dayID sql.NullString
		var dayOfWeek int32
		var trainingDate sql.NullTime
		var trainingType, notes sql.NullString
		var isRestDay bool
		var dayDuration sql.NullInt32
		var exerciseID sql.NullString
		var exerciseName sql.NullString
		var exDuration, exSets, exReps, exRest, exSortOrder sql.NullInt32
		var exIntensity sql.NullFloat64
		var exDescription sql.NullString

		if scanErr := rows.Scan(&weekNum, &totalDays, &totalDurMinutes,
			&dayID, &dayOfWeek, &trainingDate, &trainingType, &isRestDay, &dayDuration, &notes,
			&exerciseID, &exerciseName, &exDuration, &exIntensity, &exSets, &exReps, &exRest, &exDescription, &exSortOrder); scanErr != nil {
			s.log.Error("Failed to scan plan data", zap.Error(scanErr))
			return nil, status.Error(codes.Internal, "database error")
		}

		if _, exists := weeks[weekNum]; !exists {
			weeks[weekNum] = map[string]interface{}{
				"week_number":            weekNum,
				"total_training_days":    totalDays,
				"total_duration_minutes": totalDurMinutes,
				"days":                   []map[string]interface{}{},
			}
		}

		if dayID.Valid && dayID.String != "" {
			dayKey := dayID.String
			if _, exists := dayExercises[dayKey]; !exists {
				dayData := map[string]interface{}{
					"day_id":        dayID.String,
					"day_of_week":   dayOfWeek,
					"training_type": stringValue(trainingType),
					"is_rest_day":   isRestDay,
					"duration":      int32Value(dayDuration),
					"notes":         stringValue(notes),
					"exercises":     []map[string]interface{}{},
				}
				if trainingDate.Valid {
					dayData["training_date"] = trainingDate.Time.Format("2006-01-02")
				}
				dayExercises[dayKey] = []map[string]interface{}{}
				weeks[weekNum]["days"] = append(weeks[weekNum]["days"].([]map[string]interface{}), dayData)
			}

			if exerciseID.Valid && exerciseID.String != "" {
				dayExercises[dayKey] = append(dayExercises[dayKey], map[string]interface{}{
					"exercise_name": stringValue(exerciseName),
					"duration":      int32Value(exDuration),
					"intensity":     float64Value(exIntensity),
					"sets":          int32Value(exSets),
					"reps":          int32Value(exReps),
					"rest_seconds":  int32Value(exRest),
					"description":   stringValue(exDescription),
					"sort_order":    int32Value(exSortOrder),
				})
			}
		}
	}

	// Вставляем упражнения в дни
	for _, weekData := range weeks {
		days := weekData["days"].([]map[string]interface{})
		for dayIdx := range days {
			dayKey := days[dayIdx]["day_id"].(string)
			days[dayIdx]["exercises"] = dayExercises[dayKey]
		}
	}

	weekList := make([]map[string]interface{}, 0, len(weeks))
	for _, w := range weeks {
		weekList = append(weekList, w)
	}

	planData := map[string]interface{}{
		"name":  planName,
		"weeks": weekList,
	}

	planDataOut, err := structpb.NewStruct(planData)
	if err != nil {
		s.log.Error("Failed to create plan struct", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to process plan data")
	}

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
	`, req.UserId, req.PageSize, req.Page*req.PageSize)
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

// === Helpers ===

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
