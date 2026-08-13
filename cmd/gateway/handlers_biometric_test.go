package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/MAMUER/project/internal/middleware"

	biometricpb "github.com/MAMUER/project/api/gen/biometric"
)

func TestBiometric_ProxyToBiometricWithUser_Unauthorized(t *testing.T) {
	g := newTestGateway()
	withProxy(g, "http://localhost:99999")

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/biometric/webhook", nil)

	g.proxyToBiometricWithUser(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBiometric_ProxyToBiometricWithUser_Success(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer backend.Close()

	g := newTestGateway()
	withProxy(g, backend.URL)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "GET", "/api/v1/biometric/webhook", nil)

	g.proxyToBiometricWithUser(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBiometric_AddBiometricRecordHandler_Unauthorized(t *testing.T) {
	g := newTestGateway()
	withBiometricClient(g)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/biometric/records", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	g.addBiometricRecordHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBiometric_AddBiometricRecordHandler_InvalidJSON(t *testing.T) {
	g := newTestGateway()
	withBiometricClient(g)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/biometric/records", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")

	g.addBiometricRecordHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBiometric_AddBiometricRecordHandler_InvalidData(t *testing.T) {
	g := newTestGateway()
	withBiometricClient(g)

	w := httptest.NewRecorder()
	reqBody := []byte(`{"metric_type":"heart_rate","value":-10}`)
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/biometric/records", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	g.addBiometricRecordHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBiometric_AddBiometricRecordHandler_Success(t *testing.T) {
	g := newTestGateway()
	withBiometricClient(g)

	w := httptest.NewRecorder()
	reqBody := []byte(`{"metric_type":"heart_rate","value":70,"timestamp":"2024-01-01T00:00:00Z","device_type":"watch"}`)
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/biometric/records", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	g.addBiometricRecordHandler(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "status")
}

func TestBiometric_AddBiometricRecordHandler_GRPCError(t *testing.T) {
	g := newTestGateway()
	withBiometricClient(g)
	bioClient := g.biometricClient.(*mockBiometricClient)
	bioClient.addErr = grpcError(codes.Internal, "internal error")

	w := httptest.NewRecorder()
	reqBody := []byte(`{"metric_type":"heart_rate","value":70,"timestamp":"2024-01-01T00:00:00Z","device_type":"watch"}`)
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/biometric/records", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	g.addBiometricRecordHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestBiometric_GetBiometricRecordsHandler_Unauthorized(t *testing.T) {
	g := newTestGateway()
	withBiometricClient(g)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/biometric/records", nil)

	g.getBiometricRecordsHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBiometric_GetBiometricRecordsHandler_Success(t *testing.T) {
	g := newTestGateway()
	withBiometricClient(g)
	bioClient := g.biometricClient.(*mockBiometricClient)
	bioClient.records["heart_rate"] = &biometricpb.BiometricRecord{
		Id:        "rec-1",
		UserId:    "user-123",
		MetricType: "heart_rate",
		Value:     70,
		DeviceType: "watch",
		Timestamp:  timestamppb.New(time.Now()),
		CreatedAt:  timestamppb.New(time.Now()),
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "GET", "/api/v1/biometric/records?metric_type=heart_rate", nil)

	g.getBiometricRecordsHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "records")
}

func TestBiometric_GetBiometricRecordsHandler_GRPCError(t *testing.T) {
	g := newTestGateway()
	withBiometricClient(g)
	bioClient := g.biometricClient.(*mockBiometricClient)
	bioClient.getErr = grpcError(codes.Internal, "internal error")

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "GET", "/api/v1/biometric/records", nil)

	g.getBiometricRecordsHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
