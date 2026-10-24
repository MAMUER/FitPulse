package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/auth/claims"
	"github.com/MAMUER/project/internal/cache"
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

func authValkeyGateway(t *testing.T) *gateway {
	t.Helper()
	g := newTestGateway()
	mr, mrErr := miniredis.Run()
	require.NoError(t, mrErr)
	t.Cleanup(mr.Close)
	g.valkeyDB = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return g
}

func authGatewayWithSession(t *testing.T) *gateway {
	t.Helper()
	g := authValkeyGateway(t)
	g.sessionStore = cache.NewSessionStoreFromRedis(g.valkeyDB)
	return g
}

func authUserRequest() *http.Request {
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user-123")
	return httptest.NewRequestWithContext(ctx, "POST", "/api/v1/auth", nil)
}

func authJSONRequest(target, body string) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), "POST", target, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func authUserJSONRequest(body string) *http.Request {
	req := authJSONRequest("/api/v1/auth", body)
	*req = *req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, "user-123"))
	return req
}

func authCookieValue(t *testing.T, w *httptest.ResponseRecorder, name string) string {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

type authFailingWriter struct {
	header http.Header
	status int
}

func (w *authFailingWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *authFailingWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

func (w *authFailingWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}

type authMessageUserClient struct {
	totpEnabledUserClient
}

func (m *authMessageUserClient) Register(ctx context.Context, req *user.RegisterRequest, opts ...grpc.CallOption) (*user.RegisterResponse, error) {
	return &user.RegisterResponse{UserId: "user-123", Message: "Registration complete"}, nil
}

func (m *authMessageUserClient) RegisterWithInvite(ctx context.Context, req *user.RegisterWithInviteRequest, opts ...grpc.CallOption) (*user.RegisterResponse, error) {
	return &user.RegisterResponse{UserId: "invited-1", Message: "Invite accepted"}, nil
}

type authClaimsFailingClient struct {
	totpEnabledUserClient
}

func (m *authClaimsFailingClient) GetUserClaims(ctx context.Context, req *user.GetUserClaimsRequest, opts ...grpc.CallOption) (*user.GetUserClaimsResponse, error) {
	return nil, grpcError(codes.Internal, "claims unavailable")
}

type authInvalidTOTPClient struct {
	totpEnabledUserClient
}

func (m *authInvalidTOTPClient) VerifyTOTP(ctx context.Context, req *user.VerifyTOTPRequest, opts ...grpc.CallOption) (*user.VerifyTOTPResponse, error) {
	return &user.VerifyTOTPResponse{Valid: false, BackupCodesRemaining: 1}, nil
}

type authBadQRUserClient struct {
	totpEnabledUserClient
}

func (m *authBadQRUserClient) SetupTOTP(ctx context.Context, req *user.SetupTOTPRequest, opts ...grpc.CallOption) (*user.SetupTOTPResponse, error) {
	return &user.SetupTOTPResponse{QrCodeUrl: "://bad-qr", Secret: "SECRET", BackupCodes: []string{"code-1"}}, nil
}

type authBrokenTokenProvider struct{}

func (authBrokenTokenProvider) GenerateAccessToken(userID, email, role string, ttl time.Duration) (string, error) {
	return "", errors.New("token provider unavailable")
}

func (authBrokenTokenProvider) GenerateRefreshToken() string {
	return "refresh-token"
}

func (authBrokenTokenProvider) ValidateAccessToken(token string) (*claims.Claims, error) {
	return nil, errors.New("token provider unavailable")
}

func (authBrokenTokenProvider) ComputeTokenFingerprint(token string) string {
	return "fingerprint"
}

func (authBrokenTokenProvider) PublicKeyPEMToJWKS(publicKeyPEM string) ([]byte, error) {
	return []byte("{}"), nil
}

func (authBrokenTokenProvider) PublicKeyPEM() string {
	return ""
}

func TestAuth_RegisterHandler_WithMessage(t *testing.T) {
	g := newTestGateway()
	g.userClient = &authMessageUserClient{}

	w := httptest.NewRecorder()
	g.registerHandler(w, authJSONRequest("/api/v1/register", `{"email":"test@example.com","password":"password123"}`))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Registration complete")
}

func TestAuth_RegisterHandler_EncodeFailure(t *testing.T) {
	g := newTestGateway()

	w := &authFailingWriter{}
	g.registerHandler(w, authJSONRequest("/api/v1/register", `{"email":"test@example.com","password":"password123"}`))

	assert.Equal(t, http.StatusInternalServerError, w.status)
}

func TestAuth_RegisterWithInviteHandler_GRPCError(t *testing.T) {
	g := newTestGateway()
	g.userClient = &errorUserServiceClient{err: grpcError(codes.Internal, "invite service down")}

	w := httptest.NewRecorder()
	g.registerWithInviteHandler(w, authJSONRequest("/api/v1/register/invite", `{"email":"test@example.com","invite_code":"INV-1"}`))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuth_RegisterWithInviteHandler_WithMessage(t *testing.T) {
	g := newTestGateway()
	g.userClient = &authMessageUserClient{}

	w := httptest.NewRecorder()
	g.registerWithInviteHandler(w, authJSONRequest("/api/v1/register/invite", `{"email":"test@example.com","invite_code":"INV-1"}`))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Invite accepted")
}

func TestAuth_RegisterWithInviteHandler_EncodeFailure(t *testing.T) {
	g := newTestGateway()

	w := &authFailingWriter{}
	g.registerWithInviteHandler(w, authJSONRequest("/api/v1/register/invite", `{"email":"test@example.com","invite_code":"INV-1"}`))

	assert.Equal(t, http.StatusInternalServerError, w.status)
}

func TestAuth_ValidateInviteCodeHandler_GRPCError(t *testing.T) {
	g := newTestGateway()
	g.userClient = &errorUserServiceClient{err: grpcError(codes.NotFound, "user not found")}

	w := httptest.NewRecorder()
	g.validateInviteCodeHandler(w, authJSONRequest("/api/v1/invite/validate", `{"code":"INV-1"}`))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAuth_ValidateInviteCodeHandler_EncodeFailure(t *testing.T) {
	g := newTestGateway()

	w := &authFailingWriter{}
	g.validateInviteCodeHandler(w, authJSONRequest("/api/v1/invite/validate", `{"code":"INV-1"}`))

	assert.Equal(t, http.StatusInternalServerError, w.status)
}

func TestAuth_ConfirmEmailHandler_GRPCError(t *testing.T) {
	g := newTestGateway()
	g.userClient = &errorUserServiceClient{err: grpcError(codes.NotFound, "token not found")}

	w := httptest.NewRecorder()
	g.confirmEmailHandler(w, authJSONRequest("/api/v1/confirm-email", `{"token":"expired"}`))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAuth_ConfirmEmailHandler_EncodeFailure(t *testing.T) {
	g := newTestGateway()

	w := &authFailingWriter{}
	g.confirmEmailHandler(w, authJSONRequest("/api/v1/confirm-email", `{"token":"valid"}`))

	assert.Equal(t, http.StatusInternalServerError, w.status)
}

func TestAuth_EmailConfirmPageHandler_Success(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	repoRoot := filepath.Join(wd, "..", "..")
	if _, statErr := os.Stat(filepath.Join(repoRoot, "web", "dist", "index.html")); statErr != nil {
		t.Skip("web/dist/index.html is not available")
	}
	require.NoError(t, os.Chdir(repoRoot))
	t.Cleanup(func() {
		if cErr := os.Chdir(wd); cErr != nil {
			t.Logf("chdir cleanup error: %v", cErr)
		}
	})


	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/confirm-email?token=valid", nil)

	g.emailConfirmPageHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	assert.NotEmpty(t, w.Body.String())
}

func TestAuth_YandexCallbackHandler_ValidStateMissingCode(t *testing.T) {
	g := newTestGateway()
	withOAuth(g)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/auth/yandex/callback?state=abc", nil)
	req.AddCookie(&http.Cookie{Name: yandexOAuthStateCookie, Value: "abc", Secure: true, HttpOnly: true})

	g.yandexCallbackHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing authorization code")
}

func TestAuth_YandexCallbackHandler_ExchangeThenUserInfoFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(headerContentType, contentTypeJSON)
		if _, err := w.Write([]byte(`{"access_token":"yandex-access","token_type":"bearer","expires_in":3600}`)); err != nil {
			t.Errorf("write token response: %v", err)
		}
	}))
	defer server.Close()

	g := newTestGateway()
	g.userClient = &mockUserServiceClient{}
	g.yandexOAuthConfig = &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  server.URL + "/authorize",
			TokenURL: server.URL + "/token",
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/auth/yandex/callback?state=abc&code=xyz", nil)
	req.AddCookie(&http.Cookie{Name: yandexOAuthStateCookie, Value: "abc", Secure: true, HttpOnly: true})

	g.yandexCallbackHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "failed to get user info")
}

func TestAuth_LoginHandler_IssuesRefreshCookie(t *testing.T) {
	g := authValkeyGateway(t)

	w := httptest.NewRecorder()
	g.loginHandler(w, authJSONRequest("/api/v1/login", `{"email":"test@example.com","password":"password123"}`))

	assert.Equal(t, http.StatusOK, w.Code)

	refreshValue := authCookieValue(t, w, refreshTokenCookieName)
	require.NotEmpty(t, refreshValue)

	userID, err := g.valkeyDB.Get(context.Background(), refreshTokenPrefix+refreshValue).Result()
	require.NoError(t, err)
	assert.Equal(t, "user-123", userID)
}

func TestAuth_LoginHandler_EncodeFailure(t *testing.T) {
	g := authValkeyGateway(t)

	w := &authFailingWriter{status: http.StatusOK}
	g.loginHandler(w, authJSONRequest("/api/v1/login", `{"email":"test@example.com","password":"password123"}`))

	assert.Equal(t, http.StatusOK, w.status)
}

func TestAuth_LoginHandler_TwoFARequiresCode_EncodeFailure(t *testing.T) {
	g := authValkeyGateway(t)
	g.userClient = &totpEnabledUserClient{}

	w := &authFailingWriter{status: http.StatusOK}
	g.loginHandler(w, authJSONRequest("/api/v1/login", `{"email":"test@example.com","password":"password123"}`))

	assert.Equal(t, http.StatusOK, w.status)
}

func TestAuth_LogoutHandler_InvalidatesSession(t *testing.T) {
	g := authGatewayWithSession(t)
	ctx := context.Background()
	require.NoError(t, g.sessionStore.AddUserSession(ctx, "user-123", "session-token", time.Hour))

	w := httptest.NewRecorder()
	g.logoutHandler(w, authUserRequest())

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, authCookieValue(t, w, accessTokenCookieName))
	assert.Empty(t, authCookieValue(t, w, refreshTokenCookieName))

	exists, err := g.valkeyDB.Exists(ctx, "user_sessions:user-123").Result()
	require.NoError(t, err)
	assert.Zero(t, exists)
}

func TestAuth_LogoutHandler_EncodeFailure(t *testing.T) {
	g := newTestGateway()

	w := &authFailingWriter{}
	g.logoutHandler(w, authUserRequest())

	assert.Equal(t, http.StatusOK, w.status)
}

func TestAuth_CriticalSessionHandler_Success(t *testing.T) {
	g := authGatewayWithSession(t)

	w := httptest.NewRecorder()
	g.criticalSessionHandler(w, authUserRequest())

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "critical_session_token")
}

func TestAuth_RequireCriticalSession_MissingHeader(t *testing.T) {
	g := authGatewayWithSession(t)

	err := g.requireCriticalSession(authUserRequest(), "user-123")

	assert.NoError(t, err)
}

func TestAuth_RequireCriticalSession_ValidToken(t *testing.T) {
	g := authGatewayWithSession(t)
	ctx := context.Background()
	token, err := g.sessionStore.CreateCriticalSession(ctx, "user-123")
	require.NoError(t, err)

	req := authUserRequest()
	req.Header.Set("X-Critical-Session-Token", token)

	require.NoError(t, g.requireCriticalSession(req, "user-123"))

	_, getErr := g.valkeyDB.Get(ctx, "critical_session:"+token).Result()
	assert.Error(t, getErr)
}

func TestAuth_SetupTOTPHandler_CriticalSessionRequired(t *testing.T) {
	g := authGatewayWithSession(t)

	req := authUserRequest()
	req.Header.Set("X-Critical-Session-Token", "unknown-token")

	w := httptest.NewRecorder()
	g.setupTOTPHandler(w, req)

	assert.Equal(t, http.StatusPreconditionRequired, w.Code)
}

func TestAuth_SetupTOTPHandler_WithCriticalSession(t *testing.T) {
	g := authGatewayWithSession(t)
	ctx := context.Background()
	token, err := g.sessionStore.CreateCriticalSession(ctx, "user-123")
	require.NoError(t, err)

	req := authUserRequest()
	req.Header.Set("X-Critical-Session-Token", token)

	w := httptest.NewRecorder()
	g.setupTOTPHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "qr_code_url")
}

func TestAuth_SetupTOTPHandler_GRPCError(t *testing.T) {
	g := authGatewayWithSession(t)
	g.userClient = &errorUserServiceClient{err: grpcError(codes.Internal, "totp unavailable")}

	w := httptest.NewRecorder()
	g.setupTOTPHandler(w, authUserRequest())

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuth_SetupTOTPHandler_QRCodeFailure(t *testing.T) {
	g := authGatewayWithSession(t)
	g.userClient = &authBadQRUserClient{}

	w := httptest.NewRecorder()
	g.setupTOTPHandler(w, authUserRequest())

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "qr_code_url")
	assert.Contains(t, w.Body.String(), "code-1")
}

func TestAuth_SetupTOTPHandler_EncodeFailure(t *testing.T) {
	g := authGatewayWithSession(t)

	w := &authFailingWriter{}
	g.setupTOTPHandler(w, authUserRequest())

	assert.Equal(t, http.StatusInternalServerError, w.status)
}

func TestAuth_ConfirmTOTPHandler_Success(t *testing.T) {
	g := authGatewayWithSession(t)

	w := httptest.NewRecorder()
	req := authUserJSONRequest(`{"passcode":"123456","temp_secret":"SECRET","backup_codes":["code-1"]}`)
	g.confirmTOTPHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

func TestAuth_ConfirmTOTPHandler_CriticalSessionRequired(t *testing.T) {
	g := authGatewayWithSession(t)

	req := authUserJSONRequest(`{"passcode":"123456"}`)
	req.Header.Set("X-Critical-Session-Token", "unknown-token")

	w := httptest.NewRecorder()
	g.confirmTOTPHandler(w, req)

	assert.Equal(t, http.StatusPreconditionRequired, w.Code)
}

func TestAuth_ConfirmTOTPHandler_RateLimited(t *testing.T) {
	g := authValkeyGateway(t)
	ctx := context.Background()

	for i := 0; i < totpRateLimitAttempts; i++ {
		require.NoError(t, g.enforceTOTPRateLimit(ctx, "confirm:user-123"))
	}

	w := httptest.NewRecorder()
	g.confirmTOTPHandler(w, authUserJSONRequest(`{"passcode":"123456"}`))

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestAuth_ConfirmTOTPHandler_EncodeFailure(t *testing.T) {
	g := authGatewayWithSession(t)

	w := &authFailingWriter{}
	g.confirmTOTPHandler(w, authUserJSONRequest(`{"passcode":"123456"}`))

	assert.Equal(t, http.StatusInternalServerError, w.status)
}

func TestAuth_DisableTOTPHandler_Success(t *testing.T) {
	g := authGatewayWithSession(t)

	w := httptest.NewRecorder()
	g.disableTOTPHandler(w, authUserJSONRequest(`{"passcode":"123456"}`))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

func TestAuth_DisableTOTPHandler_CriticalSessionRequired(t *testing.T) {
	g := authGatewayWithSession(t)

	req := authUserJSONRequest(`{"passcode":"123456"}`)
	req.Header.Set("X-Critical-Session-Token", "unknown-token")

	w := httptest.NewRecorder()
	g.disableTOTPHandler(w, req)

	assert.Equal(t, http.StatusPreconditionRequired, w.Code)
}

func TestAuth_DisableTOTPHandler_RateLimited(t *testing.T) {
	g := authValkeyGateway(t)
	ctx := context.Background()

	for i := 0; i < totpRateLimitAttempts; i++ {
		require.NoError(t, g.enforceTOTPRateLimit(ctx, "disable:user-123"))
	}

	w := httptest.NewRecorder()
	g.disableTOTPHandler(w, authUserJSONRequest(`{"passcode":"123456"}`))

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestAuth_DisableTOTPHandler_EncodeFailure(t *testing.T) {
	g := authGatewayWithSession(t)

	w := &authFailingWriter{}
	g.disableTOTPHandler(w, authUserJSONRequest(`{"passcode":"123456"}`))

	assert.Equal(t, http.StatusInternalServerError, w.status)
}

func TestAuth_TOTPStatusHandler_ClaimsError(t *testing.T) {
	g := newTestGateway()
	g.userClient = &authClaimsFailingClient{}

	w := httptest.NewRecorder()
	g.totpStatusHandler(w, authUserRequest())

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuth_TOTPStatusHandler_EncodeFailure(t *testing.T) {
	g := newTestGateway()

	w := &authFailingWriter{}
	g.totpStatusHandler(w, authUserRequest())

	assert.Equal(t, http.StatusInternalServerError, w.status)
}

func TestAuth_TOTPStatusHandler_EmptyUserID(t *testing.T) {
	g := newTestGateway()

	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "")
	req := httptest.NewRequestWithContext(ctx, "GET", "/api/v1/totp/status", nil)

	w := httptest.NewRecorder()
	g.totpStatusHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_RefreshHandler_Success(t *testing.T) {
	g := authValkeyGateway(t)
	ctx := context.Background()

	token, err := g.issueRefreshToken(ctx, "user-123")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(ctx, "POST", "/api/v1/refresh", nil)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: token, Secure: true, HttpOnly: true})

	g.refreshHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	newRefresh := authCookieValue(t, w, refreshTokenCookieName)
	require.NotEmpty(t, newRefresh)
	assert.NotEqual(t, token, newRefresh)

	_, getErr := g.valkeyDB.Get(ctx, refreshTokenPrefix+token).Result()
	assert.Error(t, getErr)
}

func TestAuth_RefreshHandler_EncodeFailure(t *testing.T) {
	g := authValkeyGateway(t)
	ctx := context.Background()

	token, err := g.issueRefreshToken(ctx, "user-123")
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(ctx, "POST", "/api/v1/refresh", nil)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: token, Secure: true, HttpOnly: true})

	w := &authFailingWriter{status: http.StatusOK}
	g.refreshHandler(w, req)

	assert.Equal(t, http.StatusOK, w.status)
	assert.Contains(t, w.Header().Get("Set-Cookie"), accessTokenCookieName)
}

func TestAuth_ForgotPasswordHandler_InvalidJSON(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.forgotPasswordHandler(w, authJSONRequest("/api/v1/auth/forgot-password", `{invalid`))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuth_ForgotPasswordHandler_MissingEmail(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.forgotPasswordHandler(w, authJSONRequest("/api/v1/auth/forgot-password", `{}`))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuth_ForgotPasswordHandler_Success(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.forgotPasswordHandler(w, authJSONRequest("/api/v1/auth/forgot-password", `{"email":"test@example.com"}`))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "debug_code")
}

func TestAuth_ForgotPasswordHandler_EncodeFailure(t *testing.T) {
	g := newTestGateway()

	w := &authFailingWriter{status: http.StatusOK}
	g.forgotPasswordHandler(w, authJSONRequest("/api/v1/auth/forgot-password", `{"email":"test@example.com"}`))

	assert.Equal(t, http.StatusOK, w.status)
}

func TestAuth_ResetPasswordHandler_InvalidJSON(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.resetPasswordHandler(w, authJSONRequest("/api/v1/auth/reset", `{invalid`))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuth_ResetPasswordHandler_MissingFields(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.resetPasswordHandler(w, authJSONRequest("/api/v1/auth/reset", `{"email":"test@example.com"}`))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuth_ResetPasswordHandler_ShortPassword(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.resetPasswordHandler(w, authJSONRequest("/api/v1/auth/reset", `{"email":"test@example.com","code":"123456","new_password":"12345"}`))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuth_ResetPasswordHandler_Success(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.resetPasswordHandler(w, authJSONRequest("/api/v1/auth/reset", `{"email":"test@example.com","code":"123456","new_password":"newpassword"}`))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestAuth_EncodeQRCodeBase64_InvalidURL(t *testing.T) {
	out, err := encodeQRCodeBase64("://bad-qr")

	assert.Error(t, err)
	assert.Empty(t, out)
}

func TestAuth_EncodeQRCodeBase64_ValidURL(t *testing.T) {
	out, err := encodeQRCodeBase64("otpauth://totp/FitPulse:user@example.com?algorithm=SHA1&digits=6&issuer=FitPulse&period=30&secret=JBSWY3DPEHPK3PXP")

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(out, "data:image/png;base64,"))
}

func TestAuth_IssueJWT_ClaimsError(t *testing.T) {
	g := newTestGateway()
	g.userClient = &authClaimsFailingClient{}

	token, err := g.issueJWT(context.Background(), "user-123")

	assert.Empty(t, token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query user for JWT")
}

func TestAuth_IssueJWT_TokenProviderError(t *testing.T) {
	g := newTestGateway()
	g.tokenProvider = authBrokenTokenProvider{}

	token, err := g.issueJWT(context.Background(), "user-123")

	assert.Empty(t, token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "issue jwt")
}

func TestAuth_IssueJWT_Success(t *testing.T) {
	g := newTestGateway()

	token, err := g.issueJWT(context.Background(), "user-123")

	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestAuth_UserTOTPEnabled_ClaimsError(t *testing.T) {
	g := newTestGateway()
	g.userClient = &authClaimsFailingClient{}

	assert.False(t, g.userTOTPEnabled(context.Background(), "user-123"))
}

func TestAuth_EnforceTOTPRateLimit_ValkeyBackend(t *testing.T) {
	g := authValkeyGateway(t)
	ctx := context.Background()

	for i := 0; i < totpRateLimitAttempts; i++ {
		require.NoError(t, g.enforceTOTPRateLimit(ctx, "valkey:user-123"))
	}

	err := g.enforceTOTPRateLimit(ctx, "valkey:user-123")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many 2FA attempts")
}

func TestAuth_RotateRefreshToken_Success(t *testing.T) {
	g := authValkeyGateway(t)
	ctx := context.Background()

	token, err := g.issueRefreshToken(ctx, "user-123")
	require.NoError(t, err)

	access, refresh, err := g.rotateRefreshToken(ctx, token)
	require.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, refresh)

	_, getErr := g.valkeyDB.Get(ctx, refreshTokenPrefix+token).Result()
	assert.Error(t, getErr)

	revoked, err := g.valkeyDB.SIsMember(ctx, refreshRevokedPrefix+"user-123", g.tokenProvider.ComputeTokenFingerprint(token)).Result()
	require.NoError(t, err)
	assert.True(t, revoked)
}

func TestAuth_RotateRefreshToken_ReuseDetected_Valkey(t *testing.T) {
	g := authValkeyGateway(t)
	ctx := context.Background()

	oldToken := "reused-refresh-token"
	fingerprint := g.tokenProvider.ComputeTokenFingerprint(oldToken)

	require.NoError(t, g.valkeyDB.Set(ctx, refreshFingerprintPrefix+fingerprint, "user-123", time.Hour).Err())
	require.NoError(t, g.valkeyDB.SAdd(ctx, refreshRevokedPrefix+"user-123", fingerprint).Err())

	access, refresh, err := g.rotateRefreshToken(ctx, oldToken)

	assert.Empty(t, access)
	assert.Empty(t, refresh)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reuse detected")
}

func TestAuth_InvalidateAllUserSessions_WithSessionStore(t *testing.T) {
	g := authGatewayWithSession(t)
	ctx := context.Background()

	require.NoError(t, g.valkeyDB.Set(ctx, refreshIssuedPrefix+"user-123", "token-1", time.Hour).Err())
	require.NoError(t, g.sessionStore.AddUserSession(ctx, "user-123", "session-token", time.Hour))

	g.invalidateAllUserSessions(ctx, "user-123")

	_, issuedErr := g.valkeyDB.Get(ctx, refreshIssuedPrefix+"user-123").Result()
	assert.Error(t, issuedErr)

	_, sessionErr := g.valkeyDB.Get(ctx, "user_sessions:user-123").Result()
	assert.Error(t, sessionErr)
}

func TestAuth_VerifyTOTPHandler_SuccessEstablishesSession(t *testing.T) {
	g := authValkeyGateway(t)
	ctx := context.Background()
	require.NoError(t, g.valkeyDB.Set(ctx, twoFATempPrefix+"temp", "user-123", 5*time.Minute).Err())

	w := httptest.NewRecorder()
	g.verifyTOTPHandler(w, authJSONRequest("/api/v1/totp/verify", `{"temp_token":"temp","passcode":"123456","is_backup_code":false}`))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "backup_codes_remaining")
	assert.NotEmpty(t, authCookieValue(t, w, accessTokenCookieName))
	assert.NotEmpty(t, authCookieValue(t, w, refreshTokenCookieName))

	_, tempErr := g.valkeyDB.Get(ctx, twoFATempPrefix+"temp").Result()
	assert.Error(t, tempErr)
}

func TestAuth_VerifyTOTPHandler_BackupCode(t *testing.T) {
	g := authValkeyGateway(t)
	ctx := context.Background()
	require.NoError(t, g.valkeyDB.Set(ctx, twoFATempPrefix+"temp", "user-123", 5*time.Minute).Err())

	w := httptest.NewRecorder()
	g.verifyTOTPHandler(w, authJSONRequest("/api/v1/totp/verify", `{"temp_token":"temp","passcode":"backup-code","is_backup_code":true}`))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, authCookieValue(t, w, refreshTokenCookieName))
}

func TestAuth_VerifyTOTPHandler_InvalidPasscode(t *testing.T) {
	g := authValkeyGateway(t)
	g.userClient = &authInvalidTOTPClient{}
	ctx := context.Background()
	require.NoError(t, g.valkeyDB.Set(ctx, twoFATempPrefix+"temp", "user-123", 5*time.Minute).Err())

	w := httptest.NewRecorder()
	g.verifyTOTPHandler(w, authJSONRequest("/api/v1/totp/verify", `{"temp_token":"temp","passcode":"000000"}`))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid TOTP code")
}

func TestAuth_VerifyTOTPHandler_RateLimited_Valkey(t *testing.T) {
	g := authValkeyGateway(t)
	ctx := context.Background()
	require.NoError(t, g.valkeyDB.Set(ctx, twoFATempPrefix+"temp", "user-123", 5*time.Minute).Err())

	for i := 0; i < totpRateLimitAttempts; i++ {
		require.NoError(t, g.enforceTOTPRateLimit(ctx, "verify:user-123"))
	}

	w := httptest.NewRecorder()
	g.verifyTOTPHandler(w, authJSONRequest("/api/v1/totp/verify", `{"temp_token":"temp","passcode":"123456"}`))

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestAuth_VerifyTOTPHandler_TokenIssueFailure(t *testing.T) {
	g := authValkeyGateway(t)
	g.tokenProvider = authBrokenTokenProvider{}
	ctx := context.Background()
	require.NoError(t, g.valkeyDB.Set(ctx, twoFATempPrefix+"temp", "user-123", 5*time.Minute).Err())

	w := httptest.NewRecorder()
	g.verifyTOTPHandler(w, authJSONRequest("/api/v1/totp/verify", `{"temp_token":"temp","passcode":"123456"}`))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuth_VerifyTOTPHandler_EncodeFailure(t *testing.T) {
	g := authValkeyGateway(t)
	ctx := context.Background()
	require.NoError(t, g.valkeyDB.Set(ctx, twoFATempPrefix+"temp", "user-123", 5*time.Minute).Err())

	w := &authFailingWriter{status: http.StatusOK}
	g.verifyTOTPHandler(w, authJSONRequest("/api/v1/totp/verify", `{"temp_token":"temp","passcode":"123456"}`))

	assert.Equal(t, http.StatusOK, w.status)
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

	for i := 0; i < totpRateLimitAttempts; i++ {

		require.NoError(t, g.enforceTOTPRateLimit(ctx, "setup:user-123"))

	}

	err := g.enforceTOTPRateLimit(ctx, "setup:user-123")

	require.Error(t, err)

	assert.Contains(t, err.Error(), "too many 2FA attempts")

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

	for i := 0; i < totpRateLimitAttempts; i++ {

		require.NoError(t, g.enforceTOTPRateLimit(ctx, "user-456"))

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

	for i := 0; i < totpRateLimitAttempts; i++ {

		require.NoError(t, g.enforceTOTPRateLimit(ctx, "verify:user-123"))

	}

	err = g.enforceTOTPRateLimit(ctx, "verify:user-123")

	require.Error(t, err)

	assert.Contains(t, err.Error(), "too many 2FA attempts")

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

type authVerifyFailingClient struct {
	totpEnabledUserClient
}

func (m *authVerifyFailingClient) VerifyTOTP(ctx context.Context, req *user.VerifyTOTPRequest, opts ...grpc.CallOption) (*user.VerifyTOTPResponse, error) {
	return nil, grpcError(codes.Internal, "verify unavailable")
}

func TestAuth_EmailConfirmPageHandler_WriteFailure(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	repoRoot := filepath.Join(wd, "..", "..")
	if _, statErr := os.Stat(filepath.Join(repoRoot, "web", "dist", "index.html")); statErr != nil {
		t.Skip("web/dist/index.html is not available")
	}
	require.NoError(t, os.Chdir(repoRoot))
	t.Cleanup(func() {
		if cErr := os.Chdir(wd); cErr != nil {
			t.Logf("chdir cleanup error: %v", cErr)
		}
	})


	g := newTestGateway()

	w := &authFailingWriter{}
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/confirm-email?token=valid", nil)

	g.emailConfirmPageHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.status)
}

func TestAuth_VerifyTOTPHandler_VerifyUpstreamError(t *testing.T) {
	g := authValkeyGateway(t)
	g.userClient = &authVerifyFailingClient{}
	ctx := context.Background()
	require.NoError(t, g.valkeyDB.Set(ctx, twoFATempPrefix+"temp", "user-123", 5*time.Minute).Err())

	w := httptest.NewRecorder()
	g.verifyTOTPHandler(w, authJSONRequest("/api/v1/totp/verify", `{"temp_token":"temp","passcode":"123456"}`))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuth_RefreshHandler_InvalidRefreshToken(t *testing.T) {
	g := authValkeyGateway(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/refresh", nil)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: "unknown-token", Secure: true, HttpOnly: true})

	g.refreshHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
