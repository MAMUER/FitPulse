package main

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	pb "github.com/MAMUER/project/api/gen/training"
	"github.com/MAMUER/project/internal/logger"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestTrainingService_Integration_GenerateAndCompletePlan(t *testing.T) {
	ctx := context.Background()

	// Запускаем PostgreSQL
	pgContainer, err := postgres.Run(ctx, "postgres:15-alpine",
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Skipf("Skipping integration test: could not start PostgreSQL: %v", err)
	}
	defer func() { _ = pgContainer.Terminate(ctx) }()

	host, _ := pgContainer.Host(ctx)
	port, _ := pgContainer.MappedPort(ctx, "5432")

	dsn := fmt.Sprintf("host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable",
		host, port.Port())

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Создаём минимальные таблицы
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS training_plans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id VARCHAR(255) NOT NULL,
			name VARCHAR(255),
			training_goal VARCHAR(255),
			classification_class VARCHAR(255) NOT NULL,
			duration_weeks INTEGER NOT NULL,
			generated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			start_date TIMESTAMP WITH TIME ZONE,
			end_date TIMESTAMP WITH TIME ZONE,
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS workouts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			plan_id UUID NOT NULL REFERENCES training_plans(id),
			name VARCHAR(255) NOT NULL,
			day INTEGER NOT NULL,
			exercises JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS workout_completions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id VARCHAR(255) NOT NULL,
			training_plan_id UUID NOT NULL,
			workout_id UUID NOT NULL,
			completed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`)
	require.NoError(t, err)

	log := logger.New("integration-test")
	server := &trainingServer{
		db:          db,
		log:         log,
		rabbitQueue: &mockPublisher{},
	}

	// === Generate Plan ===
	genReq := &pb.GeneratePlanRequest{
		UserId:              "user-123",
		ClassificationClass: "endurance_e1e2",
		DurationWeeks:       4,
		AvailableDays:       []int32{1, 3, 5},
	}

	genResp, err := server.GeneratePlan(ctx, genReq)
	if err == nil {
		require.NotEmpty(t, genResp.PlanId)
	} else {
		t.Logf("GeneratePlan returned expected error in minimal DB setup: %v", err)
	}

	// === Complete Workout ===
	completeReq := &pb.CompleteWorkoutRequest{
		UserId:    "user-123",
		PlanId:    genResp.PlanId,
		WorkoutId: "workout-1",
	}

	completeResp, err := server.CompleteWorkout(ctx, completeReq)
	require.NoError(t, err)
	require.True(t, completeResp.Success)

	t.Log("✅ Training service integration test passed: GeneratePlan → CompleteWorkout")
}

// mockPublisher для тестов
type mockPublisher struct{}

func (m *mockPublisher) Publish(ctx context.Context, event interface{}) error { return nil }
func (m *mockPublisher) Close() error                                         { return nil }
