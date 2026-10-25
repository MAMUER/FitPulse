package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/auth/jwt"
	"github.com/MAMUER/project/internal/logger"

	userpb "github.com/MAMUER/project/api/gen/user"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

func setupSurveyGateway() *gateway {
	log := &logger.Logger{Logger: zap.NewNop()}
	privateKeyPEM, publicKeyPEM, _ := jwt.GenerateES256KeyPair()
	return &gateway{
		log:           log,
		tokenProvider: jwt.NewJWTAdapter(privateKeyPEM, publicKeyPEM),
		userClient:    &mockUserServiceClient{},
	}
}

func TestSaveSurveyHandler_Success(t *testing.T) {
	g := setupSurveyGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(withUserID(context.Background(), "user-1"), "POST", "/api/v1/survey", bytes.NewReader([]byte(`{"survey":{"goal":"weight_loss"},"survey_completed":true}`)))
	req.Header.Set("Content-Type", "application/json")

	g.saveSurveyHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestSaveSurveyHandler_Unauthorized(t *testing.T) {
	g := setupSurveyGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/survey", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	g.saveSurveyHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSaveSurveyHandler_InvalidJSON(t *testing.T) {
	g := setupSurveyGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(withUserID(context.Background(), "user-1"), "POST", "/api/v1/survey", bytes.NewReader([]byte(`invalid`)))
	req.Header.Set("Content-Type", "application/json")

	g.saveSurveyHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoadSurveyHandler_Success(t *testing.T) {
	g := setupSurveyGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(withUserID(context.Background(), "user-1"), "GET", "/api/v1/survey", nil)

	g.loadSurveyHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "status")
}

func TestLoadSurveyHandler_Unauthorized(t *testing.T) {
	g := setupSurveyGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/survey", nil)

	g.loadSurveyHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMustStruct(t *testing.T) {
	m := map[string]interface{}{"goal": "weight_loss"}
	s := mustStruct(m)
	assert.NotNil(t, s)
	assert.Equal(t, "weight_loss", s.Fields["goal"].GetStringValue())
}

func TestMustTimestamp(t *testing.T) {
	now := time.Now()
	ts := mustTimestamp(now)
	assert.NotNil(t, ts)
	assert.Equal(t, now.Unix(), ts.AsTime().Unix())
}

func TestMustTimestamp_Zero(t *testing.T) {
	ts := mustTimestamp(time.Time{})
	assert.Nil(t, ts)
}
func TestSaveSurveyHandler_NotCompleted(t *testing.T) {
	g := setupSurveyGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(withUserID(context.Background(), "user-1"), "POST", "/api/v1/survey", bytes.NewReader([]byte(`{"survey":{"goal":"weight_loss"},"survey_completed":false}`)))
	req.Header.Set("Content-Type", "application/json")

	g.saveSurveyHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSaveSurveyHandler_Error(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{
		log:        log,
		userClient: &errorUserServiceClient{err: assert.AnError},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(withUserID(context.Background(), "user-1"), "POST", "/api/v1/survey", bytes.NewReader([]byte(`{"survey":{"goal":"weight_loss"},"survey_completed":true}`)))
	req.Header.Set("Content-Type", "application/json")

	g.saveSurveyHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestLoadSurveyHandler_Error(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{
		log:        log,
		userClient: &errorUserServiceClient{err: assert.AnError},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(withUserID(context.Background(), "user-1"), "GET", "/api/v1/survey", nil)

	g.loadSurveyHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

type surveyWithClient struct {
	mockUserServiceClient
}

func (m *surveyWithClient) LoadSurvey(ctx context.Context, req *userpb.LoadSurveyRequest, opts ...grpc.CallOption) (*userpb.LoadSurveyResponse, error) {
	st, _ := structpb.NewStruct(map[string]interface{}{"goal": "weight_loss"})
	return &userpb.LoadSurveyResponse{Status: "ok", Survey: st, SurveyCompleted: true}, nil
}

func TestLoadSurveyHandler_WithSurvey(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{
		log:        log,
		userClient: &surveyWithClient{},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(withUserID(context.Background(), "user-1"), "GET", "/api/v1/survey", nil)

	g.loadSurveyHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "weight_loss")
}
func TestMustStruct_Error(t *testing.T) {
	s := mustStruct(map[string]interface{}{"bad": make(chan int)})
	assert.NotNil(t, s)
	assert.Empty(t, s.Fields)
}
