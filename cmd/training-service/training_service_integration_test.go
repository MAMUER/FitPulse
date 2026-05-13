package main

import (
	"context"
	"testing"
	"time"

	pb "github.com/MAMUER/project/api/gen/training"
	"github.com/MAMUER/project/internal/db"
	"github.com/MAMUER/project/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestTrainingService_Integration_GeneratePlan(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:15-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "testuser",
				"POSTGRES_PASSWORD": "testpass",
				"POSTGRES_DB":       "testdb",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithStartupTimeout(15 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Skipf("Skipping integration test because Docker/testcontainers is unavailable: %v", err)
	}
	defer func() { assert.NoError(t, pgContainer.Terminate(ctx)) }()

	// Get connection details
	host, err := pgContainer.Host(ctx)
	require.NoError(t, err)

	port, err := pgContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Connect to database
	dbCfg := db.Config{
		Host:     host,
		Port:     port.Port(),
		User:     "testuser",
		Password: "testpass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	database, err := db.NewConnection(dbCfg)
	if err != nil {
		t.Skip("PostgreSQL not available")
	}
	defer func() { _ = database.Close() }()

	// Run migrations (simplified - just create tables)
	_, err = database.ExecContext(context.Background(), `
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
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			completed_at TIMESTAMP WITH TIME ZONE
		);

		CREATE TABLE IF NOT EXISTS training_plan_weeks (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			training_plan_id UUID REFERENCES training_plans(id),
			week_number INTEGER NOT NULL,
			total_training_days INTEGER,
			total_duration_minutes INTEGER
		);

		CREATE TABLE IF NOT EXISTS training_plan_days (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			week_id UUID REFERENCES training_plan_weeks(id),
			day_of_week INTEGER NOT NULL,
			training_date TIMESTAMP WITH TIME ZONE,
			training_type VARCHAR(255),
			is_rest_day BOOLEAN DEFAULT false,
			total_duration_minutes INTEGER
		);

		CREATE TABLE IF NOT EXISTS training_exercises (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			day_id UUID REFERENCES training_plan_days(id),
			exercise_name VARCHAR(255) NOT NULL,
			sets INTEGER,
			reps INTEGER,
			sort_order INTEGER
		);
	`)
	require.NoError(t, err)

	// Create training server
	log := logger.New("test-integration")

	// Mock queue publisher
	publisher := &mockPublisher{}

	server := &trainingServer{
		db:          database,
		log:         log,
		rabbitQueue: publisher,
	}

	// Test GeneratePlan
	req := &pb.GeneratePlanRequest{
		UserId:              "integration-test-user",
		ClassificationClass: "endurance_e1e2",
		DurationWeeks:       2,
		AvailableDays:       []int32{1, 3, 5},
	}

	resp, err := server.GeneratePlan(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.PlanId)

	// Verify plan was created in database
	var count int
	err = database.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM training_plans WHERE user_id = $1", req.UserId).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify weeks were created (2 weeks)
	err = database.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM training_plan_weeks WHERE training_plan_id = $1", resp.PlanId).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Verify days were created (2 weeks * 3 days = 6 days)
	err = database.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM training_plan_days WHERE week_id IN (SELECT id FROM training_plan_weeks WHERE training_plan_id = $1)", resp.PlanId).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 6, count)

	// Test ListPlans
	listReq := &pb.ListPlansRequest{
		UserId:   req.UserId,
		PageSize: 10,
		Page:     1,
	}

	listResp, err := server.ListPlans(ctx, listReq)
	require.NoError(t, err)
	require.NotNil(t, listResp)
	assert.Len(t, listResp.Plans, 1)
	assert.Equal(t, resp.PlanId, listResp.Plans[0].Id)
}

func TestTrainingService_Integration_GetProgress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:15-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "testuser",
				"POSTGRES_PASSWORD": "testpass",
				"POSTGRES_DB":       "testdb",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithStartupTimeout(15 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Skipf("Skipping integration test because Docker/testcontainers is unavailable: %v", err)
	}
	defer func() { assert.NoError(t, pgContainer.Terminate(ctx)) }()

	// Get connection details
	host, err := pgContainer.Host(ctx)
	require.NoError(t, err)

	port, err := pgContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Connect to database
	dbCfg := db.Config{
		Host:     host,
		Port:     port.Port(),
		User:     "testuser",
		Password: "testpass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	database, err := db.NewConnection(dbCfg)
	if err != nil {
		t.Skip("PostgreSQL not available")
	}
	defer func() { _ = database.Close() }()

	// Create tables and sample data
	_, err = database.ExecContext(context.Background(), `
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
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			completed_at TIMESTAMP WITH TIME ZONE
		);

		CREATE TABLE IF NOT EXISTS training_plan_weeks (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			training_plan_id UUID REFERENCES training_plans(id),
			week_number INTEGER NOT NULL,
			total_training_days INTEGER,
			total_duration_minutes INTEGER,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS training_plan_days (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			week_id UUID REFERENCES training_plan_weeks(id),
			day_of_week INTEGER NOT NULL,
			training_date TIMESTAMP WITH TIME ZONE,
			training_type VARCHAR(255),
			is_rest_day BOOLEAN DEFAULT false,
			total_duration_minutes INTEGER,
			notes TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			completed_at TIMESTAMP WITH TIME ZONE
		);

		CREATE TABLE IF NOT EXISTS training_plan_exercises (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			day_id UUID REFERENCES training_plan_days(id),
			exercise_name VARCHAR(255) NOT NULL,
			duration_minutes INTEGER,
			intensity FLOAT,
			sets INTEGER,
			reps INTEGER,
			rest_seconds INTEGER,
			description TEXT,
			sort_order INTEGER,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS workout_completions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id VARCHAR(255) NOT NULL,
			training_plan_id UUID,
			workout_id VARCHAR(255),
			completed BOOLEAN DEFAULT false,
			completed_at TIMESTAMP WITH TIME ZONE,
			scheduled_date TIMESTAMP WITH TIME ZONE,
			feedback TEXT
		);

		-- Insert test data
		INSERT INTO training_plans (id, user_id, classification_class, duration_weeks, status)
		VALUES ('550e8400-e29b-41d4-a716-446655440000', 'test-user-progress', 'endurance_e1e2', 4, 'active');

		INSERT INTO workout_completions (id, user_id, training_plan_id, workout_id, completed, completed_at, scheduled_date)
		VALUES
			('aaaa0000-0000-0000-0000-000000000001', 'test-user-progress', '550e8400-e29b-41d4-a716-446655440000', 'workout-1', true, '2026-05-01T10:00:00Z', '2026-05-01T08:00:00Z'),
			('aaaa0000-0000-0000-0000-000000000002', 'test-user-progress', '550e8400-e29b-41d4-a716-446655440000', 'workout-2', true, '2026-05-02T10:00:00Z', '2026-05-02T08:00:00Z'),
			('aaaa0000-0000-0000-0000-000000000003', 'test-user-progress', '550e8400-e29b-41d4-a716-446655440000', 'workout-3', false, NULL, '2026-05-03T08:00:00Z'),
			('aaaa0000-0000-0000-0000-000000000004', 'test-user-progress', '550e8400-e29b-41d4-a716-446655440000', 'workout-4', false, NULL, '2026-05-04T08:00:00Z');
	`)
	require.NoError(t, err)

	// Create training server
	log := logger.New("test-integration")

	server := &trainingServer{
		db:          database,
		log:         log,
		rabbitQueue: &mockPublisher{},
	}

	// Test GetProgress
	req := &pb.GetProgressRequest{
		UserId: "test-user-progress",
	}

	progressResp, err := server.GetProgress(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, progressResp)

	// Should have some completion history
	assert.NotEmpty(t, progressResp.History)

	// Check completion rate (2 out of 4 workouts completed = 50%)
	assert.Equal(t, float64(50.0), progressResp.CompletionRate)
	assert.Equal(t, int32(4), progressResp.TotalWorkouts)
	assert.Equal(t, int32(2), progressResp.CompletedWorkouts)

	// Check history contains completed workouts
	completedWorkouts := 0
	for _, completion := range progressResp.History {
		if completion.CompletedAt != nil {
			completedWorkouts++
		}
	}
	assert.Equal(t, 2, completedWorkouts)
}
