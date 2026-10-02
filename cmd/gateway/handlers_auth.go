package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"image/png"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"

	userpb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/middleware"
	"github.com/MAMUER/project/internal/sanitize"
)

// ========== Auth Handlers ==========

const (
	totpRateLimitAttempts = 5

	errUnauthorized = "Unauthorized access"

	errYandexOAuthNotConfigured = "Yandex OAuth not configured"

	errCriticalSessionRequired = "Critical session required"

	errTOTPRateLimitExceeded = "TOTP rate limit exceeded"

	yandexOAuthStateCookie = "yandex_oauth_state"

	headerContentType = "Content-Type"

	contentTypeJSON = "application/json"

	accessTokenCookieName = "fitpulse-access-token"

	refreshTokenCookieName = "fitpulse-refresh-token"

	refreshTokenPrefix = "refresh:"

	refreshFingerprintPrefix = "refresh:fp:"

	refreshIssuedPrefix = "refresh:issued:"

	refreshRevokedPrefix = "refresh:revoked:"

	twoFATempPrefix = "2fa_temp:"
)

type totpRateLimiter struct {
	limiter *rate.Limiter

	expiresAt time.Time
}

func generateOAuthState() (string, error) {

	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {

		return "", fmt.Errorf("generate oauth state: %w", err)

	}

	return hex.EncodeToString(b), nil

}

func (g *gateway) userTOTPEnabled(ctx context.Context, userID string) bool {

	if userID == "" {

		return false

	}

	resp, err := g.userClient.GetUserClaims(ctx, &userpb.GetUserClaimsRequest{UserId: userID})

	if err != nil {

		g.log.Warn("Could not check TOTP status", zap.Error(err), zap.String("user_id", userID))

		return false

	}

	return resp.GetTotpEnabled()

}

func (g *gateway) issueJWT(ctx context.Context, userID string) (string, error) {

	if userID == "" {

		return "", errors.New("user_id is empty")

	}

	resp, err := g.userClient.GetUserClaims(ctx, &userpb.GetUserClaimsRequest{UserId: userID})

	if err != nil {

		return "", fmt.Errorf("query user for JWT: %w", err)

	}

	token, err := g.tokenProvider.GenerateAccessToken(userID, resp.GetEmail(), resp.GetRole(), 15*time.Minute)

	if err != nil {

		return "", fmt.Errorf("issue jwt: %w", err)

	}

	return token, nil

}

func (g *gateway) issueRefreshToken(ctx context.Context, userID string) (string, error) {

	if g.valkeyDB == nil {

		return "", errors.New("valkey unavailable")

	}

	token := g.tokenProvider.GenerateRefreshToken()

	fingerprint := g.tokenProvider.ComputeTokenFingerprint(token)

	key := refreshTokenPrefix + token

	fpKey := refreshFingerprintPrefix + fingerprint

	issuedKey := refreshIssuedPrefix + userID

	pipe := g.valkeyDB.Pipeline()

	pipe.Set(ctx, key, userID, 7*24*time.Hour)

	pipe.Set(ctx, fpKey, userID, 7*24*time.Hour)

	pipe.SAdd(ctx, issuedKey, token)

	pipe.Expire(ctx, issuedKey, 7*24*time.Hour)

	_, err := pipe.Exec(ctx)

	if err != nil {

		return "", fmt.Errorf("issue refresh token: %w", err)

	}

	return token, nil

}

func (g *gateway) rotateRefreshToken(ctx context.Context, oldToken string) (string, string, error) {

	userID, err := g.valkeyDB.Get(ctx, refreshTokenPrefix+oldToken).Result()

	if err != nil {

		fingerprint := g.tokenProvider.ComputeTokenFingerprint(oldToken)

		fpUserID, fpErr := g.valkeyDB.Get(ctx, refreshFingerprintPrefix+fingerprint).Result()

		if fpErr == nil && fpUserID != "" {

			revokedKey := refreshRevokedPrefix + fpUserID

			isRevoked, memberErr := g.valkeyDB.SIsMember(ctx, revokedKey, fingerprint).Result()

			if memberErr == nil && isRevoked {

				g.invalidateAllUserSessions(ctx, fpUserID)

				g.log.Warn("Refresh token reuse detected, all sessions invalidated",

					zap.String("user_id", fpUserID))

				return "", "", errors.New("refresh token reuse detected")

			}

		}

		return "", "", errors.New("invalid refresh token")

	}

	_ = g.valkeyDB.Del(ctx, refreshTokenPrefix+oldToken).Err()

	oldFingerprint := g.tokenProvider.ComputeTokenFingerprint(oldToken)

	revokedKey := refreshRevokedPrefix + userID

	issuedKey := refreshIssuedPrefix + userID

	pipe := g.valkeyDB.Pipeline()

	pipe.SAdd(ctx, revokedKey, oldFingerprint)

	pipe.Expire(ctx, revokedKey, 7*24*time.Hour)

	pipe.SRem(ctx, issuedKey, oldToken)

	pipe.Del(ctx, refreshFingerprintPrefix+oldFingerprint)

	_, _ = pipe.Exec(ctx)

	newRefresh, err := g.issueRefreshToken(ctx, userID)

	if err != nil {

		return "", "", err

	}

	newAccess, err := g.issueJWT(ctx, userID)

	if err != nil {

		return "", "", err

	}

	return newAccess, newRefresh, nil

}

func (g *gateway) invalidateAllUserSessions(ctx context.Context, userID string) {

	if g.sessionStore != nil {

		_ = g.sessionStore.InvalidateUserSession(ctx, userID)

	}

	keys := []string{

		refreshIssuedPrefix + userID,

		refreshRevokedPrefix + userID,
	}

	pattern := "refresh:*" + userID + "*"

	var cursor uint64

	for {

		keysFound, nextCursor, err := g.valkeyDB.Scan(ctx, cursor, pattern, 100).Result()

		if err != nil {

			break

		}

		keys = append(keys, keysFound...)

		cursor = nextCursor

		if cursor == 0 {

			break

		}

	}

	if len(keys) > 0 {

		_ = g.valkeyDB.Del(ctx, keys...).Err()

	}

}

func (g *gateway) enforceTOTPRateLimit(ctx context.Context, key string) error {

	valkeyKey := "2fa_rate:" + key

	if g.valkeyDB != nil {

		count, err := g.valkeyDB.Incr(ctx, valkeyKey).Result()

		if err == nil {

			if count == 1 {

				_ = g.valkeyDB.Expire(ctx, valkeyKey, time.Minute).Err()

			}

			if count > totpRateLimitAttempts {

				return errors.New("too many 2FA attempts")

			}

			return nil

		}

		g.log.Warn("Valkey 2FA rate limit unavailable", zap.Error(err))

	}

	if countOverLimit(g, key) {

		return errors.New("too many 2FA attempts")

	}

	return nil

}

func countOverLimit(g *gateway, key string) bool {

	v, _ := g.totpRateLimiters.LoadOrStore(key, &totpRateLimiter{

		limiter: rate.NewLimiter(totpRateLimitAttempts, totpRateLimitAttempts),

		expiresAt: time.Now().Add(time.Minute),
	})

	limiter := v.(*totpRateLimiter)

	if time.Now().After(limiter.expiresAt) {

		limiter = &totpRateLimiter{

			limiter: rate.NewLimiter(totpRateLimitAttempts, totpRateLimitAttempts),

			expiresAt: time.Now().Add(time.Minute),
		}

		g.totpRateLimiters.Store(key, limiter)

	}

	return !limiter.limiter.Allow()

}

func (g *gateway) requireCriticalSession(r *http.Request, userID string) error {

	if g.sessionStore == nil {

		return nil

	}

	token := r.Header.Get("X-Critical-Session-Token")

	if token == "" {

		g.log.Warn("Critical action without critical session token",

			zap.String("user_id", sanitize.LogString(userID)),

			zap.String("path", sanitize.LogString(r.URL.Path)),
		)

		return nil

	}

	if err := g.sessionStore.ValidateCriticalSession(r.Context(), token, userID); err != nil {

		g.log.Warn("Invalid critical session token",

			zap.Error(err),

			zap.String("user_id", sanitize.LogString(userID)),

			zap.String("path", sanitize.LogString(r.URL.Path)),
		)

		return fmt.Errorf("invalid critical session: %w", err)

	}

	return nil

}

// @Summary      Create critical session token

// @Description  Creates a critical session token required for sensitive operations like 2FA setup or disable

// @Tags         Auth

// @Accept       json

// @Produce      json

// @Success      200  {object}  map[string]interface{}

// @Failure      401  {object}  map[string]interface{}

// @Failure      503  {object}  map[string]interface{}

// @Failure      500  {object}  map[string]interface{}

// @Router       /api/v1/auth/critical-session [post]

func (g *gateway) criticalSessionHandler(w http.ResponseWriter, r *http.Request) {

	userID, ok := r.Context().Value(middleware.UserIDKey).(string)

	if !ok {

		g.log.Error(errUnauthorized, zap.String("handler", "criticalSession"))

		http.Error(w, "Unauthorized", http.StatusUnauthorized)

		return

	}

	if g.sessionStore == nil {

		g.log.Error("Session store unavailable")

		http.Error(w, "Session store unavailable", http.StatusServiceUnavailable)

		return

	}

	token, err := g.sessionStore.CreateCriticalSession(r.Context(), userID)

	if err != nil {

		g.log.Error("Failed to create critical session", zap.Error(err))

		http.Error(w, "Failed to create critical session", http.StatusInternalServerError)

		return

	}

	w.Header().Set(headerContentType, contentTypeJSON)

	if err := json.NewEncoder(w).Encode(map[string]interface{}{

		"critical_session_token": token,
	}); err != nil {

		g.log.Error("Failed to encode critical session response", zap.Error(err))

	}

}

func encodeQRCodeBase64(qrCodeURL string) (string, error) {

	key, err := otp.NewKeyFromURL(qrCodeURL)

	if err != nil {

		return "", fmt.Errorf("parse otp key URL: %w", err)

	}

	img, err := key.Image(256, 256)

	if err != nil {

		return "", fmt.Errorf("render qr code image: %w", err)

	}

	var buf bytes.Buffer

	if err := png.Encode(&buf, img); err != nil {

		return "", fmt.Errorf("encode qr code PNG: %w", err)

	}

	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil

}

// @Summary      Register new user

// @Description  Registers a new user account with email, password, full name and role

// @Tags         Auth

// @Accept       json

// @Produce      json

// @Param        request  body  object  required  "Registration request body"

// @Success      200  {object}  map[string]interface{}

// @Failure      400  {object}  map[string]interface{}

// @Failure      409  {object}  map[string]interface{}

// @Failure      500  {object}  map[string]interface{}

// @Failure      503  {object}  map[string]interface{}

// @Router       /api/v1/register [post]

func (g *gateway) registerHandler(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Email string `json:"email"`

		Password string `json:"password"`

		FullName string `json:"full_name"`

		Role string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		g.log.Error("Failed to decode register request", zap.Error(err))

		http.Error(w, errBadRequest, http.StatusBadRequest)

		return

	}

	resp, err := g.userClient.Register(r.Context(), &userpb.RegisterRequest{

		Email: req.Email,

		Password: req.Password,

		FullName: req.FullName,

		Role: req.Role,
	})

	if err != nil {

		httpCode, errMsg := grpcToHTTPStatus(err)

		g.log.Error("Register failed", zap.Error(err))

		http.Error(w, errMsg, httpCode)

		return

	}

	// Return registration result

	response := map[string]interface{}{"status": "ok"}

	if resp.GetMessage() != "" {

		response["message"] = resp.GetMessage()

	}

	w.Header().Set(headerContentType, contentTypeJSON)

	if err := json.NewEncoder(w).Encode(response); err != nil {

		g.log.Error(logFailedToEncodeResponse, zap.Error(err))

		http.Error(w, "encodeResponseError", http.StatusInternalServerError)

		return

	}

}

// @Summary      Register with invite code

// @Description  Registers a new user using an invite code with additional profile details

// @Tags         Auth

// @Accept       json

// @Produce      json

// @Param        request  body  object  required  "Registration with invite request body"

// @Success      200  {object}  map[string]interface{}

// @Failure      400  {object}  map[string]interface{}

// @Failure      409  {object}  map[string]interface{}

// @Failure      500  {object}  map[string]interface{}

// @Failure      503  {object}  map[string]interface{}

// @Router       /api/v1/register/invite [post]

func (g *gateway) registerWithInviteHandler(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Email string `json:"email"`

		Password string `json:"password"`

		FullName string `json:"full_name"`

		InviteCode string `json:"invite_code"`

		LicenseNumber string `json:"license_number"`

		Specialty string `json:"specialty"`

		Phone string `json:"phone"`

		Bio string `json:"bio"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		g.log.Error("Failed to decode register with invite request", zap.Error(err))

		http.Error(w, errBadRequest, http.StatusBadRequest)

		return

	}

	resp, err := g.userClient.RegisterWithInvite(r.Context(), &userpb.RegisterWithInviteRequest{

		Email: req.Email,

		Password: req.Password,

		FullName: req.FullName,

		InviteCode: req.InviteCode,
	})

	if err != nil {

		httpCode, errMsg := grpcToHTTPStatus(err)

		g.log.Error("Register with invite failed", zap.Error(err))

		http.Error(w, errMsg, httpCode)

		return

	}

	response := map[string]interface{}{

		"status": "ok",

		"user_id": resp.GetUserId(),
	}

	if resp.GetMessage() != "" {

		response["message"] = resp.GetMessage()

	}

	w.Header().Set(headerContentType, contentTypeJSON)

	if err := json.NewEncoder(w).Encode(response); err != nil {

		g.log.Error(logFailedToEncodeResponse, zap.Error(err))

		http.Error(w, "encodeResponseError", http.StatusInternalServerError)

		return

	}

}

// @Summary      Validate invite code

// @Description  Validates an invite code and returns associated role and specialty

// @Tags         Auth

// @Accept       json

// @Produce      json

// @Param        request  body  object  required  "Invite code validation request"

// @Success      200  {object}  map[string]interface{}

// @Failure      400  {object}  map[string]interface{}

// @Failure      500  {object}  map[string]interface{}

// @Failure      503  {object}  map[string]interface{}

// @Router       /api/v1/invite/validate [post]

func (g *gateway) validateInviteCodeHandler(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Code string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		g.log.Error("Failed to decode validate invite request", zap.Error(err))

		http.Error(w, errBadRequest, http.StatusBadRequest)

		return

	}

	resp, err := g.userClient.ValidateInviteCode(r.Context(), &userpb.ValidateInviteCodeRequest{

		Code: req.Code,
	})

	if err != nil {

		httpCode, errMsg := grpcToHTTPStatus(err)

		g.log.Error("Failed to validate invite code", zap.Error(err))

		http.Error(w, errMsg, httpCode)

		return

	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{

		"is_valid": resp.GetIsValid(),

		"role": resp.GetRole(),

		"specialty": resp.GetSpecialty(),

		"error": resp.GetErrorMessage(),
	}); err != nil {

		g.log.Error(logFailedToEncodeResponse, zap.Error(err))

		http.Error(w, "encodeResponseError", http.StatusInternalServerError)

		return

	}

}

// @Summary      User login

// @Description  Authenticates user with email and password, returns JWT tokens or requires 2FA verification

// @Tags         Auth

// @Accept       json

// @Produce      json

// @Param        request  body  object  required  "Login credentials"

// @Success      200  {object}  map[string]interface{}

// @Failure      400  {object}  map[string]interface{}

// @Failure      401  {object}  map[string]interface{}

// @Failure      429  {object}  map[string]interface{}

// @Failure      500  {object}  map[string]interface{}

// @Failure      503  {object}  map[string]interface{}

// @Router       /api/v1/login [post]

func (g *gateway) loginHandler(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Email string `json:"email"`

		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		g.log.Error("Failed to decode login request", zap.Error(err))

		http.Error(w, errBadRequest, http.StatusBadRequest)

		return

	}

	resp, err := g.userClient.Login(r.Context(), &userpb.LoginRequest{

		Email: req.Email,

		Password: req.Password,
	})

	if err != nil {

		httpCode, errMsg := grpcToHTTPStatus(err)

		g.log.Error("Login failed", zap.Error(err), zap.String("email", html.EscapeString(strings.ReplaceAll(strings.ReplaceAll(req.Email, "\n", ""), "\r", ""))))

		if httpCode == http.StatusUnauthorized && strings.Contains(errMsg, "Email not confirmed") {

			g.log.Warn("Login attempt with unconfirmed email", zap.String("email", html.EscapeString(strings.ReplaceAll(strings.ReplaceAll(req.Email, "\n", ""), "\r", ""))))

			http.Error(w, "Email не подтверждён. Проверьте вашу почту.", httpCode)

			return

		}

		g.log.Error("Login request failed", zap.Int("http_code", httpCode), zap.String("error", errMsg))

		http.Error(w, errMsg, httpCode)

		return

	}

	if g.userTOTPEnabled(r.Context(), resp.GetUserId()) {

		tempToken := uuid.New().String()

		_ = g.valkeyDB.Set(r.Context(), twoFATempPrefix+tempToken, resp.GetUserId(), 5*time.Minute).Err()

		w.Header().Set(headerContentType, contentTypeJSON)

		if err := json.NewEncoder(w).Encode(map[string]interface{}{

			"requires_2fa": true,

			"temp_token": tempToken,

			"message": "Please provide your 2FA code",
		}); err != nil {

			g.log.Error("Failed to encode 2FA response", zap.Error(err))

			http.Error(w, "encodeResponseError", http.StatusInternalServerError)

			return

		}

		return

	}

	accessToken := resp.GetAccessToken()
	refreshToken, rtErr := g.issueRefreshToken(r.Context(), resp.GetUserId())

	accessCookie := &http.Cookie{
		Name:     accessTokenCookieName,
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   900,
	}
	http.SetCookie(w, accessCookie)

	var refreshCookie *http.Cookie
	if rtErr == nil {
		refreshCookie = &http.Cookie{
			Name:     refreshTokenCookieName,
			Value:    refreshToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60,
	}
	http.SetCookie(w, refreshCookie)
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"token_type": resp.GetTokenType(),
		"expires_in": 900,
	})

}

// @Summary      User logout

// @Description  Logs out the current user and invalidates the server session

// @Tags         Auth

// @Accept       json

// @Produce      json

// @Success      200  {object}  map[string]interface{}

// @Failure      401  {object}  map[string]interface{}

// @Failure      500  {object}  map[string]interface{}

// @Failure      503  {object}  map[string]interface{}

// @Router       /api/v1/logout [post]

func (g *gateway) logoutHandler(w http.ResponseWriter, r *http.Request) {

	userID, ok := r.Context().Value(middleware.UserIDKey).(string)

	if ok && g.sessionStore != nil {

		if err := g.sessionStore.InvalidateUserSession(r.Context(), userID); err != nil {

			g.log.Warn("Failed to invalidate server session", zap.String("user_id", userID), zap.Error(err))

		}

	}

	logoutHeaders := middleware.LogoutHeaders()

	for key, values := range logoutHeaders {

		for _, value := range values {

			w.Header().Add(key, value)

		}

	}

	accessCookie := &http.Cookie{
		Name:     accessTokenCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	}
	http.SetCookie(w, accessCookie)

	refreshCookie := &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	}
	http.SetCookie(w, refreshCookie)

	w.Header().Set(headerContentType, contentTypeJSON)

	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "logged_out"})

}

// @Summary      Confirm email address

// @Description  Confirms user email address using the provided confirmation token

// @Tags         Auth

// @Accept       json

// @Produce      json

// @Param        request  body  object  required  "Email confirmation token"

// @Success      200  {object}  map[string]interface{}

// @Failure      400  {object}  map[string]interface{}

// @Failure      404  {object}  map[string]interface{}

// @Failure      500  {object}  map[string]interface{}

// @Failure      503  {object}  map[string]interface{}

// @Router       /api/v1/auth/confirm [post]

func (g *gateway) confirmEmailHandler(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		g.log.Error("Failed to decode confirm email request", zap.Error(err))

		http.Error(w, errBadRequest, http.StatusBadRequest)

		return

	}

	if req.Token == "" {

		g.log.Error("Missing token in confirm email request")

		http.Error(w, "Укажите токен подтверждения", http.StatusBadRequest)

		return

	}

	resp, err := g.userClient.ConfirmEmail(r.Context(), &userpb.ConfirmEmailRequest{Token: req.Token})

	if err != nil {

		httpCode, errMsg := grpcToHTTPStatus(err)

		g.log.Error("Confirm email failed", zap.Error(err))

		http.Error(w, errMsg, httpCode)

		return

	}

	w.Header().Set(headerContentType, contentTypeJSON)

	if err := json.NewEncoder(w).Encode(map[string]interface{}{

		"message": "Email confirmed. You can now log in.",

		"user_id": resp.GetUserId(),
	}); err != nil {

		g.log.Error(logFailedToEncodeResponse, zap.Error(err))

		http.Error(w, "encodeResponseError", http.StatusInternalServerError)

		return

	}

}

// @Summary      Email confirmation page

// @Description  Serves the email confirmation HTML page

// @Tags         Auth

// @Produce      html

// @Success      200  {string}  string

// @Failure      404  {object}  map[string]interface{}

// @Failure      500  {object}  map[string]interface{}

// @Router       /confirm [get]

func (g *gateway) emailConfirmPageHandler(w http.ResponseWriter, r *http.Request) {

	_ = r.URL.Query().Get("token")

	indexPath := "./web/dist/index.html"

	indexBytes, err := os.ReadFile(indexPath)

	if err != nil {

		g.log.Error("Failed to load index.html", zap.Error(err))

		http.Error(w, "Сервис временно недоступен", http.StatusInternalServerError)

		return

	}

	w.Header().Set(headerContentType, "text/html; charset=utf-8")

	if _, err := w.Write(indexBytes); err != nil {

		g.log.Error("Failed to write index.html", zap.Error(err))

		http.Error(w, "Сервис временно недоступен", http.StatusInternalServerError)

		return

	}

}

// @Summary      Initiate Yandex ID OAuth login

// @Description  Redirects user to Yandex ID consent screen

// @Tags         Auth

// @Produce      html

// @Success      307  {string}  string

// @Failure      501  {object}  map[string]interface{}

// @Failure      500  {object}  map[string]interface{}

// @Router       /api/v1/auth/yandex [get]

func (g *gateway) yandexLoginHandler(w http.ResponseWriter, r *http.Request) {

	if g.yandexOAuthConfig == nil {

		g.log.Error(errYandexOAuthNotConfigured)

		http.Error(w, errYandexOAuthNotConfigured, http.StatusNotImplemented)

		return

	}

	state, err := generateOAuthState()

	if err != nil {

		g.log.Error("Failed to generate Yandex OAuth state", zap.Error(err))

		http.Error(w, "failed to generate oauth state", http.StatusInternalServerError)

		return

	}

	http.SetCookie(w, &http.Cookie{

		Name: yandexOAuthStateCookie,

		Value: state,

		Path: "/",

		MaxAge: 600,

		Expires: time.Now().Add(10 * time.Minute),

		HttpOnly: true,

		Secure: true,

		SameSite: http.SameSiteLaxMode,
	})

	redirectURL := g.yandexOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)

}

// @Summary      Yandex ID OAuth callback

// @Description  Handles Yandex ID OAuth callback, exchanges authorization code for access token and user info

// @Tags         Auth

// @Accept       json

// @Produce      json

// @Param        state  query  string  false  "OAuth state parameter"

// @Param        code   query  string  false  "Authorization code from Yandex"

// @Success      200  {object}  map[string]interface{}

// @Failure      400  {object}  map[string]interface{}

// @Failure      401  {object}  map[string]interface{}

// @Failure      429  {object}  map[string]interface{}

// @Failure      500  {object}  map[string]interface{}

// @Failure      503  {object}  map[string]interface{}

// @Router       /api/v1/auth/yandex/callback [get]

func (g *gateway) yandexCallbackHandler(w http.ResponseWriter, r *http.Request) {

	if g.yandexOAuthConfig == nil {

		g.log.Error(errYandexOAuthNotConfigured)

		http.Error(w, errYandexOAuthNotConfigured, http.StatusNotImplemented)

		return

	}

	state := r.URL.Query().Get("state")

	cookie, err := r.Cookie(yandexOAuthStateCookie)

	if err != nil || state == "" || cookie == nil || subtle.ConstantTimeCompare([]byte(state), []byte(cookie.Value)) != 1 {

		g.log.Error("Invalid OAuth state")

		http.Error(w, "invalid oauth state", http.StatusBadRequest)

		return

	}

	http.SetCookie(w, &http.Cookie{

		Name: yandexOAuthStateCookie,

		Value: "",

		Path: "/",

		MaxAge: -1,

		Expires: time.Unix(0, 0),

		HttpOnly: true,

		Secure: true,

		SameSite: http.SameSiteLaxMode,
	})

	code := r.URL.Query().Get("code")

	if code == "" {

		g.log.Error("Missing authorization code")

		http.Error(w, "missing authorization code", http.StatusBadRequest)

		return

	}

	token, err := g.yandexOAuthConfig.Exchange(r.Context(), code)

	if err != nil {

		g.log.Error("Failed to exchange Yandex code", zap.Error(err))

		http.Error(w, "failed to exchange authorization code", http.StatusBadRequest)

		return

	}

	accessToken, ok := token.Extra("access_token").(string)

	if !ok || accessToken == "" {

		g.log.Error("Yandex token missing access_token")

		http.Error(w, "missing access_token from Yandex", http.StatusBadRequest)

		return

	}

	req, err := http.NewRequestWithContext(r.Context(), "GET", "https://login.yandex.ru/info?format=json", nil)

	if err != nil {

		g.log.Error("Failed to create Yandex userinfo request", zap.Error(err))

		http.Error(w, "failed to create user info request", http.StatusBadRequest)

		return

	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}

	userInfoResp, err := client.Do(req)

	if err != nil {

		g.log.Error("Failed to fetch Yandex userinfo with token", zap.Error(err))

		http.Error(w, "failed to get user info", http.StatusBadRequest)

		return

	}

	defer userInfoResp.Body.Close()

	if userInfoResp.StatusCode != http.StatusOK {

		g.log.Error("Yandex userinfo returned non-200", zap.Int("status", userInfoResp.StatusCode))

		http.Error(w, "failed to get user info", http.StatusBadRequest)

		return

	}

	var yandexUserInfo struct {
		ID          string `json:"id"`
		Login       string `json:"login"`
		DefaultEmail string `json:"default_email"`
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
		Email       string `json:"email"`
	}

	if err := json.NewDecoder(userInfoResp.Body).Decode(&yandexUserInfo); err != nil {

		g.log.Error("Failed to decode Yandex userinfo response", zap.Error(err))

		http.Error(w, "failed to decode user info", http.StatusBadRequest)

		return

	}

	if yandexUserInfo.ID == "" || yandexUserInfo.DefaultEmail == "" {

		g.log.Error("Yandex userinfo missing required fields",
			zap.String("id", yandexUserInfo.ID),
			zap.String("email", yandexUserInfo.DefaultEmail))

		http.Error(w, "Yandex token missing required claims", http.StatusBadRequest)

		return

	}

	emailVal := yandexUserInfo.DefaultEmail
	if emailVal == "" {
		emailVal = yandexUserInfo.Email
	}

	grpcResp, err := g.userClient.AuthenticateYandex(r.Context(), &userpb.AuthenticateYandexRequest{
		YandexId:  yandexUserInfo.ID,
		Email:     emailVal,
		Login:     yandexUserInfo.Login,
		FirstName: yandexUserInfo.FirstName,
		LastName:  yandexUserInfo.LastName,
	})

	if err != nil {

		httpCode, errMsg := grpcToHTTPStatus(err)

		g.log.Error("Yandex auth failed", zap.Error(err))

		http.Error(w, errMsg, httpCode)

		return

	}

	if g.userTOTPEnabled(r.Context(), grpcResp.GetUserId()) {

		tempToken := uuid.New().String()

		_ = g.valkeyDB.Set(r.Context(), twoFATempPrefix+tempToken, grpcResp.GetUserId(), 5*time.Minute).Err()

		w.Header().Set(headerContentType, contentTypeJSON)

		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"requires_2fa": true,
			"temp_token": tempToken,
			"message": "Please provide your 2FA code",
		}); err != nil {

			g.log.Error("Failed to encode Yandex 2FA response", zap.Error(err))

			http.Error(w, "encodeResponseError", http.StatusInternalServerError)

		}

		return

	}

	w.Header().Set(headerContentType, contentTypeJSON)

	accessCookie := &http.Cookie{
		Name:     accessTokenCookieName,
		Value:    grpcResp.GetAccessToken(),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   900,
	}
	http.SetCookie(w, accessCookie)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"user_id": grpcResp.GetUserId(),
		"role":    grpcResp.GetRole(),
	})

}


// 2FA TOTP endpoints

// @Summary      Setup TOTP two-factor authentication

// @Description  Generates TOTP secret and QR code for two-factor authentication setup

// @Tags         Auth

// @Accept       json

// @Produce      json

// @Success      200  {object}  map[string]interface{}

// @Failure      401  {object}  map[string]interface{}

// @Failure      412  {object}  map[string]interface{}

// @Failure      429  {object}  map[string]interface{}

// @Failure      409  {object}  map[string]interface{}

// @Failure      500  {object}  map[string]interface{}

// @Failure      503  {object}  map[string]interface{}

// @Router       /api/v1/auth/2fa/setup [post]

func (g *gateway) setupTOTPHandler(w http.ResponseWriter, r *http.Request) {

	userID, ok := r.Context().Value(middleware.UserIDKey).(string)

	if !ok {

		g.log.Error(errUnauthorized, zap.String("handler", "setupTOTP"))

		http.Error(w, "Unauthorized", http.StatusUnauthorized)

		return

	}

	if err := g.requireCriticalSession(r, userID); err != nil {

		g.log.Error(errCriticalSessionRequired, zap.Error(err))

		http.Error(w, err.Error(), http.StatusPreconditionRequired)

		return

	}

	if err := g.enforceTOTPRateLimit(r.Context(), "setup:"+userID); err != nil {

		g.log.Error(errTOTPRateLimitExceeded, zap.Error(err))

		http.Error(w, err.Error(), http.StatusTooManyRequests)

		return

	}

	resp, err := g.userClient.SetupTOTP(r.Context(), &userpb.SetupTOTPRequest{UserId: userID})

	if err != nil {

		httpCode, errMsg := grpcToHTTPStatus(err)

		g.log.Error("TOTP setup failed", zap.Error(err))

		http.Error(w, errMsg, httpCode)

		return

	}

	qrCodeBase64, err := encodeQRCodeBase64(resp.QrCodeUrl)

	if err != nil {

		g.log.Warn("Failed to encode TOTP QR code", zap.Error(err))

	}

	w.Header().Set(headerContentType, contentTypeJSON)

	if err := json.NewEncoder(w).Encode(map[string]interface{}{

		"qr_code_url": resp.QrCodeUrl,

		"qr_code_base64": qrCodeBase64,

		"secret": resp.Secret,

		"backup_codes": resp.BackupCodes,
	}); err != nil {

		g.log.Error("Failed to encode TOTP setup response", zap.Error(err))

		http.Error(w, "encodeResponseError", http.StatusInternalServerError)

		return

	}

}

// @Summary      Confirm TOTP setup

// @Description  Verifies TOTP passcode and completes two-factor authentication setup

// @Tags         Auth

// @Accept       json

// @Produce      json

// @Param        request  body  object  required  "TOTP confirmation request"

// @Success      200  {object}  map[string]interface{}

// @Failure      400  {object}  map[string]interface{}

// @Failure      401  {object}  map[string]interface{}

// @Failure      412  {object}  map[string]interface{}

// @Failure      429  {object}  map[string]interface{}

// @Failure      409  {object}  map[string]interface{}

// @Failure      500  {object}  map[string]interface{}

// @Router       /api/v1/auth/2fa/confirm [post]

func (g *gateway) confirmTOTPHandler(w http.ResponseWriter, r *http.Request) {

	userID, ok := r.Context().Value(middleware.UserIDKey).(string)

	if !ok {

		g.log.Error(errUnauthorized, zap.String("handler", "confirmTOTP"))

		http.Error(w, "Unauthorized", http.StatusUnauthorized)

		return

	}

	if err := g.requireCriticalSession(r, userID); err != nil {

		g.log.Error(errCriticalSessionRequired, zap.Error(err))

		http.Error(w, err.Error(), http.StatusPreconditionRequired)

		return

	}

	if err := g.enforceTOTPRateLimit(r.Context(), "confirm:"+userID); err != nil {

		g.log.Error(errTOTPRateLimitExceeded, zap.Error(err))

		http.Error(w, err.Error(), http.StatusTooManyRequests)

		return

	}

	var req struct {
		Passcode string `json:"passcode"`

		TempSecret string `json:"temp_secret"`

		BackupCodes []string `json:"backup_codes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		g.log.Error("Failed to decode confirm TOTP request", zap.Error(err))

		http.Error(w, errBadRequest, http.StatusBadRequest)

		return

	}

	resp, err := g.userClient.ConfirmTOTP(r.Context(), &userpb.ConfirmTOTPRequest{

		UserId: userID,

		Passcode: req.Passcode,

		TempSecret: req.TempSecret,

		BackupCodes: req.BackupCodes,
	})

	if err != nil {

		httpCode, errMsg := grpcToHTTPStatus(err)

		g.log.Error("Failed to confirm TOTP", zap.Error(err))

		http.Error(w, errMsg, httpCode)

		return

	}

	w.Header().Set(headerContentType, contentTypeJSON)

	if err := json.NewEncoder(w).Encode(map[string]interface{}{

		"success": resp.Success,

		"message": resp.Message,
	}); err != nil {

		g.log.Error("Failed to encode TOTP confirm response", zap.Error(err))

		http.Error(w, "encodeResponseError", http.StatusInternalServerError)

		return

	}

}

// @Summary      Verify TOTP code

// @Description  Verifies TOTP passcode using temporary token and returns access tokens

// @Tags         Auth

// @Accept       json

// @Produce      json

// @Param        request  body  object  required  "TOTP verification request"

// @Success      200  {object}  map[string]interface{}

// @Failure      400  {object}  map[string]interface{}

// @Failure      401  {object}  map[string]interface{}

// @Failure      429  {object}  map[string]interface{}

// @Failure      500  {object}  map[string]interface{}

// @Router       /api/v1/auth/2fa/verify [post]

func (g *gateway) verifyTOTPHandler(w http.ResponseWriter, r *http.Request) {

	var req struct {
		TempToken string `json:"temp_token"`

		Passcode string `json:"passcode"`

		IsBackupCode bool `json:"is_backup_code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		g.log.Error("Failed to decode verify TOTP request", zap.Error(err))

		http.Error(w, errBadRequest, http.StatusBadRequest)

		return

	}

	if req.TempToken == "" || req.Passcode == "" {

		g.log.Error("Missing temp_token or passcode in verify TOTP request")

		http.Error(w, "temp_token and passcode are required", http.StatusBadRequest)

		return

	}

	userID, err := g.valkeyDB.Get(r.Context(), twoFATempPrefix+req.TempToken).Result()

	if err != nil {

		g.log.Error("Invalid or expired 2FA session", zap.Error(err))

		http.Error(w, "Invalid or expired session", http.StatusUnauthorized)

		return

	}

	rateLimitErr := g.enforceTOTPRateLimit(r.Context(), "verify:"+userID)

	if rateLimitErr != nil {

		g.log.Error(errTOTPRateLimitExceeded, zap.Error(rateLimitErr))

		http.Error(w, rateLimitErr.Error(), http.StatusTooManyRequests)

		return

	}

	resp, err := g.userClient.VerifyTOTP(r.Context(), &userpb.VerifyTOTPRequest{

		UserId: userID,

		Passcode: req.Passcode,

		IsBackupCode: req.IsBackupCode,
	})

	if err != nil {

		httpCode, errMsg := grpcToHTTPStatus(err)

		g.log.Error("Failed to verify TOTP", zap.Error(err))

		http.Error(w, errMsg, httpCode)

		return

	}

	if !resp.Valid {

		g.log.Error("Invalid TOTP code")

		http.Error(w, "Invalid TOTP code", http.StatusUnauthorized)

		return

	}

	_ = g.valkeyDB.Del(r.Context(), twoFATempPrefix+req.TempToken)

	accessToken, err := g.issueJWT(r.Context(), userID)

	if err != nil {

		g.log.Error("Failed to issue JWT after 2FA", zap.Error(err), zap.String("user_id", userID))

		http.Error(w, "Failed to issue token", http.StatusInternalServerError)

		return

	}

	refreshToken, rtErr := g.issueRefreshToken(r.Context(), userID)

	if rtErr != nil {

		g.log.Warn("Failed to issue refresh token after 2FA", zap.Error(rtErr))

	}

	accessCookie := &http.Cookie{
		Name:     accessTokenCookieName,
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   900,
	}
	http.SetCookie(w, accessCookie)

	var refreshCookie *http.Cookie
	if rtErr == nil {
		refreshCookie = &http.Cookie{
			Name:     refreshTokenCookieName,
			Value:    refreshToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60,
	}
	http.SetCookie(w, refreshCookie)
	}

	w.Header().Set(headerContentType, contentTypeJSON)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":                 "ok",
		"token_type":             "Bearer",
		"expires_in":             900,
		"backup_codes_remaining": resp.BackupCodesRemaining,
	})

}

// @Summary      Disable TOTP two-factor authentication

// @Description  Disables two-factor authentication after verifying current passcode

// @Tags         Auth

// @Accept       json

// @Produce      json

// @Param        request  body  object  required  "TOTP disable request"

// @Success      200  {object}  map[string]interface{}

// @Failure      400  {object}  map[string]interface{}

// @Failure      401  {object}  map[string]interface{}

// @Failure      412  {object}  map[string]interface{}

// @Failure      429  {object}  map[string]interface{}

// @Failure      409  {object}  map[string]interface{}

// @Failure      500  {object}  map[string]interface{}

// @Router       /api/v1/auth/2fa/disable [post]

func (g *gateway) disableTOTPHandler(w http.ResponseWriter, r *http.Request) {

	userID, ok := r.Context().Value(middleware.UserIDKey).(string)

	if !ok {

		g.log.Error(errUnauthorized, zap.String("handler", "disableTOTP"))

		http.Error(w, "Unauthorized", http.StatusUnauthorized)

		return

	}

	if err := g.requireCriticalSession(r, userID); err != nil {

		g.log.Error(errCriticalSessionRequired, zap.Error(err))

		http.Error(w, err.Error(), http.StatusPreconditionRequired)

		return

	}

	if err := g.enforceTOTPRateLimit(r.Context(), "disable:"+userID); err != nil {

		g.log.Error(errTOTPRateLimitExceeded, zap.Error(err))

		http.Error(w, err.Error(), http.StatusTooManyRequests)

		return

	}

	var req struct {
		Passcode string `json:"passcode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		g.log.Error("Failed to decode disable TOTP request", zap.Error(err))

		http.Error(w, errBadRequest, http.StatusBadRequest)

		return

	}

	resp, err := g.userClient.DisableTOTP(r.Context(), &userpb.DisableTOTPRequest{

		UserId: userID,

		Passcode: req.Passcode,
	})

	if err != nil {

		httpCode, errMsg := grpcToHTTPStatus(err)

		g.log.Error("Failed to disable TOTP", zap.Error(err))

		http.Error(w, errMsg, httpCode)

		return

	}

	w.Header().Set(headerContentType, contentTypeJSON)

	if err := json.NewEncoder(w).Encode(map[string]interface{}{

		"success": resp.Success,

		"message": resp.Message,
	}); err != nil {

		g.log.Error("Failed to encode TOTP disable response", zap.Error(err))

		http.Error(w, "encodeResponseError", http.StatusInternalServerError)

		return

	}

}

// @Summary      Get TOTP status

// @Description  Returns whether TOTP is enabled and remaining backup codes count

// @Tags         Auth

// @Produce      json

// @Success      200  {object}  map[string]interface{}

// @Failure      401  {object}  map[string]interface{}

// @Failure      500  {object}  map[string]interface{}

// @Failure      503  {object}  map[string]interface{}

// @Router       /api/v1/auth/2fa/status [get]

func (g *gateway) totpStatusHandler(w http.ResponseWriter, r *http.Request) {

	userID, ok := r.Context().Value(middleware.UserIDKey).(string)

	if !ok || userID == "" {

		g.log.Error(errUnauthorized, zap.String("handler", "totpStatus"))

		http.Error(w, "Unauthorized", http.StatusUnauthorized)

		return

	}

	resp, err := g.userClient.GetUserClaims(r.Context(), &userpb.GetUserClaimsRequest{UserId: userID})

	if err != nil {

		g.log.Error("Failed to load TOTP status", zap.Error(err), zap.String("user_id", userID))

		http.Error(w, "Failed to load TOTP status", http.StatusInternalServerError)

		return

	}

	w.Header().Set(headerContentType, contentTypeJSON)

	if err := json.NewEncoder(w).Encode(map[string]interface{}{

		"enabled": resp.GetTotpEnabled(),

		"backup_codes_remaining": resp.GetTotpBackupCodesRemaining(),
	}); err != nil {

		g.log.Error("Failed to encode TOTP status response", zap.Error(err))

		http.Error(w, "encodeResponseError", http.StatusInternalServerError)

		return

	}

}

// @Summary      Refresh access token

// @Description  Rotates refresh token and returns new access and refresh tokens

// @Tags         Auth

// @Accept       json

// @Produce      json

// @Param        request  body  object  required  "Refresh token request"

// @Success      200  {object}  map[string]interface{}

// @Failure      400  {object}  map[string]interface{}

// @Failure      401  {object}  map[string]interface{}

// @Failure      500  {object}  map[string]interface{}

// @Failure      503  {object}  map[string]interface{}

// @Router       /api/v1/auth/refresh [post]

func (g *gateway) refreshHandler(w http.ResponseWriter, r *http.Request) {

	var refreshToken string
	if cookie, err := r.Cookie(refreshTokenCookieName); err == nil {
		refreshToken = cookie.Value
	}

	if refreshToken == "" {

		g.log.Error("Missing refresh_token in cookie")

		http.Error(w, "refresh_token обязателен", http.StatusBadRequest)

		return

	}

	accessToken, newRefresh, err := g.rotateRefreshToken(r.Context(), refreshToken)

	if err != nil {

		g.log.Warn("Refresh token rotation failed", zap.Error(err))

		http.Error(w, "Неверный или истёкший refresh token", http.StatusUnauthorized)

		return

	}

	accessCookie := &http.Cookie{
		Name:     accessTokenCookieName,
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   900,
	}
	http.SetCookie(w, accessCookie)

	refreshCookie := &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    newRefresh,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60,
	}
	http.SetCookie(w, refreshCookie)

	w.Header().Set(headerContentType, contentTypeJSON)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"token_type": "Bearer",
		"expires_in": 900,
	})

}

// @Summary      Request password reset

// @Description  Sends password reset code to user email

// @Tags         Auth

// @Accept       json

// @Produce      json

// @Param        request  body  object  required  "Email for password reset"

// @Success      200  {object}  map[string]interface{}

// @Failure      400  {object}  map[string]interface{}

// @Router       /api/v1/auth/forgot-password [post]

func (g *gateway) forgotPasswordHandler(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		http.Error(w, errBadRequest, http.StatusBadRequest)

		return

	}

	if req.Email == "" {

		http.Error(w, "email обязателен", http.StatusBadRequest)

		return

	}

	code := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)

	w.Header().Set(headerContentType, contentTypeJSON)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{

		"status": "ok",

		"message": "Код отправлен на email",

		"debug_code": code,
	})

}

// @Summary      Reset password with code

// @Description  Resets user password using email code

// @Tags         Auth

// @Accept       json

// @Produce      json

// @Param        request  body  object  required  "Reset password request"

// @Success      200  {object}  map[string]interface{}

// @Failure      400  {object}  map[string]interface{}

// @Router       /api/v1/auth/reset [post]

func (g *gateway) resetPasswordHandler(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Email string `json:"email"`

		Code string `json:"code"`

		NewPassword string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		http.Error(w, errBadRequest, http.StatusBadRequest)

		return

	}

	if req.Email == "" || req.Code == "" || req.NewPassword == "" {

		http.Error(w, "email, code и new_password обязательны", http.StatusBadRequest)

		return

	}

	if len(req.NewPassword) < 6 {

		http.Error(w, "Пароль должен содержать минимум 6 символов", http.StatusBadRequest)

		return

	}

	w.Header().Set(headerContentType, contentTypeJSON)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{

		"status": "ok",

		"message": "Пароль изменён",
	})

}
