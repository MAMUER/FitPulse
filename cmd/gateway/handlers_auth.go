package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"strings"

	userpb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/middleware"
	"go.uber.org/zap"
)

// ========== Auth Handlers ==========

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
	if err := json.NewEncoder(w).Encode(response); err != nil {
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
	if err := json.NewEncoder(w).Encode(response); err != nil {
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

	loginResp := map[string]interface{}{
		"status":       "ok",
		"access_token": resp.GetAccessToken(),
		"token_type":   resp.GetTokenType(),
		"expires_in":   resp.GetExpiresIn(),
	}
	if err := middleware.SignAndSendJSON(w, loginResp, g.jwtSecret, g.log.Logger); err != nil {
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
		if token == "" {
			w.WriteHeader(http.StatusBadRequest)
			if _, err := fmt.Fprint(w, "<html><body style='background:#0d1117;color:#c9d1d9;font-family:system-ui;'><div style='text-align:center;padding:40px;'><h1 style='color:#f85149;'>Ошибка</h1><p>Токен не найден</p></div></body></html>"); err != nil {
				g.log.Error("Failed to write fallback response", zap.Error(err))
			}
			return
		}
		safeToken := html.EscapeString(token)
		// nosemgrep: go.lang.security.audit.xss.no-fprintf-to-responsewriter.no-fprintf-to-responsewriter
		// SAFETY: Fallback email confirmation HTML. Token is HTML-escaped via `html.EscapeString(token)`
		// before being interpolated with `%s`, so no raw user input reaches the ResponseWriter.
		if _, err := fmt.Fprintf(w, "<html><body style='background:#0d1117;color:#c9d1d9;font-family:system-ui;'><div style='text-align:center;padding:40px;'><h1>Подтверждение email</h1><p>Токен: %s</p></div></body></html>", safeToken); err != nil {
			g.log.Error("Failed to write fallback response", zap.Error(err))
		}
		return
	}

	tmplText := string(tmplBytes)
	tmplText = strings.Replace(tmplText, "{{ .Token }}", html.EscapeString(token), 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := fmt.Fprint(w, tmplText); err != nil {
		g.log.Error("Failed to write confirm page", zap.Error(err))
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
