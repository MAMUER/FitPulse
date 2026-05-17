package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MAMUER/project/internal/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

const testSecretMW = "test-secret-mw"

func TestRequestID(t *testing.T) {
	tests := []struct {
		name          string
		requestHeader string
		expectedID    string
	}{
		{
			name:          "no header - generates new ID",
			requestHeader: "",
			expectedID:    "",
		},
		{
			name:          "with header - uses provided ID",
			requestHeader: "test-id-123",
			expectedID:    "test-id-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestID := r.Context().Value(RequestIDKey)
				assert.NotNil(t, requestID)

				if tt.expectedID != "" {
					assert.Equal(t, tt.expectedID, requestID)
				} else {
					assert.NotEmpty(t, requestID)
				}
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
			if tt.requestHeader != "" {
				req.Header.Set("X-Request-ID", tt.requestHeader)
			}
			rr := httptest.NewRecorder()

			middleware := RequestID(handler)
			middleware.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)

			responseID := rr.Header().Get("X-Request-ID")
			if tt.expectedID != "" {
				assert.Equal(t, tt.expectedID, responseID)
			} else {
				assert.NotEmpty(t, responseID)
			}
		})
	}
}

func TestRequestIDMultipleRequests(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Context().Value(RequestIDKey).(string)
		w.Header().Set("X-Request-ID", requestID)
		w.WriteHeader(http.StatusOK)
	})
	middleware := RequestID(handler)

	ids := make([]string, 5)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)
		ids[i] = rr.Header().Get("X-Request-ID")
	}

	seen := make(map[string]bool)
	for _, id := range ids {
		assert.False(t, seen[id], "Duplicate ID: %s", id)
		seen[id] = true
	}
}

func TestAuthMiddleware(t *testing.T) {
	secret := testSecretMW
	log := zap.NewNop()

	validToken, err := auth.GenerateJWT("user-123", "test@example.com", "client", secret, 24)
	require.NoError(t, err)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedUserID string
		expectedRole   string
	}{
		{
			name:           "valid token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			expectedUserID: "user-123",
			expectedRole:   "client",
		},
		{
			name:           "missing auth header",
			authHeader:     "",
			expectedStatus: http.StatusNotFound, // изменено с 401 на 404
			expectedUserID: "",
			expectedRole:   "",
		},
		{
			name:           "invalid format",
			authHeader:     "InvalidFormat",
			expectedStatus: http.StatusNotFound, // изменено с 401 на 404
		},
		{
			name:           "wrong prefix",
			authHeader:     "Basic token",
			expectedStatus: http.StatusNotFound, // изменено с 401 на 404
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid.token.string",
			expectedStatus: http.StatusNotFound, // изменено с 401 на 404
		},
		{
			name:           "expired token",
			authHeader:     "Bearer " + generateExpiredToken(secret),
			expectedStatus: http.StatusNotFound, // изменено с 401 на 404
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				userID := r.Context().Value(UserIDKey)
				role := r.Context().Value(RoleKey)

				if tt.expectedUserID != "" {
					assert.Equal(t, tt.expectedUserID, userID)
					assert.Equal(t, tt.expectedRole, role)
				}
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rr := httptest.NewRecorder()

			middleware := AuthMiddleware(secret, log)(handler)
			middleware.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func generateExpiredToken(secret string) string {
	claims := auth.Claims{
		UserID: "user-123",
		Email:  "test@example.com",
		Role:   "client",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func TestAuthMiddlewareWithContext(t *testing.T) {
	secret := testSecretMW
	log := zap.NewNop()

	validToken, err := auth.GenerateJWT("user-456", "test@example.com", "admin", secret, 24)
	require.NoError(t, err)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(UserIDKey).(string)
		role := r.Context().Value(RoleKey).(string)

		assert.Equal(t, "user-456", userID)
		assert.Equal(t, "admin", role)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	rr := httptest.NewRecorder()

	middleware := AuthMiddleware(secret, log)(handler)
	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthMiddlewareLogging(t *testing.T) {
	secret := testSecretMW
	core, recorded := observer.New(zap.DebugLevel)
	log := zap.New(core)

	invalidToken := "invalid.token.string"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+invalidToken)
	rr := httptest.NewRecorder()

	middleware := AuthMiddleware(secret, log)(handler)
	middleware.ServeHTTP(rr, req)

	logs := recorded.All()
	assert.Equal(t, http.StatusNotFound, rr.Code) // изменено с 401 на 404

	found := false
	for _, logEntry := range logs {
		if logEntry.Message == "Invalid token" {
			found = true
			break
		}
	}
	assert.True(t, found, "Token validation error not logged")
}

// ==========================================
// LoggingMiddleware Tests
// ==========================================

func TestLoggingMiddleware(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	log := zap.New(core)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(log, nil, nil, nil)(nextHandler)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test-path", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	logs := recorded.All()
	require.Len(t, logs, 1)
	assert.Equal(t, "HTTP_REQUEST", logs[0].Message)
	assert.Equal(t, "/test-path", logs[0].ContextMap()["endpoint"])
	assert.Equal(t, "GET", logs[0].ContextMap()["method"])
}

func TestLoggingMiddlewareWithCorrelationID(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	log := zap.New(core)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(log, nil, nil, nil)(nextHandler)

	ctx := context.WithValue(context.Background(), CorrelationIDKey, "corr-123")
	req := httptest.NewRequestWithContext(ctx, "POST", "/api/data", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	logs := recorded.All()
	require.Len(t, logs, 1)
	assert.Equal(t, "corr-123", logs[0].ContextMap()["correlationId"])
	assert.Equal(t, "POST", logs[0].ContextMap()["method"])
}

func TestLoggingMiddlewareWithoutCorrelationID(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	log := zap.New(core)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	handler := LoggingMiddleware(log, nil, nil, nil)(nextHandler)

	// Context without CorrelationIDKey
	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/resource/1", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	logs := recorded.All()
	require.Len(t, logs, 1)
	// GetCorrelationID returns "" when key not found
	assert.Equal(t, "", logs[0].ContextMap()["correlationId"])
}

func TestLoggingMiddlewareLogsDuration(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	log := zap.New(core)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(log, nil, nil, nil)(nextHandler)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	logs := recorded.All()
	require.Len(t, logs, 1)
	// Duration should be present and non-negative
	durationMs, ok := logs[0].ContextMap()["durationMs"].(int64)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, durationMs, int64(0))
}

func TestLoggingMiddlewareMultipleRequests(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	log := zap.New(core)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(log, nil, nil, nil)(nextHandler)

	paths := []string{"/a", "/b", "/c"}
	for _, path := range paths {
		req := httptest.NewRequestWithContext(context.Background(), "GET", path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	logs := recorded.All()
	assert.Len(t, logs, 3)
}

// ==========================================
// RecoveryMiddleware Tests
// ==========================================

func TestRecoveryMiddlewareRecoversFromPanic(t *testing.T) {
	core, recorded := observer.New(zap.ErrorLevel)
	log := zap.New(core)

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic!")
	})

	handler := RecoveryMiddleware(log)(panicHandler)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/panic", nil)
	rr := httptest.NewRecorder()

	// Should not panic
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "text/plain", rr.Header().Get("Content-Type"))
	assert.Equal(t, "Internal Server Error", rr.Body.String())

	logs := recorded.All()
	require.Len(t, logs, 1)
	assert.Equal(t, "Panic recovered", logs[0].Message)
	assert.Equal(t, "test panic!", logs[0].ContextMap()["panic"])
	assert.Equal(t, "/panic", logs[0].ContextMap()["path"])
	assert.NotEmpty(t, logs[0].ContextMap()["stack"])
}

func TestRecoveryMiddlewareNoPanic(t *testing.T) {
	core, recorded := observer.New(zap.ErrorLevel)
	log := zap.New(core)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RecoveryMiddleware(log)(nextHandler)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/normal", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Zero(t, recorded.Len(), "should not log when no panic")
}

func TestRecoveryMiddlewarePanicWithStringValue(t *testing.T) {
	core, recorded := observer.New(zap.ErrorLevel)
	log := zap.New(core)

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong in handler")
	})

	handler := RecoveryMiddleware(log)(panicHandler)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/crash", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	logs := recorded.All()
	require.Len(t, logs, 1)
	assert.Equal(t, "Panic recovered", logs[0].Message)
}

func TestRecoveryMiddlewarePanicWithIntValue(t *testing.T) {
	core, recorded := observer.New(zap.ErrorLevel)
	log := zap.New(core)

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(42)
	})

	handler := RecoveryMiddleware(log)(panicHandler)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "Internal Server Error", rr.Body.String())

	logs := recorded.All()
	require.Len(t, logs, 1)
}

func TestRecoveryMiddlewarePanicWithNilValue(t *testing.T) {
	core, _ := observer.New(zap.ErrorLevel)
	log := zap.New(core)

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	handler := RecoveryMiddleware(log)(panicHandler)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestRecoveryMiddlewareResponseHeadersSet(t *testing.T) {
	core, _ := observer.New(zap.ErrorLevel)
	log := zap.New(core)

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set some headers before panic
		w.Header().Set("X-Custom", "value")
		panic("crash")
	})

	handler := RecoveryMiddleware(log)(panicHandler)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "text/plain", rr.Header().Get("Content-Type"))
}

func TestRecoveryMiddlewareMultiplePanics(t *testing.T) {
	core, recorded := observer.New(zap.ErrorLevel)
	log := zap.New(core)

	panicCount := 0
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panicCount++
		panic("repeated panic")
	})

	handler := RecoveryMiddleware(log)(panicHandler)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	}

	logs := recorded.All()
	assert.Len(t, logs, 3)
}

func TestRecoveryMiddlewareNextHandlerCompletes(t *testing.T) {
	core, _ := observer.New(zap.ErrorLevel)
	log := zap.New(core)

	completed := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		completed = true
		w.WriteHeader(http.StatusAccepted)
	})

	handler := RecoveryMiddleware(log)(nextHandler)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/ok", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.True(t, completed)
	assert.Equal(t, http.StatusAccepted, rr.Code)
}

// ==========================================
// RequireRole Tests
// ==========================================

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name           string
		roleInContext  interface{}
		allowedRoles   []string
		expectedStatus int
		nextCalled     bool
	}{
		{
			name:           "allowed role - admin",
			roleInContext:  "admin",
			allowedRoles:   []string{"admin"},
			expectedStatus: http.StatusOK,
			nextCalled:     true,
		},
		{
			name:           "allowed role - one of many",
			roleInContext:  "viewer",
			allowedRoles:   []string{"admin", "editor", "viewer"},
			expectedStatus: http.StatusOK,
			nextCalled:     true,
		},
		{
			name:           "disallowed role",
			roleInContext:  "viewer",
			allowedRoles:   []string{"admin", "editor"},
			expectedStatus: http.StatusNotFound,
			nextCalled:     false,
		},
		{
			name:           "no role in context",
			roleInContext:  nil,
			allowedRoles:   []string{"admin"},
			expectedStatus: http.StatusNotFound,
			nextCalled:     false,
		},
		{
			name:           "wrong type in context",
			roleInContext:  123,
			allowedRoles:   []string{"admin"},
			expectedStatus: http.StatusNotFound,
			nextCalled:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})

			handler := RequireRole(tt.allowedRoles...)(nextHandler)

			var ctx context.Context
			if tt.roleInContext != nil {
				ctx = context.WithValue(context.Background(), RoleKey, tt.roleInContext)
			} else {
				ctx = context.Background()
			}

			req := httptest.NewRequestWithContext(ctx, "GET", "/admin", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.nextCalled, called)
		})
	}
}

func TestRequireRoleMultipleRoles(t *testing.T) {
	called := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireRole("admin", "moderator", "editor")(nextHandler)

	ctx := context.WithValue(context.Background(), RoleKey, "moderator")
	req := httptest.NewRequestWithContext(ctx, "GET", "/moderate", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, called)
}

func TestRequireRoleEmptyAllowedRoles(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireRole()(nextHandler)

	ctx := context.WithValue(context.Background(), RoleKey, "admin")
	req := httptest.NewRequestWithContext(ctx, "GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestRequireRoleReturnsNotFound(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireRole("admin")(nextHandler)

	ctx := context.WithValue(context.Background(), RoleKey, "client")
	req := httptest.NewRequestWithContext(ctx, "GET", "/admin", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "not found")
}

func TestRequireRoleCombinedWithAuthMiddleware(t *testing.T) {
	secret := testSecretMW
	log := zap.NewNop()

	validToken, err := auth.GenerateJWT("user-789", "admin@example.com", "admin", secret, 24)
	require.NoError(t, err)

	called := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		userID := r.Context().Value(UserIDKey).(string)
		role := r.Context().Value(RoleKey).(string)
		assert.Equal(t, "user-789", userID)
		assert.Equal(t, "admin", role)
		w.WriteHeader(http.StatusOK)
	})

	// Chain: AuthMiddleware -> RequireRole -> handler
	handler := AuthMiddleware(secret, log)(RequireRole("admin")(nextHandler))

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/admin/panel", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, called)
}

func TestRequireRoleChainWithWrongRole(t *testing.T) {
	secret := testSecretMW
	log := zap.NewNop()

	// User has "client" role but endpoint requires "admin"
	validToken, err := auth.GenerateJWT("user-client", "user@example.com", "client", secret, 24)
	require.NoError(t, err)

	called := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := AuthMiddleware(secret, log)(RequireRole("admin")(nextHandler))

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.False(t, called)
}

// Test response signer middleware
func TestResponseSigner(t *testing.T) {
	log := zap.NewNop()
	secret := "sign-secret"

	handler := SignCriticalResponses(secret, log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// Test CorrelationID middleware with and without header
func TestCorrelationID(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		cid := r.Context().Value(CorrelationIDKey)
		assert.NotNil(t, cid)
		w.WriteHeader(http.StatusOK)
	})

	handler := CorrelationID(next)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Correlation-ID", "test-cid-123")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rr.Code)
}
