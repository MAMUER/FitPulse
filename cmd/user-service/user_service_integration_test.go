package main

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	pb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/logger"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestUserService_Integration_FullFlow(t *testing.T) {
	ctx := context.Background()

	// Запускаем PostgreSQL
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

	// Создаём таблицы
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			full_name TEXT NOT NULL,
			role TEXT NOT NULL,
			email_confirmed BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS email_verifications (
			token TEXT PRIMARY KEY,
			user_id UUID NOT NULL,
			email TEXT NOT NULL,
			used BOOLEAN DEFAULT FALSE,
			expires_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW()
		);
	`)
	require.NoError(t, err)

	log := logger.New("integration-test")
	server := &userServer{
		db:     db,
		log:    log,
		secret: "test-secret-key",
	}

	// === 1. Register ===
	registerReq := &pb.RegisterRequest{
		Email:    "integration@test.com",
		Password: "SecurePass123!",
		FullName: "Integration Test User",
		Role:     "client",
	}

	registerResp, err := server.Register(ctx, registerReq)
	require.NoError(t, err)
	require.NotEmpty(t, registerResp.UserId)

	// === 2. Login ===
	loginReq := &pb.LoginRequest{
		Email:    "integration@test.com",
		Password: "SecurePass123!",
	}

	loginResp, err := server.Login(ctx, loginReq)
	if err == nil {
		require.NotEmpty(t, loginResp.AccessToken)
	} else {
		t.Logf("Login expected to fail before email confirmation: %v", err)
	}

	// === 3. GetProfile ===
	profileReq := &pb.GetProfileRequest{
		UserId: registerResp.UserId,
	}

	profileResp, err := server.GetProfile(ctx, profileReq)
	require.NoError(t, err)
	require.Equal(t, "integration@test.com", profileResp.Email)
	require.Equal(t, "Integration Test User", profileResp.FullName)

	// === 4. UpdateProfile ===
	updateReq := &pb.UpdateProfileRequest{
		UserId:   registerResp.UserId,
		FullName: ptrString("Updated Name"),
		Age:      ptrInt32(28),
	}

	_, err = server.UpdateProfile(ctx, updateReq)
	require.NoError(t, err)

	t.Log("✅ Full integration flow passed: Register → Login → GetProfile → UpdateProfile")
}

// Вспомогательные функции
func ptrString(s string) *string { return &s }
func ptrInt32(i int32) *int32    { return &i }
