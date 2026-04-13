package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestRequirePrivilege_MissingUserID(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	core, observed := observer.New(zapcore.WarnLevel)
	log := zap.New(core)

	handler := RequirePrivilege(db, "admin", log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	// Context without UserIDKey
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "not found")
	// No DB query should be made
	require.NoError(t, mock.ExpectationsWereMet())
	assert.Zero(t, observed.Len(), "no logs expected for missing userID")
}

func TestRequirePrivilege_UserNotFound(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	core, observed := observer.New(zapcore.WarnLevel)
	log := zap.New(core)

	mock.ExpectQuery("SELECT role FROM users WHERE id = $1").
		WithArgs("user-1").
		WillReturnError(sql.ErrNoRows)

	handler := RequirePrivilege(db, "admin", log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	ctx := context.WithValue(req.Context(), UserIDKey, "user-1")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "not found")
	require.NoError(t, mock.ExpectationsWereMet())

	logs := observed.All()
	require.Len(t, logs, 1)
	assert.Equal(t, "User not found during privilege check", logs[0].Message)
	assert.Equal(t, "user-1", logs[0].ContextMap()["user_id"])
}

func TestRequirePrivilege_DBError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	core, observed := observer.New(zapcore.ErrorLevel)
	log := zap.New(core)

	mock.ExpectQuery("SELECT role FROM users WHERE id = $1").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	handler := RequirePrivilege(db, "admin", log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	ctx := context.WithValue(req.Context(), UserIDKey, "user-1")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "internal error")
	require.NoError(t, mock.ExpectationsWereMet())

	logs := observed.All()
	require.Len(t, logs, 1)
	assert.Equal(t, "Database error during privilege check", logs[0].Message)
}

func TestRequirePrivilege_RoleMismatch(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	core, observed := observer.New(zapcore.WarnLevel)
	log := zap.New(core)

	mock.ExpectQuery("SELECT role FROM users WHERE id = $1").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow("viewer"))

	handler := RequirePrivilege(db, "admin", log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	ctx := context.WithValue(req.Context(), UserIDKey, "user-1")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "not found")
	require.NoError(t, mock.ExpectationsWereMet())

	logs := observed.All()
	require.Len(t, logs, 1)
	assert.Equal(t, "Insufficient privileges", logs[0].Message)
	assert.Equal(t, "user-1", logs[0].ContextMap()["user_id"])
	assert.Equal(t, "admin", logs[0].ContextMap()["required_role"])
	assert.Equal(t, "viewer", logs[0].ContextMap()["actual_role"])
}

func TestRequirePrivilege_RoleMatch(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	log, _ := zap.NewDevelopment()

	mock.ExpectQuery("SELECT role FROM users WHERE id = $1").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow("admin"))

	nextCalled := false
	handler := RequirePrivilege(db, "admin", log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		role, ok := GetConfirmedPrivilege(r.Context())
		assert.True(t, ok)
		assert.Equal(t, "admin", role)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	ctx := context.WithValue(req.Context(), UserIDKey, "user-1")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, nextCalled, "next handler should be called")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRequirePrivilege_ContextCancelled(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	core, observed := observer.New(zapcore.ErrorLevel)
	log := zap.New(core)

	// When context is already cancelled, QueryRowContext returns ctx.Err()
	// without executing the query, so we don't set a mock expectation.

	handler := RequirePrivilege(db, "admin", log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	ctx, cancel := context.WithCancel(req.Context())
	ctx = context.WithValue(ctx, UserIDKey, "user-1")
	cancel() // Cancel before execution
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	// No DB query executed because context was already cancelled
	require.NoError(t, mock.ExpectationsWereMet())

	logs := observed.All()
	require.Len(t, logs, 1)
	assert.Equal(t, "Database error during privilege check", logs[0].Message)
}

func TestGetConfirmedPrivilege(t *testing.T) {
	t.Run("returns role and true when privilege is set", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), PrivilegeKey{}, "admin")
		role, ok := GetConfirmedPrivilege(ctx)
		assert.True(t, ok)
		assert.Equal(t, "admin", role)
	})

	t.Run("returns empty and false when privilege is not set", func(t *testing.T) {
		ctx := context.Background()
		role, ok := GetConfirmedPrivilege(ctx)
		assert.False(t, ok)
		assert.Empty(t, role)
	})

	t.Run("returns empty and false when value is wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), PrivilegeKey{}, 123)
		role, ok := GetConfirmedPrivilege(ctx)
		assert.False(t, ok)
		assert.Empty(t, role)
	})

	t.Run("returns empty and false for nil context value", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), PrivilegeKey{}, nil)
		role, ok := GetConfirmedPrivilege(ctx)
		assert.False(t, ok)
		assert.Empty(t, role)
	})
}
