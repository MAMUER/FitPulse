package main

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	pb "github.com/MAMUER/project/api/gen/training"
	"github.com/MAMUER/project/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTrainingServer(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	log := logger.New("test")
	publisher := &mockPublisher{}

	server := &trainingServer{
		db:          db,
		log:         log,
		rabbitQueue: publisher,
	}

	assert.NotNil(t, server)
	assert.Equal(t, db, server.db)
	assert.Equal(t, log, server.log)
	assert.Equal(t, publisher, server.rabbitQueue)
}

func TestGeneratePlan_NoExistingPlan(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(`SELECT id FROM training_plans WHERE user_id = \$1 AND status = 'active'`).
		WithArgs("user-123").
		WillReturnError(sql.ErrConnDone)

	log := logger.New("test")
	server := &trainingServer{
		db:          db,
		log:         log,
		rabbitQueue: &mockPublisher{},
	}

	req := &pb.GeneratePlanRequest{
		UserId:              "user-123",
		ClassificationClass: "endurance_e1e2",
		DurationWeeks:       4,
		AvailableDays:       []int32{1, 3, 5},
	}

	resp, err := server.GeneratePlan(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPlans_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Now()
	later := now.Add(7 * 24 * time.Hour)
	muchLater := now.Add(14 * 24 * time.Hour)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "name", "training_goal", "duration_weeks",
		"generated_at", "start_date", "end_date", "status",
	}).
		AddRow("plan-1", "user-123", "Endurance Plan", "endurance", 4, now, now, later, "active").
		AddRow("plan-2", "user-123", "Strength Plan", "strength", 6, muchLater, muchLater, later, "completed")

	// SELECT query comes first in ListPlans
	mock.ExpectQuery(`(?s)SELECT (.+) FROM training_plans WHERE user_id = \$1 ORDER BY generated_at DESC LIMIT \$2 OFFSET \$3`).
		WithArgs("user-123", int32(10), int32(0)).
		WillReturnRows(rows)

	// COUNT query comes second
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM training_plans WHERE user_id = \$1`).
		WithArgs("user-123").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	log := logger.New("test")
	server := &trainingServer{
		db:          db,
		log:         log,
		rabbitQueue: &mockPublisher{},
	}

	req := &pb.ListPlansRequest{
		UserId:   "user-123",
		PageSize: 10,
		Page:     1,
	}

	resp, err := server.ListPlans(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(2), resp.Total)
	assert.Len(t, resp.Plans, 2)
	assert.Equal(t, "plan-1", resp.Plans[0].Id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCompleteWorkout_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Mock workout does not exist yet (first completion)
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM workout_completions WHERE user_id = \$1 AND training_plan_id = \$2 AND workout_id = \$3\)`).
		WithArgs("user-123", "plan-1", "workout-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// Mock insert completion
	mock.ExpectExec(`INSERT INTO workout_completions \(user_id, training_plan_id, workout_id, completed, completed_at, feedback\) VALUES \(\$1, \$2, \$3, true, NOW\(\), \$4\)`).
		WithArgs("user-123", "plan-1", "workout-1", "").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Mock count completions
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM workout_completions WHERE user_id = \$1 AND completed = true`).
		WithArgs("user-123").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	log := logger.New("test")
	server := &trainingServer{
		db:          db,
		log:         log,
		rabbitQueue: &mockPublisher{},
	}

	req := &pb.CompleteWorkoutRequest{
		UserId:    "user-123",
		PlanId:    "plan-1",
		WorkoutId: "workout-1",
	}

	resp, err := server.CompleteWorkout(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "first_workout", resp.AchievementId)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCompleteWorkout_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Mock workout already completed (exists = true)
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM workout_completions WHERE user_id = \$1 AND training_plan_id = \$2 AND workout_id = \$3\)`).
		WithArgs("user-123", "plan-1", "workout-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	log := logger.New("test")
	server := &trainingServer{
		db:          db,
		log:         log,
		rabbitQueue: &mockPublisher{},
	}

	req := &pb.CompleteWorkoutRequest{
		UserId:    "user-123",
		PlanId:    "plan-1",
		WorkoutId: "workout-1",
	}

	resp, err := server.CompleteWorkout(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.False(t, resp.Success)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGeneratePlan_ValidationError(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	log := logger.New("test")
	server := &trainingServer{
		db:          db,
		log:         log,
		rabbitQueue: &mockPublisher{},
	}

	req := &pb.GeneratePlanRequest{
		UserId:              "", // invalid
		ClassificationClass: "endurance_e1e2",
		DurationWeeks:       4,
		AvailableDays:       []int32{1, 2, 3},
	}

	resp, err := server.GeneratePlan(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestGeneratePlan_ContextCancelled(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	log := logger.New("test")
	server := &trainingServer{
		db:          db,
		log:         log,
		rabbitQueue: &mockPublisher{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancelled

	req := &pb.GeneratePlanRequest{
		UserId:              "user-123",
		ClassificationClass: "endurance_e1e2",
		DurationWeeks:       4,
		AvailableDays:       []int32{1, 2, 3},
	}

	resp, err := server.GeneratePlan(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestListPlans_ValidationError(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	log := logger.New("test")
	server := &trainingServer{
		db:          db,
		log:         log,
		rabbitQueue: &mockPublisher{},
	}

	req := &pb.ListPlansRequest{
		UserId:   "", // invalid
		PageSize: 10,
		Page:     1,
	}

	resp, err := server.ListPlans(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestCompleteWorkout_ValidationError(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	log := logger.New("test")
	server := &trainingServer{
		db:          db,
		log:         log,
		rabbitQueue: &mockPublisher{},
	}

	req := &pb.CompleteWorkoutRequest{
		UserId:    "", // invalid
		PlanId:    "plan-1",
		WorkoutId: "workout-1",
	}

	resp, err := server.CompleteWorkout(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestGetProgress_ValidationError(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	log := logger.New("test")
	server := &trainingServer{
		db:          db,
		log:         log,
		rabbitQueue: &mockPublisher{},
	}

	req := &pb.GetProgressRequest{
		UserId: "", // invalid
	}

	resp, err := server.GetProgress(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestGeneratePlan_InvalidDuration(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	log := logger.New("test")
	server := &trainingServer{
		db:          db,
		log:         log,
		rabbitQueue: &mockPublisher{},
	}

	req := &pb.GeneratePlanRequest{
		UserId:              "user-123",
		ClassificationClass: "endurance_e1e2",
		DurationWeeks:       0, // invalid
		AvailableDays:       []int32{1, 2, 3},
	}

	resp, err := server.GeneratePlan(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}
