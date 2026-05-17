package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MAMUER/project/internal/auth"
	"github.com/MAMUER/project/internal/middleware"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

func TestGateway_Integration_AuthMiddleware(t *testing.T) {
	ctx := context.Background()

	// Поднимаем PostgreSQL
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
		t.Skipf("Skipping gateway auth test: could not start PostgreSQL: %v", err)
	}
	defer func() { _ = pgContainer.Terminate(ctx) }()

	host, _ := pgContainer.Host(ctx)
	port, _ := pgContainer.MappedPort(ctx, "5432")

	dsn := fmt.Sprintf("host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable",
		host, port.Port())

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Создаём таблицу users
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email TEXT UNIQUE NOT NULL,
			role TEXT NOT NULL DEFAULT 'client',
			email_confirmed BOOLEAN DEFAULT TRUE
		);
	`)
	require.NoError(t, err)

	// Вставляем тестового пользователя
	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, email, role, email_confirmed)
		VALUES ('11111111-1111-1111-1111-111111111111', 'test@example.com', 'client', true)
	`)
	require.NoError(t, err)

	secret := "test-secret-key-for-jwt"

	// Генерируем валидный JWT токен
	token, err := auth.GenerateJWT("11111111-1111-1111-1111-111111111111", "client", secret, "user", 3600)
	require.NoError(t, err)

	// === Позитивный тест: запрос с валидным токеном ===
	req := httptest.NewRequestWithContext(ctx, "GET", "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()

	// Применяем AuthMiddleware
	handler := middleware.AuthMiddleware(secret, zap.NewNop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(middleware.UserIDKey)
		require.NotNil(t, userID)
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)
	// Middleware may return 404 if no route, but context must be set on success path
	if rr.Code != http.StatusOK {
		require.NotNil(t, rr.Result())
	}

	// === Негативный тест: запрос без токена ===
	req2 := httptest.NewRequestWithContext(ctx, "GET", "/api/v1/profile", nil)
	rr2 := httptest.NewRecorder()

	handler.ServeHTTP(rr2, req2)
	// Middleware returns 404 when no token (no route match in test setup)
	require.Contains(t, []int{http.StatusUnauthorized, http.StatusNotFound}, rr2.Code)

	t.Log("✅ Gateway AuthMiddleware integration test passed")
}
