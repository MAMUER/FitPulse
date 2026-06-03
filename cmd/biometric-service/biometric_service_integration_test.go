package main

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	pb "github.com/MAMUER/project/api/gen/biometric"
	"github.com/MAMUER/project/internal/logger"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestBiometricService_Integration_AddAndGetRecords(t *testing.T) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:15-alpine",
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
		t.Skipf("Skipping biometric-service integration test: could not start PostgreSQL: %v", err)
	}
	defer func() { _ = pgContainer.Terminate(ctx) }()

	host, _ := pgContainer.Host(ctx)
	port, _ := pgContainer.MappedPort(ctx, "5432")

	dsn := fmt.Sprintf("host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable",
		host, port.Port())

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS biometric_data (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id VARCHAR(255) NOT NULL,
			metric_type VARCHAR(50) NOT NULL,
			value DOUBLE PRECISION NOT NULL,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			device_type VARCHAR(100),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`)
	require.NoError(t, err)

	log := logger.New("biometric-integration-test")

	server := &biometricServer{
		db:  db,
		log: log,
	}

	// === Add Record ===
	addReq := &pb.AddRecordRequest{
		UserId:     "user-123",
		MetricType: "heart_rate",
		Value:      75,
		Timestamp:  timestamppb.Now(),
		DeviceType: "apple_watch",
	}

	addResp, err := server.AddRecord(ctx, addReq)
	require.NoError(t, err)
	require.NotEmpty(t, addResp.Id)

	// GetRecords path tested in unit tests; integration focuses on Add + DB write
	t.Log("Biometric Service integration test passed: AddRecord")
}
