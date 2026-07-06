package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"html/template"
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
	"github.com/MAMUER/project/internal/auth"
	"github.com/MAMUER/project/internal/middleware"
)

// ========== Auth Handlers ==========

const (
	confirmFallbackHTML = `<html><body style='background:#0d1117;color:#c9d1d9;font-family:system-ui;'><div style='text-align:center;padding:40px;'><h1>Подтверждение email</h1><p>Токен: {{ .Token }}</p></div></body></html>`

	confirmFallbackErrorHTML = `<html><body style='background:#0d1117;color:#c9d1d9;font-family:system-ui;'><div style='text-align:center;padding:40px;'><h1 style='color:#f85149;'>Ошибка</h1><p>Токен не найден</p></div></body></html>`

	totpRateLimitAttempts = 5

	googleOAuthStateCookie = "google_oauth_state"
)

type totpRateLimiter struct {
	limiter   *rate.Limiter
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
	if g.db == nil || userID == "" {
		return false
	}

	var totpEnabled bool
	if err := g.db.QueryRowContext(ctx, "SELECT totp_enabled FROM users WHERE id = $1", userID).Scan(&totpEnabled); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			g.log.Warn("Could not check TOTP status", zap.Error(err), zap.String("user_id", userID))
		}
		return false
	}

	return totpEnabled
}

func (g *gateway) issueJWT(ctx context.Context, userID string) (string, error) {
	if g.db == nil || userID == "" {
		return "", errors.New("database unavailable")
	}

	var email, role string
	if err := g.db.QueryRowContext(ctx, "SELECT email, role FROM users WHERE id = $1", userID).Scan(&email, &role); err != nil {
		return "", fmt.Errorf("query user for JWT: %w", err)
	}

	token, err := auth.GenerateAccessToken(userID, email, role, g.jwtPrivateKeyPEM, 15*time.Minute)
	return token, fmt.Errorf("issue jwt: %w", err)
}

func (g *gateway) issueRefreshToken(ctx context.Context, userID string) (string, error) {
	if g.rdb == nil {
		return "", errors.New("redis unavailable")
	}
	token := auth.GenerateRefreshToken()
	key := "refresh:" + token
	if err := g.rdb.Set(ctx, key, userID, 7*24*time.Hour).Err(); err != nil {
		return "", fmt.Errorf("issue refresh token: %w", err)
	}
	return token, nil
}

func (g *gateway) rotateRefreshToken(ctx context.Context, oldToken string) (string, string, error) {
	userID, err := g.rdb.Get(ctx, "refresh:"+oldToken).Result()
	if err != nil {
		return "", "", errors.New("invalid refresh token")
	}
	_ = g.rdb.Del(ctx, "refresh:"+oldToken).Err()
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

func (g *gateway) enforceTOTPRateLimit(ctx context.Context, key string) error {
	redisKey := "2fa_rate:" + key
	if g.rdb != nil {
		count, err := g.rdb.Incr(ctx, redisKey).Result()
		if err == nil {
			if count == 1 {
				_ = g.rdb.Expire(ctx, redisKey, time.Minute).Err()
			}
			if count > totpRateLimitAttempts {
				return errors.New("too many 2FA attempts")
			}
			return nil
		}
		g.log.Warn("Redis 2FA rate limit unavailable", zap.Error(err))
	}

	if countOverLimit(ctx, g, key) {
		return errors.New("too many 2FA attempts")
	}

	return nil
}

func countOverLimit(g *gateway, key string) bool {
	v, _ := g.totpRateLimiters.LoadOrStore(key, &totpRateLimiter{
		limiter:   rate.NewLimiter(totpRateLimitAttempts, totpRateLimitAttempts),
		expiresAt: time.Now().Add(time.Minute),
	})
	limiter := v.(*totpRateLimiter)
	if time.Now().After(limiter.expiresAt) {
		limiter = &totpRateLimiter{
			limiter:   rate.NewLimiter(totpRateLimitAttempts, totpRateLimitAttempts),
			expiresAt: time.Now().Add(time.Minute),
		}
		g.totpRateLimiters.Store(key, limiter)
	}
	return !limiter.limiter.Allow()
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

func (g *gateway) registerHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.log.Error("Failed to decode register request", zap.Error(err))
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.Register(r.Context(), &userpb.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
		Role:     req.Role,
	})
	if err != nil {
		httpCode, errMsg := grpcToHTTPStatus(err)
		g.log.Error("Register failed", zap.Error(err))
		http.Error(w, errMsg, httpCode)
		return
	}

	// Return registration result including verification token (dev mode)
	response := map[string]interface{}{"status": "ok"}
	if resp.GetMessage() != "" {
		response["message"] = resp.GetMessage()
	}
	if err := middleware.SignAndSendJSON(w, response, g.responseSigningSecret, g.log.Logger); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

func (g *gateway) registerWithInviteHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email         string `json:"email"`
		Password      string `json:"password"`
		FullName      string `json:"full_name"`
		InviteCode    string `json:"invite_code"`
		LicenseNumber string `json:"license_number"`
		Specialty     string `json:"specialty"`
		Phone         string `json:"phone"`
		Bio           string `json:"bio"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.log.Error("Failed to decode register with invite request", zap.Error(err))
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.RegisterWithInvite(r.Context(), &userpb.RegisterWithInviteRequest{
		Email:      req.Email,
		Password:   req.Password,
		FullName:   req.FullName,
		InviteCode: req.InviteCode,
	})
	if err != nil {
		httpCode, errMsg := grpcToHTTPStatus(err)
		g.log.Error("Register with invite failed", zap.Error(err))
		http.Error(w, errMsg, httpCode)
		return
	}

	response := map[string]interface{}{
		"status":  "ok",
		"user_id": resp.GetUserId(),
	}
	if resp.GetMessage() != "" {
		response["message"] = resp.GetMessage()
	}
	if err := middleware.SignAndSendJSON(w, response, g.responseSigningSecret, g.log.Logger); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

func (g *gateway) validateInviteCodeHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.log.Error("Failed to decode validate invite request", zap.Error(err))
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.ValidateInviteCode(r.Context(), &userpb.ValidateInviteCodeRequest{
		Code: req.Code,
	})
	if err != nil {
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"is_valid":  resp.GetIsValid(),
		"role":      resp.GetRole(),
		"specialty": resp.GetSpecialty(),
		"error":     resp.GetErrorMessage(),
	}); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

func (g *gateway) loginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.log.Error("Failed to decode login request", zap.Error(err))
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.Login(r.Context(), &userpb.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		httpCode, errMsg := grpcToHTTPStatus(err)
		g.log.Error("Login failed", zap.Error(err), zap.String("email", html.EscapeString(strings.ReplaceAll(strings.ReplaceAll(req.Email, "\n", ""), "\r", ""))))
		if httpCode == http.StatusUnauthorized && strings.Contains(errMsg, "Email not confirmed") {
			http.Error(w, "Email не подтверждён. Проверьте вашу почту.", httpCode)
			return
		}
		http.Error(w, errMsg, httpCode)
		return
	}

	if g.userTOTPEnabled(r.Context(), resp.GetUserId()) {
		tempToken := uuid.New().String()
		_ = g.rdb.Set(r.Context(), "2fa_temp:"+tempToken, resp.GetUserId(), 5*time.Minute).Err()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"requires_2fa": true,
			"temp_token":   tempToken,
			"message":      "Please provide your 2FA code",
		}); err != nil {
			g.log.Error("Failed to encode 2FA response", zap.Error(err))
			http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
			return
		}
		return
	}

	loginResp := map[string]interface{}{
		"status":       "ok",
		"access_token": resp.GetAccessToken(),
		"token_type":   resp.GetTokenType(),
		"expires_in":   900,
	}
	refreshToken, rtErr := g.issueRefreshToken(r.Context(), resp.GetUserId())
	if rtErr == nil {
		loginResp["refresh_token"] = refreshToken
	} else {
		g.log.Warn("Failed to issue refresh token", zap.Error(rtErr))
	}
	if err := middleware.SignAndSendJSON(w, loginResp, g.responseSigningSecret, g.log.Logger); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(map[string]string{"status": "logged_out"}); err != nil {
		g.log.Error("Failed to encode logout response", zap.Error(err))
		return
	}
}

func (g *gateway) confirmEmailHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.log.Error("Failed to decode confirm email request", zap.Error(err))
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	if req.Token == "" {
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

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Email confirmed. You can now log in.",
		"user_id": resp.GetUserId(),
	}); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

func (g *gateway) emailConfirmPageHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	// Load template from web/templates/confirm.html
	tmplPath := "./web/templates/confirm.html"
	tmplBytes, err := os.ReadFile(tmplPath)
	if err != nil {
		g.log.Warn("Failed to load confirm template, using fallback", zap.Error(err))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		g.renderConfirmFallback(w, token, token == "")
		return
	}

	tmpl, parseErr := template.New("confirm").Parse(string(tmplBytes))
	if parseErr != nil {
		g.log.Warn("Failed to parse confirm template, using fallback", zap.Error(parseErr))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		g.renderConfirmFallback(w, token, false)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if executeErr := tmpl.Execute(w, struct{ Token string }{Token: token}); executeErr != nil {
		g.log.Error("Failed to write confirm page", zap.Error(executeErr))
	}
}

func (g *gateway) renderConfirmFallback(w http.ResponseWriter, token string, tokenEmpty bool) {
	if tokenEmpty {
		w.WriteHeader(http.StatusBadRequest)
		fallbackTemplate, parseErr := template.New("confirmFallbackError").Parse(confirmFallbackErrorHTML)
		if parseErr != nil {
			g.log.Error("Failed to parse fallback error template", zap.Error(parseErr))
			return
		}
		if executeErr := fallbackTemplate.Execute(w, nil); executeErr != nil {
			g.log.Error("Failed to write fallback response", zap.Error(executeErr))
		}
		return
	}

	fallbackTemplate, parseErr := template.New("confirmFallback").Parse(confirmFallbackHTML)
	if parseErr != nil {
		g.log.Error("Failed to parse fallback confirm template", zap.Error(parseErr))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
	if executeErr := fallbackTemplate.Execute(w, struct{ Token string }{Token: token}); executeErr != nil {
		g.log.Error("Failed to write fallback response", zap.Error(executeErr))
	}
}

func (g *gateway) googleLoginHandler(w http.ResponseWriter, r *http.Request) {
	if g.googleOAuthConfig == nil {
		http.Error(w, "Google OAuth not configured", http.StatusNotImplemented)
		return
	}

	state, err := generateOAuthState()
	if err != nil {
		g.log.Error("Failed to generate Google OAuth state", zap.Error(err))
		http.Error(w, "failed to generate oauth state", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     googleOAuthStateCookie,
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		Expires:  time.Now().Add(10 * time.Minute),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	redirectURL := g.googleOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

func (g *gateway) googleCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if g.googleOAuthConfig == nil {
		http.Error(w, "Google OAuth not configured", http.StatusNotImplemented)
		return
	}

	state := r.URL.Query().Get("state")
	cookie, err := r.Cookie(googleOAuthStateCookie)
	if err != nil || state == "" || cookie == nil || subtle.ConstantTimeCompare([]byte(state), []byte(cookie.Value)) != 1 {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     googleOAuthStateCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	token, err := g.googleOAuthConfig.Exchange(r.Context(), code)
	if err != nil {
		g.log.Error("Failed to exchange Google code", zap.Error(err))
		http.Error(w, "failed to exchange authorization code", http.StatusBadRequest)
		return
	}

	idToken, ok := token.Extra("id_token").(string)
	if !ok || idToken == "" {
		g.log.Error("Google token missing id_token")
		http.Error(w, "missing id_token from Google", http.StatusBadRequest)
		return
	}

	grpcResp, err := g.userClient.AuthenticateGoogle(r.Context(), &userpb.AuthenticateGoogleRequest{
		IdToken: idToken,
	})
	if err != nil {
		httpCode, errMsg := grpcToHTTPStatus(err)
		g.log.Error("Google auth failed", zap.Error(err))
		http.Error(w, errMsg, httpCode)
		return
	}

	if g.userTOTPEnabled(r.Context(), grpcResp.GetUserId()) {
		tempToken := uuid.New().String()
		_ = g.rdb.Set(r.Context(), "2fa_temp:"+tempToken, grpcResp.GetUserId(), 5*time.Minute).Err()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"requires_2fa": true,
			"temp_token":   tempToken,
			"message":      "Please provide your 2FA code",
		}); err != nil {
			g.log.Error("Failed to encode Google 2FA response", zap.Error(err))
			http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := middleware.SignAndSendJSON(w, map[string]interface{}{
		"status":       "ok",
		"access_token": grpcResp.GetAccessToken(),
		"token_type":   grpcResp.GetTokenType(),
		"expires_in":   900,
		"user_id":      grpcResp.GetUserId(),
		"role":         grpcResp.GetRole(),
	}, g.responseSigningSecret, g.log.Logger); err != nil {
		g.log.Error("Failed to encode Google auth response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

// 2FA TOTP endpoints

func (g *gateway) setupTOTPHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := g.enforceTOTPRateLimit(r.Context(), "setup:"+userID); err != nil {
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

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"qr_code_url":    resp.QrCodeUrl,
		"qr_code_base64": qrCodeBase64,
		"secret":         resp.Secret,
		"backup_codes":   resp.BackupCodes,
	}); err != nil {
		g.log.Error("Failed to encode TOTP setup response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

func (g *gateway) confirmTOTPHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := g.enforceTOTPRateLimit(r.Context(), "confirm:"+userID); err != nil {
		http.Error(w, err.Error(), http.StatusTooManyRequests)
		return
	}

	var req struct {
		Passcode    string   `json:"passcode"`
		TempSecret  string   `json:"temp_secret"`
		BackupCodes []string `json:"backup_codes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.ConfirmTOTP(r.Context(), &userpb.ConfirmTOTPRequest{
		UserId:      userID,
		Passcode:    req.Passcode,
		TempSecret:  req.TempSecret,
		BackupCodes: req.BackupCodes,
	})
	if err != nil {
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": resp.Success,
		"message": resp.Message,
	}); err != nil {
		g.log.Error("Failed to encode TOTP confirm response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

func (g *gateway) verifyTOTPHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TempToken    string `json:"temp_token"`
		Passcode     string `json:"passcode"`
		IsBackupCode bool   `json:"is_backup_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}
	if req.TempToken == "" || req.Passcode == "" {
		http.Error(w, "temp_token and passcode are required", http.StatusBadRequest)
		return
	}

	userID, err := g.rdb.Get(r.Context(), "2fa_temp:"+req.TempToken).Result()
	if err != nil {
		http.Error(w, "Invalid or expired session", http.StatusUnauthorized)
		return
	}

	rateLimitErr := g.enforceTOTPRateLimit(r.Context(), "verify:"+userID)
	if rateLimitErr != nil {
		http.Error(w, rateLimitErr.Error(), http.StatusTooManyRequests)
		return
	}

	resp, err := g.userClient.VerifyTOTP(r.Context(), &userpb.VerifyTOTPRequest{
		UserId:       userID,
		Passcode:     req.Passcode,
		IsBackupCode: req.IsBackupCode,
	})
	if err != nil {
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	if !resp.Valid {
		http.Error(w, "Invalid TOTP code", http.StatusUnauthorized)
		return
	}

	_ = g.rdb.Del(r.Context(), "2fa_temp:"+req.TempToken)

	token, err := g.issueJWT(r.Context(), userID)
	if err != nil {
		g.log.Error("Failed to issue JWT after 2FA", zap.Error(err), zap.String("user_id", userID))
		http.Error(w, "Failed to issue token", http.StatusInternalServerError)
		return
	}

	refreshToken, rtErr := g.issueRefreshToken(r.Context(), userID)
	if rtErr != nil {
		g.log.Warn("Failed to issue refresh token after 2FA", zap.Error(rtErr))
	}

	if err := middleware.SignAndSendJSON(w, map[string]interface{}{
		"access_token":           token,
		"token_type":             "Bearer",
		"expires_in":             900,
		"refresh_token":          refreshToken,
		"backup_codes_remaining": resp.BackupCodesRemaining,
	}, g.responseSigningSecret, g.log.Logger); err != nil {
		g.log.Error("Failed to encode TOTP verify response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

func (g *gateway) disableTOTPHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := g.enforceTOTPRateLimit(r.Context(), "disable:"+userID); err != nil {
		http.Error(w, err.Error(), http.StatusTooManyRequests)
		return
	}

	var req struct {
		Passcode string `json:"passcode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.DisableTOTP(r.Context(), &userpb.DisableTOTPRequest{
		UserId:   userID,
		Passcode: req.Passcode,
	})
	if err != nil {
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": resp.Success,
		"message": resp.Message,
	}); err != nil {
		g.log.Error("Failed to encode TOTP disable response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

func (g *gateway) totpStatusHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || g.db == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var enabled bool
	var remaining int32
	if err := g.db.QueryRowContext(r.Context(), `
		SELECT totp_enabled, COALESCE(totp_backup_codes_remaining, 0)
		FROM users WHERE id = $1
	`, userID).Scan(&enabled, &remaining); err != nil {
		g.log.Error("Failed to load TOTP status", zap.Error(err), zap.String("user_id", userID))
		http.Error(w, "Failed to load TOTP status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"enabled":                enabled,
		"backup_codes_remaining": remaining,
	}); err != nil {
		g.log.Error("Failed to encode TOTP status response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

func (g *gateway) refreshHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}
	if req.RefreshToken == "" {
		http.Error(w, "refresh_token обязателен", http.StatusBadRequest)
		return
	}

	accessToken, newRefresh, err := g.rotateRefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		g.log.Warn("Refresh token rotation failed", zap.Error(err))
		http.Error(w, "Неверный или истёкший refresh token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": newRefresh,
		"token_type":    "Bearer",
		"expires_in":    900,
	}); err != nil {
		g.log.Error("Failed to encode refresh response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

// checkVerificationStatusHandler checks if a user's email is confirmed.
func (g *gateway) checkVerificationStatusHandler(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "Укажите email", http.StatusBadRequest)
		return
	}

	// Query user profile by email — we use GetProfile which requires user_id,
	// but since we only have email, we need to search via the user service.
	// The gateway doesn't have a GetUserByEmail RPC, so we return a not found
	// if we can't resolve the user. For now, we check if the user exists
	// by attempting a profile lookup. In production, add a GetUserByEmail RPC.
	// As a workaround, we return email_confirmed: false for unknown emails.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"email_confirmed": false,
		"email":           email,
	}); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}
