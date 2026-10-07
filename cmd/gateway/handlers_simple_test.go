package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/auth/jwt"
	"github.com/MAMUER/project/internal/logger"
)

func setupSimpleGateway() *gateway {
	log := &logger.Logger{Logger: zap.NewNop()}
	privateKeyPEM, publicKeyPEM, _ := jwt.GenerateES256KeyPair()
	return &gateway{
		log:           log,
		tokenProvider: jwt.NewJWTAdapter(privateKeyPEM, publicKeyPEM),
	}
}

func TestForgotPasswordHandler_MissingEmail(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/auth/forgot-password", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	g.forgotPasswordHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestForgotPasswordHandler_Success(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/auth/forgot-password", bytes.NewReader([]byte(`{"email":"test@example.com"}`)))
	req.Header.Set("Content-Type", "application/json")

	g.forgotPasswordHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestResetPasswordHandler_InvalidPayload(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/auth/reset", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	g.resetPasswordHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetPasswordHandler_Success(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/auth/reset", bytes.NewReader([]byte(`{"email":"test@example.com","code":"123456","new_password":"newpass123"}`)))
	req.Header.Set("Content-Type", "application/json")

	g.resetPasswordHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestListVideosHandler_Success(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/videos", nil)

	g.listVideosHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "videos")
}
