package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/middleware"
)

func TestAuth_RegisterHandler_InvalidJSON(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/register", bytes.NewReader([]byte(`{invalid`)))

	req.Header.Set("Content-Type", "application/json")

	g.registerHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

}

func TestAuth_RegisterHandler_Success(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	reqBody := []byte(`{"email":"test@example.com","password":"password123","full_name":"Test","role":"client"}`)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/register", bytes.NewReader(reqBody))

	req.Header.Set("Content-Type", "application/json")

	g.registerHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	assert.Contains(t, w.Body.String(), "ok")

}

func TestAuth_RegisterHandler_GRPCError(t *testing.T) {

	g := newTestGateway()

	g.userClient = &errorUserServiceClient{err: grpcError(codes.AlreadyExists, "email already exists")}

	w := httptest.NewRecorder()

	reqBody := []byte(`{"email":"test@example.com","password":"password123","full_name":"Test","role":"client"}`)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/register", bytes.NewReader(reqBody))

	req.Header.Set("Content-Type", "application/json")

	g.registerHandler(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

}

func TestAuth_RegisterWithInviteHandler_InvalidJSON(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/register/invite", bytes.NewReader([]byte(`{invalid`)))

	req.Header.Set("Content-Type", "application/json")

	g.registerWithInviteHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

}

func TestAuth_RegisterWithInviteHandler_Success(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	reqBody := []byte(`{"email":"test@example.com","password":"password123","full_name":"Test","invite_code":"INV-123"}`)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/register/invite", bytes.NewReader(reqBody))

	req.Header.Set("Content-Type", "application/json")

	g.registerWithInviteHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	assert.Contains(t, w.Body.String(), "ok")

}

func TestAuth_ValidateInviteCodeHandler_InvalidJSON(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/invite/validate", bytes.NewReader([]byte(`{invalid`)))

	req.Header.Set("Content-Type", "application/json")

	g.validateInviteCodeHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

}

func TestAuth_ValidateInviteCodeHandler_Success(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	reqBody := []byte(`{"code":"INV-123"}`)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/invite/validate", bytes.NewReader(reqBody))

	req.Header.Set("Content-Type", "application/json")

	g.validateInviteCodeHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	assert.Contains(t, w.Body.String(), "is_valid")

}

func TestAuth_LoginHandler_InvalidJSON(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/login", bytes.NewReader([]byte(`{invalid`)))

	req.Header.Set("Content-Type", "application/json")

	g.loginHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

}

func TestAuth_LoginHandler_Success(t *testing.T) {

	g := newTestGateway()

	withRealRedis(g)

	w := httptest.NewRecorder()

	reqBody := []byte(`{"email":"test@example.com","password":"password123"}`)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/login", bytes.NewReader(reqBody))

	req.Header.Set("Content-Type", "application/json")

	g.loginHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	cookies := w.Result().Cookies()
	var accessCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "fitpulse-access-token" {
			accessCookie = c
			break
		}
	}
	assert.NotNil(t, accessCookie, "fitpulse-access-token cookie should be set")
	assert.True(t, accessCookie.HttpOnly)
	assert.Equal(t, http.SameSiteStrictMode, accessCookie.SameSite)
}

func TestAuth_LoginHandler_InvalidCredentials(t *testing.T) {

	g := newTestGateway()

	g.userClient = &errorUserServiceClient{err: grpcError(codes.Unauthenticated, "invalid credentials")}

	w := httptest.NewRecorder()

	reqBody := []byte(`{"email":"test@example.com","password":"wrongpassword"}`)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/login", bytes.NewReader(reqBody))

	req.Header.Set("Content-Type", "application/json")

	g.loginHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

}

func TestAuth_LoginHandler_EmailNotConfirmed(t *testing.T) {

	g := newTestGateway()

	g.userClient = &errorUserServiceClient{err: grpcError(codes.Unauthenticated, "Email not confirmed")}

	w := httptest.NewRecorder()

	reqBody := []byte(`{"email":"test@example.com","password":"password123"}`)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/login", bytes.NewReader(reqBody))

	req.Header.Set("Content-Type", "application/json")

	g.loginHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

}

func TestAuth_LoginHandler_TOTPEnabled(t *testing.T) {

	g := newTestGateway()

	g.userClient = &totpEnabledUserClient{}

	withRealRedis(g)

	w := httptest.NewRecorder()

	reqBody := []byte(`{"email":"test@example.com","password":"password123"}`)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/login", bytes.NewReader(reqBody))

	req.Header.Set("Content-Type", "application/json")

	g.loginHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	assert.Contains(t, w.Body.String(), "requires_2fa")

}

func TestAuth_LogoutHandler_Success(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/logout", nil)

	g.logoutHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	assert.Contains(t, w.Body.String(), "logged_out")

}

func TestAuth_ConfirmEmailHandler_InvalidJSON(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/confirm-email", bytes.NewReader([]byte(`{invalid`)))

	req.Header.Set("Content-Type", "application/json")

	g.confirmEmailHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

}

func TestAuth_ConfirmEmailHandler_MissingToken(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	reqBody := []byte(`{"token":""}`)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/confirm-email", bytes.NewReader(reqBody))

	req.Header.Set("Content-Type", "application/json")

	g.confirmEmailHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

}

func TestAuth_ConfirmEmailHandler_Success(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	reqBody := []byte(`{"token":"valid-token"}`)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/confirm-email", bytes.NewReader(reqBody))

	req.Header.Set("Content-Type", "application/json")

	g.confirmEmailHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	assert.Contains(t, w.Body.String(), "Email confirmed")

}

func TestAuth_EmailConfirmPageHandler_FileNotFound(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/confirm-email?token=valid", nil)

	g.emailConfirmPageHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

}

func TestAuth_YandexLoginHandler_NotConfigured(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/auth/yandex", nil)

	g.yandexLoginHandler(w, req)

	assert.Equal(t, http.StatusNotImplemented, w.Code)

}

func TestAuth_YandexLoginHandler_Success(t *testing.T) {

	g := newTestGateway()

	withOAuth(g)

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/auth/yandex", nil)

	g.yandexLoginHandler(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)

}

func TestAuth_YandexCallbackHandler_NotConfigured(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/auth/yandex/callback?state=abc&code=xyz", nil)

	g.yandexCallbackHandler(w, req)

	assert.Equal(t, http.StatusNotImplemented, w.Code)

}

func TestAuth_YandexCallbackHandler_InvalidState(t *testing.T) {

	g := newTestGateway()

	withOAuth(g)

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/auth/yandex/callback?state=abc&code=xyz", nil)

	g.yandexCallbackHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

}

func TestAuth_YandexCallbackHandler_MissingCode(t *testing.T) {

	g := newTestGateway()

	withOAuth(g)

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/auth/yandex/callback?state=abc", nil)

	g.yandexCallbackHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

}

func TestAuth_YandexCallbackHandler_ExchangeError(t *testing.T) {

	g := newTestGateway()

	withOAuth(g)

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/auth/yandex/callback?state=abc&code=xyz", nil)

	req.AddCookie(&http.Cookie{Name: yandexOAuthStateCookie, Value: "abc", Secure: true, HttpOnly: true})

	g.yandexCallbackHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

}

func TestAuth_YandexCallbackHandler_InvalidStateCookie(t *testing.T) {

	g := newTestGateway()

	withOAuth(g)

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/auth/yandex/callback?state=abc&code=xyz", nil)

	req.AddCookie(&http.Cookie{Name: yandexOAuthStateCookie, Value: "wrong-state", Secure: true, HttpOnly: true})

	g.yandexCallbackHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

}

func TestAuth_GoogleCallbackHandler_TOTPEnabled(t *testing.T) {

	g := newTestGateway()

	g.userClient = &totpEnabledUserClient{}

	withOAuth(g)

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/auth/yandex/callback?state=abc&code=xyz", nil)

	req.AddCookie(&http.Cookie{Name: yandexOAuthStateCookie, Value: "abc", Secure: true, HttpOnly: true})

	g.yandexCallbackHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

}

func TestAuth_SetupTOTPHandler_Unauthorized(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/totp/setup", nil)

	g.setupTOTPHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

}

func TestAuth_SetupTOTPHandler_Success(t *testing.T) {

	g := newTestGateway()

	withRealRedis(g)

	g.sessionStore = nil

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/totp/setup", nil)

	g.setupTOTPHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	assert.Contains(t, w.Body.String(), "qr_code_url")

}

func TestAuth_SetupTOTPHandler_RateLimited(t *testing.T) {

	g := newTestGateway()

	g.valkeyDB = nil

	ctx := context.Background()

	for i := 0; i < 6; i++ {

		_ = g.enforceTOTPRateLimit(ctx, "setup:user-123")

	}

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/totp/setup", nil)

	g.setupTOTPHandler(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)

}

func TestAuth_ConfirmTOTPHandler_Unauthorized(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/totp/confirm", bytes.NewReader([]byte(`{}`)))

	req.Header.Set("Content-Type", "application/json")

	g.confirmTOTPHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

}

func TestAuth_ConfirmTOTPHandler_InvalidJSON(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/totp/confirm", bytes.NewReader([]byte(`{invalid`)))

	req.Header.Set("Content-Type", "application/json")

	g.confirmTOTPHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

}

func TestAuth_ConfirmTOTPHandler_GRPCError(t *testing.T) {

	g := newTestGateway()

	g.userClient = &errorUserServiceClient{err: grpcError(codes.Internal, "internal error")}

	w := httptest.NewRecorder()

	reqBody := []byte(`{"passcode":"123456"}`)

	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/totp/confirm", bytes.NewReader(reqBody))

	req.Header.Set("Content-Type", "application/json")

	g.confirmTOTPHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

}

func TestAuth_DisableTOTPHandler_Unauthorized(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/totp/disable", bytes.NewReader([]byte(`{}`)))

	req.Header.Set("Content-Type", "application/json")

	g.disableTOTPHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

}

func TestAuth_DisableTOTPHandler_InvalidJSON(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/totp/disable", bytes.NewReader([]byte(`{invalid`)))

	req.Header.Set("Content-Type", "application/json")

	g.disableTOTPHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

}

func TestAuth_DisableTOTPHandler_GRPCError(t *testing.T) {

	g := newTestGateway()

	g.userClient = &errorUserServiceClient{err: grpcError(codes.Internal, "internal error")}

	w := httptest.NewRecorder()

	reqBody := []byte(`{"passcode":"123456"}`)

	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/totp/disable", bytes.NewReader(reqBody))

	req.Header.Set("Content-Type", "application/json")

	g.disableTOTPHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

}

func TestAuth_TOTPStatusHandler_Unauthorized(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/totp/status", nil)

	g.totpStatusHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

}

func TestAuth_TOTPStatusHandler_Success(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "GET", "/api/v1/totp/status", nil)

	g.totpStatusHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	assert.Contains(t, w.Body.String(), "enabled")

}

func TestAuth_RefreshHandler_InvalidJSON(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/refresh", bytes.NewReader([]byte(`{invalid`)))

	req.Header.Set("Content-Type", "application/json")

	g.refreshHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

}

func TestAuth_RefreshHandler_MissingToken(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	reqBody := []byte(`{"refresh_token":""}`)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/refresh", bytes.NewReader(reqBody))

	req.Header.Set("Content-Type", "application/json")

	g.refreshHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

}

func TestAuth_RefreshHandler_InvalidToken(t *testing.T) {

	g := newTestGateway()

	withRealRedis(g)

	w := httptest.NewRecorder()

	reqBody := []byte(`{"refresh_token":"invalid-token"}`)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/refresh", bytes.NewReader(reqBody))

	req.Header.Set("Content-Type", "application/json")

	g.refreshHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

}

func TestAuth_UserTOTPEnabled_EmptyUserID(t *testing.T) {

	g := newTestGateway()

	ctx := context.Background()

	result := g.userTOTPEnabled(ctx, "")

	assert.False(t, result)

}

func TestAuth_UserTOTPEnabled_Success(t *testing.T) {

	g := newTestGateway()

	g.userClient = &mockUserServiceClient{}

	ctx := context.Background()

	result := g.userTOTPEnabled(ctx, "user-123")

	assert.False(t, result)

}

func TestAuth_UserTOTPEnabled_TOTPEnabled(t *testing.T) {

	g := newTestGateway()

	g.userClient = &totpEnabledUserClient{}

	ctx := context.Background()

	result := g.userTOTPEnabled(ctx, "user-123")

	assert.True(t, result)

}

func TestAuth_EnforceTOTPRateLimit_NilValkeyDB(t *testing.T) {

	g := newTestGateway()

	g.valkeyDB = nil

	ctx := context.Background()

	err := g.enforceTOTPRateLimit(ctx, "user-123")

	assert.NoError(t, err)

}

func TestAuth_EnforceTOTPRateLimit_ExceedsLimit(t *testing.T) {

	g := newTestGateway()

	g.valkeyDB = nil

	ctx := context.Background()

	for i := 0; i < 6; i++ {

		_ = g.enforceTOTPRateLimit(ctx, "user-456")

	}

	err := g.enforceTOTPRateLimit(ctx, "user-456")

	assert.Error(t, err)

	assert.Contains(t, err.Error(), "too many 2FA attempts")

}

func TestAuth_RequireCriticalSession_NilSessionStore(t *testing.T) {

	g := newTestGateway()

	g.sessionStore = nil

	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "GET", "/", nil)

	err := g.requireCriticalSession(req, "user-123")

	assert.NoError(t, err)

}

func TestAuth_RequireCriticalSession_MissingUserID(t *testing.T) {

	g := newTestGateway()

	g.sessionStore = nil

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)

	err := g.requireCriticalSession(req, "user-123")

	assert.NoError(t, err)

}

func TestAuth_RequireCriticalSession_InvalidToken(t *testing.T) {

	g := newTestGateway()

	withRealSessionStore(g)

	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "GET", "/", nil)

	req.Header.Set("X-Critical-Session-Token", "invalid-token")

	err := g.requireCriticalSession(req, "user-123")

	assert.Error(t, err)

}

func TestAuth_CriticalSessionHandler_SessionStoreUnavailable(t *testing.T) {

	g := newTestGateway()

	g.sessionStore = nil

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/", nil)

	g.criticalSessionHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

}

func TestAuth_CriticalSessionHandler_Unauthorized(t *testing.T) {

	g := newTestGateway()

	g.sessionStore = nil

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/", nil)

	g.criticalSessionHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

}

func TestAuth_GenerateOAuthState_Success(t *testing.T) {

	state, err := generateOAuthState()

	assert.NoError(t, err)

	assert.Len(t, state, 64)

}

func TestAuth_IssueJWT_EmptyUserID(t *testing.T) {

	g := newTestGateway()

	ctx := context.Background()

	token, err := g.issueJWT(ctx, "")

	assert.Empty(t, token)

	assert.Error(t, err)

}

func TestAuth_IssueRefreshToken_NilValkeyDB(t *testing.T) {

	g := newTestGateway()

	g.valkeyDB = nil

	ctx := context.Background()

	token, err := g.issueRefreshToken(ctx, "user-123")

	assert.Empty(t, token)

	assert.Error(t, err)

	assert.Contains(t, err.Error(), "valkey unavailable")

}

func TestAuth_RotateRefreshToken_InvalidToken(t *testing.T) {

	g := newTestGateway()

	withRealRedis(g)

	ctx := context.Background()

	access, refresh, err := g.rotateRefreshToken(ctx, "invalid-token")

	assert.Empty(t, access)

	assert.Empty(t, refresh)

	assert.Error(t, err)

}

func TestAuth_RotateRefreshToken_ReuseDetected(t *testing.T) {

	g := newTestGateway()

	withRealRedis(g)

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:99999"})

	g.valkeyDB = redisClient

	ctx := context.Background()

	oldToken := "old-refresh-token"

	oldFingerprint := g.tokenProvider.ComputeTokenFingerprint(oldToken)

	userID := "user-123"

	err := redisClient.Set(ctx, refreshTokenPrefix+oldToken, userID, 7*24*time.Hour).Err()

	if err != nil {

		t.Skip("redis not available")

	}

	err = redisClient.SAdd(ctx, refreshRevokedPrefix+userID, oldFingerprint).Err()

	if err != nil {

		t.Skip("redis not available")

	}

	access, refresh, err := g.rotateRefreshToken(ctx, oldToken)

	assert.Empty(t, access)

	assert.Empty(t, refresh)

	assert.Error(t, err)

	assert.Contains(t, err.Error(), "reuse detected")

}

func TestAuth_InvalidateAllUserSessions(t *testing.T) {

	g := newTestGateway()

	withRealRedis(g)

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:99999"})

	g.valkeyDB = redisClient

	ctx := context.Background()

	err := redisClient.Set(ctx, refreshIssuedPrefix+"user-123", "token1", 7*24*time.Hour).Err()

	if err != nil {

		t.Skip("redis not available")

	}

	g.invalidateAllUserSessions(ctx, "user-123")

}

func TestAuth_InvalidateAllUserSessions_NilSessionStore(t *testing.T) {

	g := newTestGateway()

	g.sessionStore = nil

	withRealRedis(g)

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:99999"})

	g.valkeyDB = redisClient

	ctx := context.Background()

	err := redisClient.Set(ctx, refreshIssuedPrefix+"user-123", "token1", 7*24*time.Hour).Err()

	if err != nil {

		t.Skip("redis not available")

	}

	g.invalidateAllUserSessions(ctx, "user-123")

}

// ========== Helper Mock Clients ==========

type errorUserServiceClient struct {
	err error
}

func (m *errorUserServiceClient) Register(ctx context.Context, req *user.RegisterRequest, opts ...grpc.CallOption) (*user.RegisterResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) RegisterWithInvite(ctx context.Context, req *user.RegisterWithInviteRequest, opts ...grpc.CallOption) (*user.RegisterResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) ConfirmEmail(ctx context.Context, req *user.ConfirmEmailRequest, opts ...grpc.CallOption) (*user.ConfirmEmailResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) Login(ctx context.Context, req *user.LoginRequest, opts ...grpc.CallOption) (*user.LoginResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) AuthenticateYandex(ctx context.Context, req *user.AuthenticateYandexRequest, opts ...grpc.CallOption) (*user.LoginResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) GetProfile(ctx context.Context, req *user.GetProfileRequest, opts ...grpc.CallOption) (*user.UserProfile, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) GetUserByEmail(ctx context.Context, req *user.GetUserByEmailRequest, opts ...grpc.CallOption) (*user.UserProfile, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) UpdateProfile(ctx context.Context, req *user.UpdateProfileRequest, opts ...grpc.CallOption) (*user.UserProfile, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) ChangePassword(ctx context.Context, req *user.ChangePasswordRequest, opts ...grpc.CallOption) (*user.ChangePasswordResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) ChangeEmail(ctx context.Context, req *user.ChangeEmailRequest, opts ...grpc.CallOption) (*user.ChangeEmailResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) UploadProfilePhoto(ctx context.Context, req *user.UploadProfilePhotoRequest, opts ...grpc.CallOption) (*user.UploadProfilePhotoResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) RemoveProfilePhoto(ctx context.Context, req *user.RemoveProfilePhotoRequest, opts ...grpc.CallOption) (*user.RemoveProfilePhotoResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) ChangeNickname(ctx context.Context, req *user.ChangeNicknameRequest, opts ...grpc.CallOption) (*user.ChangeNicknameResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) ListDevices(ctx context.Context, req *user.ListDevicesRequest, opts ...grpc.CallOption) (*user.ListDevicesResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) AddDevice(ctx context.Context, req *user.AddDeviceRequest, opts ...grpc.CallOption) (*user.AddDeviceResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) RemoveDevice(ctx context.Context, req *user.RemoveDeviceRequest, opts ...grpc.CallOption) (*user.RemoveDeviceResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) SyncDeviceData(ctx context.Context, req *user.SyncDeviceDataRequest, opts ...grpc.CallOption) (*user.SyncDeviceDataResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) GetTrainingStats(ctx context.Context, req *user.GetTrainingStatsRequest, opts ...grpc.CallOption) (*user.GetTrainingStatsResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) GetAchievements(ctx context.Context, req *user.GetAchievementsRequest, opts ...grpc.CallOption) (*user.GetAchievementsResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) ListUsers(ctx context.Context, req *user.ListUsersRequest, opts ...grpc.CallOption) (*user.ListUsersResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) ValidateInviteCode(ctx context.Context, req *user.ValidateInviteCodeRequest, opts ...grpc.CallOption) (*user.ValidateInviteCodeResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) SetupTOTP(ctx context.Context, req *user.SetupTOTPRequest, opts ...grpc.CallOption) (*user.SetupTOTPResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) ConfirmTOTP(ctx context.Context, req *user.ConfirmTOTPRequest, opts ...grpc.CallOption) (*user.ConfirmTOTPResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) VerifyTOTP(ctx context.Context, req *user.VerifyTOTPRequest, opts ...grpc.CallOption) (*user.VerifyTOTPResponse, error) {

	return &user.VerifyTOTPResponse{Valid: true}, nil

}

func (m *errorUserServiceClient) DisableTOTP(ctx context.Context, req *user.DisableTOTPRequest, opts ...grpc.CallOption) (*user.DisableTOTPResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) RefreshToken(ctx context.Context, req *user.RefreshTokenRequest, opts ...grpc.CallOption) (*user.RefreshTokenResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) ListHealthConditions(ctx context.Context, req *user.ListHealthConditionsRequest, opts ...grpc.CallOption) (*user.ListHealthConditionsResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) UpsertHealthCondition(ctx context.Context, req *user.UpsertHealthConditionRequest, opts ...grpc.CallOption) (*user.HealthCondition, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) DeleteHealthCondition(ctx context.Context, req *user.DeleteHealthConditionRequest, opts ...grpc.CallOption) (*user.DeleteHealthConditionResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) ListBodyComposition(ctx context.Context, req *user.ListBodyCompositionRequest, opts ...grpc.CallOption) (*user.ListBodyCompositionResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) CreateBodyComposition(ctx context.Context, req *user.CreateBodyCompositionRequest, opts ...grpc.CallOption) (*user.BodyCompositionRecord, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) ListMenstrualCycles(ctx context.Context, req *user.ListMenstrualCyclesRequest, opts ...grpc.CallOption) (*user.ListMenstrualCyclesResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) CreateMenstrualCycle(ctx context.Context, req *user.CreateMenstrualCycleRequest, opts ...grpc.CallOption) (*user.MenstrualCycle, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) UpdateMenstrualCycle(ctx context.Context, req *user.UpdateMenstrualCycleRequest, opts ...grpc.CallOption) (*user.MenstrualCycle, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) DeleteMenstrualCycle(ctx context.Context, req *user.DeleteMenstrualCycleRequest, opts ...grpc.CallOption) (*user.DeleteMenstrualCycleResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) GetUserClaims(ctx context.Context, req *user.GetUserClaimsRequest, opts ...grpc.CallOption) (*user.GetUserClaimsResponse, error) {

	return &user.GetUserClaimsResponse{Email: "test@example.com", Role: "admin", TotpEnabled: false}, nil

}

func (m *errorUserServiceClient) DeleteProfile(ctx context.Context, req *user.DeleteProfileRequest, opts ...grpc.CallOption) (*user.DeleteProfileResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) ExportUserData(ctx context.Context, req *user.ExportUserDataRequest, opts ...grpc.CallOption) (*user.ExportUserDataResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) AdminListInvites(ctx context.Context, req *user.AdminListInvitesRequest, opts ...grpc.CallOption) (*user.AdminListInvitesResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) AdminCreateInvite(ctx context.Context, req *user.AdminCreateInviteRequest, opts ...grpc.CallOption) (*user.AdminCreateInviteResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) AdminRevokeInvite(ctx context.Context, req *user.AdminRevokeInviteRequest, opts ...grpc.CallOption) (*user.AdminRevokeInviteResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) AdminDeleteUser(ctx context.Context, req *user.AdminDeleteUserRequest, opts ...grpc.CallOption) (*user.AdminDeleteUserResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) AdminBanUser(ctx context.Context, req *user.AdminBanUserRequest, opts ...grpc.CallOption) (*user.AdminBanUserResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) AdminUnbanUser(ctx context.Context, req *user.AdminUnbanUserRequest, opts ...grpc.CallOption) (*user.AdminUnbanUserResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) SaveSurvey(ctx context.Context, req *user.SaveSurveyRequest, opts ...grpc.CallOption) (*user.SaveSurveyResponse, error) {

	return nil, m.err

}

func (m *errorUserServiceClient) LoadSurvey(ctx context.Context, req *user.LoadSurveyRequest, opts ...grpc.CallOption) (*user.LoadSurveyResponse, error) {

	return nil, m.err

}

type totpEnabledUserClient struct{}

func (m *totpEnabledUserClient) Register(ctx context.Context, req *user.RegisterRequest, opts ...grpc.CallOption) (*user.RegisterResponse, error) {

	return &user.RegisterResponse{UserId: "user-123"}, nil

}

func (m *totpEnabledUserClient) RegisterWithInvite(ctx context.Context, req *user.RegisterWithInviteRequest, opts ...grpc.CallOption) (*user.RegisterResponse, error) {

	return &user.RegisterResponse{UserId: "user-123"}, nil

}

func (m *totpEnabledUserClient) ConfirmEmail(ctx context.Context, req *user.ConfirmEmailRequest, opts ...grpc.CallOption) (*user.ConfirmEmailResponse, error) {

	return &user.ConfirmEmailResponse{UserId: "user-123", Message: "ok"}, nil

}

func (m *totpEnabledUserClient) Login(ctx context.Context, req *user.LoginRequest, opts ...grpc.CallOption) (*user.LoginResponse, error) {

	return &user.LoginResponse{AccessToken: "token", UserId: "user-123", Role: "client"}, nil

}

func (m *totpEnabledUserClient) GetUserClaims(ctx context.Context, req *user.GetUserClaimsRequest, opts ...grpc.CallOption) (*user.GetUserClaimsResponse, error) {

	return &user.GetUserClaimsResponse{Email: "test@example.com", Role: "admin", TotpEnabled: true}, nil

}

func (m *totpEnabledUserClient) AuthenticateYandex(ctx context.Context, req *user.AuthenticateYandexRequest, opts ...grpc.CallOption) (*user.LoginResponse, error) {

	return &user.LoginResponse{AccessToken: "token", UserId: "user-123", Role: "client"}, nil

}

func (m *totpEnabledUserClient) GetProfile(ctx context.Context, req *user.GetProfileRequest, opts ...grpc.CallOption) (*user.UserProfile, error) {

	return &user.UserProfile{UserId: "user-123", Email: "test@example.com"}, nil

}

func (m *totpEnabledUserClient) GetUserByEmail(ctx context.Context, req *user.GetUserByEmailRequest, opts ...grpc.CallOption) (*user.UserProfile, error) {

	return &user.UserProfile{UserId: "user-123", Email: req.Email, EmailConfirmed: true}, nil

}

func (m *totpEnabledUserClient) UpdateProfile(ctx context.Context, req *user.UpdateProfileRequest, opts ...grpc.CallOption) (*user.UserProfile, error) {

	return &user.UserProfile{UserId: "user-123", Email: "test@example.com"}, nil

}

func (m *totpEnabledUserClient) ChangePassword(ctx context.Context, req *user.ChangePasswordRequest, opts ...grpc.CallOption) (*user.ChangePasswordResponse, error) {

	return &user.ChangePasswordResponse{Message: "ok"}, nil

}

func (m *totpEnabledUserClient) ChangeEmail(ctx context.Context, req *user.ChangeEmailRequest, opts ...grpc.CallOption) (*user.ChangeEmailResponse, error) {

	return &user.ChangeEmailResponse{Message: "ok"}, nil

}

func (m *totpEnabledUserClient) UploadProfilePhoto(ctx context.Context, req *user.UploadProfilePhotoRequest, opts ...grpc.CallOption) (*user.UploadProfilePhotoResponse, error) {

	return &user.UploadProfilePhotoResponse{PhotoUrl: "url"}, nil

}

func (m *totpEnabledUserClient) RemoveProfilePhoto(ctx context.Context, req *user.RemoveProfilePhotoRequest, opts ...grpc.CallOption) (*user.RemoveProfilePhotoResponse, error) {

	return &user.RemoveProfilePhotoResponse{Message: "ok"}, nil

}

func (m *totpEnabledUserClient) ChangeNickname(ctx context.Context, req *user.ChangeNicknameRequest, opts ...grpc.CallOption) (*user.ChangeNicknameResponse, error) {

	return &user.ChangeNicknameResponse{Message: "ok"}, nil

}

func (m *totpEnabledUserClient) ListDevices(ctx context.Context, req *user.ListDevicesRequest, opts ...grpc.CallOption) (*user.ListDevicesResponse, error) {

	return &user.ListDevicesResponse{}, nil

}

func (m *totpEnabledUserClient) AddDevice(ctx context.Context, req *user.AddDeviceRequest, opts ...grpc.CallOption) (*user.AddDeviceResponse, error) {

	return &user.AddDeviceResponse{}, nil

}

func (m *totpEnabledUserClient) RemoveDevice(ctx context.Context, req *user.RemoveDeviceRequest, opts ...grpc.CallOption) (*user.RemoveDeviceResponse, error) {

	return &user.RemoveDeviceResponse{Message: "ok"}, nil

}

func (m *totpEnabledUserClient) SyncDeviceData(ctx context.Context, req *user.SyncDeviceDataRequest, opts ...grpc.CallOption) (*user.SyncDeviceDataResponse, error) {

	return &user.SyncDeviceDataResponse{Message: "ok"}, nil

}

func (m *totpEnabledUserClient) GetTrainingStats(ctx context.Context, req *user.GetTrainingStatsRequest, opts ...grpc.CallOption) (*user.GetTrainingStatsResponse, error) {

	return &user.GetTrainingStatsResponse{}, nil

}

func (m *totpEnabledUserClient) GetAchievements(ctx context.Context, req *user.GetAchievementsRequest, opts ...grpc.CallOption) (*user.GetAchievementsResponse, error) {

	return &user.GetAchievementsResponse{Achievements: []*user.Achievement{}}, nil

}

func (m *totpEnabledUserClient) ListUsers(ctx context.Context, req *user.ListUsersRequest, opts ...grpc.CallOption) (*user.ListUsersResponse, error) {

	return &user.ListUsersResponse{}, nil

}

func (m *totpEnabledUserClient) ValidateInviteCode(ctx context.Context, req *user.ValidateInviteCodeRequest, opts ...grpc.CallOption) (*user.ValidateInviteCodeResponse, error) {

	return &user.ValidateInviteCodeResponse{IsValid: true}, nil

}

func (m *totpEnabledUserClient) SetupTOTP(ctx context.Context, req *user.SetupTOTPRequest, opts ...grpc.CallOption) (*user.SetupTOTPResponse, error) {

	return &user.SetupTOTPResponse{}, nil

}

func (m *totpEnabledUserClient) VerifyTOTP(ctx context.Context, req *user.VerifyTOTPRequest, opts ...grpc.CallOption) (*user.VerifyTOTPResponse, error) {

	return &user.VerifyTOTPResponse{Valid: true}, nil

}

func (m *totpEnabledUserClient) ConfirmTOTP(ctx context.Context, req *user.ConfirmTOTPRequest, opts ...grpc.CallOption) (*user.ConfirmTOTPResponse, error) {

	return &user.ConfirmTOTPResponse{Success: true}, nil

}

func (m *totpEnabledUserClient) DisableTOTP(ctx context.Context, req *user.DisableTOTPRequest, opts ...grpc.CallOption) (*user.DisableTOTPResponse, error) {

	return &user.DisableTOTPResponse{Success: true}, nil

}

func (m *totpEnabledUserClient) RefreshToken(ctx context.Context, req *user.RefreshTokenRequest, opts ...grpc.CallOption) (*user.RefreshTokenResponse, error) {

	return &user.RefreshTokenResponse{}, nil

}

func (m *totpEnabledUserClient) ListHealthConditions(ctx context.Context, req *user.ListHealthConditionsRequest, opts ...grpc.CallOption) (*user.ListHealthConditionsResponse, error) {

	return &user.ListHealthConditionsResponse{}, nil

}

func (m *totpEnabledUserClient) UpsertHealthCondition(ctx context.Context, req *user.UpsertHealthConditionRequest, opts ...grpc.CallOption) (*user.HealthCondition, error) {

	return &user.HealthCondition{}, nil

}

func (m *totpEnabledUserClient) DeleteHealthCondition(ctx context.Context, req *user.DeleteHealthConditionRequest, opts ...grpc.CallOption) (*user.DeleteHealthConditionResponse, error) {

	return &user.DeleteHealthConditionResponse{Success: true}, nil

}

func (m *totpEnabledUserClient) ListBodyComposition(ctx context.Context, req *user.ListBodyCompositionRequest, opts ...grpc.CallOption) (*user.ListBodyCompositionResponse, error) {

	return &user.ListBodyCompositionResponse{}, nil

}

func (m *totpEnabledUserClient) CreateBodyComposition(ctx context.Context, req *user.CreateBodyCompositionRequest, opts ...grpc.CallOption) (*user.BodyCompositionRecord, error) {

	return &user.BodyCompositionRecord{}, nil

}

func (m *totpEnabledUserClient) ListMenstrualCycles(ctx context.Context, req *user.ListMenstrualCyclesRequest, opts ...grpc.CallOption) (*user.ListMenstrualCyclesResponse, error) {

	return &user.ListMenstrualCyclesResponse{}, nil

}

func (m *totpEnabledUserClient) CreateMenstrualCycle(ctx context.Context, req *user.CreateMenstrualCycleRequest, opts ...grpc.CallOption) (*user.MenstrualCycle, error) {

	return &user.MenstrualCycle{}, nil

}

func (m *totpEnabledUserClient) UpdateMenstrualCycle(ctx context.Context, req *user.UpdateMenstrualCycleRequest, opts ...grpc.CallOption) (*user.MenstrualCycle, error) {

	return &user.MenstrualCycle{}, nil

}

func (m *totpEnabledUserClient) DeleteMenstrualCycle(ctx context.Context, req *user.DeleteMenstrualCycleRequest, opts ...grpc.CallOption) (*user.DeleteMenstrualCycleResponse, error) {

	return &user.DeleteMenstrualCycleResponse{Success: true}, nil

}

func (m *totpEnabledUserClient) DeleteProfile(ctx context.Context, req *user.DeleteProfileRequest, opts ...grpc.CallOption) (*user.DeleteProfileResponse, error) {

	return &user.DeleteProfileResponse{Status: "deleted"}, nil

}

func (m *totpEnabledUserClient) ExportUserData(ctx context.Context, req *user.ExportUserDataRequest, opts ...grpc.CallOption) (*user.ExportUserDataResponse, error) {

	return &user.ExportUserDataResponse{UserId: req.UserId, Email: "test@example.com"}, nil

}

func (m *totpEnabledUserClient) AdminListInvites(ctx context.Context, req *user.AdminListInvitesRequest, opts ...grpc.CallOption) (*user.AdminListInvitesResponse, error) {

	return &user.AdminListInvitesResponse{}, nil

}

func (m *totpEnabledUserClient) AdminCreateInvite(ctx context.Context, req *user.AdminCreateInviteRequest, opts ...grpc.CallOption) (*user.AdminCreateInviteResponse, error) {

	return &user.AdminCreateInviteResponse{Code: "INV-test"}, nil

}

func (m *totpEnabledUserClient) AdminRevokeInvite(ctx context.Context, req *user.AdminRevokeInviteRequest, opts ...grpc.CallOption) (*user.AdminRevokeInviteResponse, error) {

	return &user.AdminRevokeInviteResponse{Success: true}, nil

}

func (m *totpEnabledUserClient) AdminDeleteUser(ctx context.Context, req *user.AdminDeleteUserRequest, opts ...grpc.CallOption) (*user.AdminDeleteUserResponse, error) {

	return &user.AdminDeleteUserResponse{Success: true}, nil

}

func (m *totpEnabledUserClient) AdminBanUser(ctx context.Context, req *user.AdminBanUserRequest, opts ...grpc.CallOption) (*user.AdminBanUserResponse, error) {

	return &user.AdminBanUserResponse{Success: true}, nil

}

func (m *totpEnabledUserClient) AdminUnbanUser(ctx context.Context, req *user.AdminUnbanUserRequest, opts ...grpc.CallOption) (*user.AdminUnbanUserResponse, error) {

	return &user.AdminUnbanUserResponse{Success: true}, nil

}

func (m *totpEnabledUserClient) SaveSurvey(ctx context.Context, req *user.SaveSurveyRequest, opts ...grpc.CallOption) (*user.SaveSurveyResponse, error) {

	return &user.SaveSurveyResponse{Status: "ok"}, nil

}

func (m *totpEnabledUserClient) LoadSurvey(ctx context.Context, req *user.LoadSurveyRequest, opts ...grpc.CallOption) (*user.LoadSurveyResponse, error) {

	return &user.LoadSurveyResponse{Status: "ok"}, nil

}

func TestAuth_VerifyTOTPHandler_InvalidJSON(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/totp/verify", bytes.NewReader([]byte(`{invalid`)))

	req.Header.Set("Content-Type", "application/json")

	g.verifyTOTPHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

}

func TestAuth_VerifyTOTPHandler_MissingFields(t *testing.T) {

	g := newTestGateway()

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/totp/verify", bytes.NewReader([]byte(`{}`)))

	req.Header.Set("Content-Type", "application/json")

	g.verifyTOTPHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

}

func TestAuth_VerifyTOTPHandler_InvalidTempToken(t *testing.T) {

	g := newTestGateway()

	withRealRedis(g)

	w := httptest.NewRecorder()

	reqBody := []byte(`{"temp_token":"invalid","passcode":"123456"}`)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/totp/verify", bytes.NewReader(reqBody))

	req.Header.Set("Content-Type", "application/json")

	g.verifyTOTPHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

}

func TestAuth_VerifyTOTPHandler_RateLimited(t *testing.T) {

	g := newTestGateway()

	withRealRedis(g)

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:99999"})

	g.valkeyDB = redisClient

	ctx := context.Background()

	err := redisClient.Set(ctx, twoFATempPrefix+"token", "user-123", 5*time.Minute).Err()

	if err != nil {

		t.Skip("redis not available")

	}

	for i := 0; i < 6; i++ {

		_ = g.enforceTOTPRateLimit(ctx, "verify:user-123")

	}

	w := httptest.NewRecorder()

	reqBody := []byte(`{"temp_token":"token","passcode":"123456"}`)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/totp/verify", bytes.NewReader(reqBody))

	req.Header.Set("Content-Type", "application/json")

	g.verifyTOTPHandler(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)

}

func TestAuth_VerifyTOTPHandler_GRPCError(t *testing.T) {

	g := newTestGateway()

	g.userClient = &errorUserServiceClient{err: grpcError(codes.Internal, "internal error")}

	withRealRedis(g)

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:99999"})

	g.valkeyDB = redisClient

	ctx := context.Background()

	err := redisClient.Set(ctx, twoFATempPrefix+"token", "user-123", 5*time.Minute).Err()

	if err != nil {

		t.Skip("redis not available")

	}

	w := httptest.NewRecorder()

	reqBody := []byte(`{"temp_token":"token","passcode":"123456"}`)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/totp/verify", bytes.NewReader(reqBody))

	req.Header.Set("Content-Type", "application/json")

	g.verifyTOTPHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

}

func TestAuth_VerifyTOTPHandler_InvalidCode(t *testing.T) {

	g := newTestGateway()

	withRealRedis(g)

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:99999"})

	g.valkeyDB = redisClient

	ctx := context.Background()

	err := redisClient.Set(ctx, twoFATempPrefix+"token", "user-123", 5*time.Minute).Err()

	if err != nil {

		t.Skip("redis not available")

	}

	w := httptest.NewRecorder()

	reqBody := []byte(`{"temp_token":"token","passcode":"123456"}`)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/totp/verify", bytes.NewReader(reqBody))

	req.Header.Set("Content-Type", "application/json")

	g.verifyTOTPHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

}

func TestAuth_VerifyTOTPHandler_Success(t *testing.T) {

	g := newTestGateway()

	withRealRedis(g)

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:99999"})

	g.valkeyDB = redisClient

	ctx := context.Background()

	err := redisClient.Set(ctx, twoFATempPrefix+"token", "user-123", 5*time.Minute).Err()

	if err != nil {

		t.Skip("redis not available")

	}

	w := httptest.NewRecorder()

	reqBody := []byte(`{"temp_token":"token","passcode":"123456","is_backup_code":false}`)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/totp/verify", bytes.NewReader(reqBody))

	req.Header.Set("Content-Type", "application/json")

	g.verifyTOTPHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	cookies := w.Result().Cookies()
	var accessCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "fitpulse-access-token" {
			accessCookie = c
			break
		}
	}
	assert.NotNil(t, accessCookie, "fitpulse-access-token cookie should be set")
	assert.True(t, accessCookie.HttpOnly)
	assert.Equal(t, http.SameSiteStrictMode, accessCookie.SameSite)
}
