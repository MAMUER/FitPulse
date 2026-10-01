package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/auth/jwt"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/middleware"
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

func TestMLChatHandler_EmptyMessage(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/ml/chat", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	g.mlChatHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMLChatHandler_Unauthorized(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/ml/chat", bytes.NewReader([]byte(`{"message":"plan"}`)))
	req.Header.Set("Content-Type", "application/json")

	g.mlChatHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMLChatHandler_Success(t *testing.T) {
	g := newTestGateway()
	withBiometricClient(g)

	classifierListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	classifierPort := classifierListener.Addr().(*net.TCPAddr).Port
	classifierServer := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"predicted_class": "endurance_basic",
			"confidence":      0.9,
			"recommendations": []string{"increase cardio", "rest more"},
		})
	})}
	go func() {
		if serveErr := classifierServer.Serve(classifierListener); serveErr != nil && serveErr != http.ErrServerClosed {
			t.Logf("classifier server error: %v", serveErr)
		}
	}()
	defer func() {
		if closeErr := classifierServer.Close(); closeErr != nil {
			t.Logf("classifier close error: %v", closeErr)
		}
	}()

	mlGeneratorListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	mlGeneratorPort := mlGeneratorListener.Addr().(*net.TCPAddr).Port
	mlGeneratorServer := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"plan": "generated-plan",
		})
	})}
	go func() {
		if serveErr := mlGeneratorServer.Serve(mlGeneratorListener); serveErr != nil && serveErr != http.ErrServerClosed {
			t.Logf("ml generator server error: %v", serveErr)
		}
	}()
	defer func() {
		if closeErr := mlGeneratorServer.Close(); closeErr != nil {
			t.Logf("ml generator close error: %v", closeErr)
		}
	}()

	g.classifierURL = "http://127.0.0.1:" + strconv.Itoa(classifierPort)
	g.mlGeneratorURL = "http://127.0.0.1:" + strconv.Itoa(mlGeneratorPort)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/ml/chat", bytes.NewReader([]byte(`{"message":"plan"}`)))
	req.Header.Set("Content-Type", "application/json")

	g.mlChatHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
	assert.Contains(t, w.Body.String(), "endurance_basic")
}

func TestListVideosHandler_Success(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/videos", nil)

	g.listVideosHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "videos")
}
