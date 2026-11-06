package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/MAMUER/project/internal/middleware"
)

func TestProfile_GetProfileHandler_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/profile", nil)

	g.getProfileHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestProfile_GetProfileHandler_Success(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "GET", "/api/v1/profile", nil)

	g.getProfileHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "status")
}

func TestProfile_GetProfileHandler_GRPCError(t *testing.T) {
	g := newTestGateway()
	g.userClient = &errorUserServiceClient{err: grpcError(codes.NotFound, "user not found")}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "GET", "/api/v1/profile", nil)

	g.getProfileHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProfile_UpdateProfileHandler_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "PUT", "/api/v1/profile", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	g.updateProfileHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestProfile_UpdateProfileHandler_InvalidJSON(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "PUT", "/api/v1/profile", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")

	g.updateProfileHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfile_UpdateProfileHandler_Success(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	reqBody := []byte(`{"full_name":"Updated Name","age":30,"gender":"male","height_cm":175,"weight_kg":70.5,"fitness_level":"intermediate"}`)
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "PUT", "/api/v1/profile", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	g.updateProfileHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestProfile_UpdateProfileHandler_GRPCError(t *testing.T) {
	g := newTestGateway()
	g.userClient = &errorUserServiceClient{err: grpcError(codes.Internal, "internal error")}

	w := httptest.NewRecorder()
	reqBody := []byte(`{"full_name":"Updated Name"}`)
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "PUT", "/api/v1/profile", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	g.updateProfileHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestProfile_DeleteProfileHandler_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/api/v1/profile", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	g.deleteProfileHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestProfile_DeleteProfileHandler_InvalidJSON(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "DELETE", "/api/v1/profile", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")

	g.deleteProfileHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfile_DeleteProfileHandler_MissingPassword(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	reqBody := []byte(`{"password":""}`)
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "DELETE", "/api/v1/profile", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	g.deleteProfileHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfile_DeleteProfileHandler_Success(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	reqBody := []byte(`{"password":"password123"}`)
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "DELETE", "/api/v1/profile", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	g.deleteProfileHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "status")
}

func TestProfile_DecodeDeleteProfileRequest_InvalidJSON(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/api/v1/profile", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")

	_, err := decodeDeleteProfileRequest(req)

	assert.Error(t, err)
}

func TestProfile_DecodeDeleteProfileRequest_MissingPassword(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/api/v1/profile", bytes.NewReader([]byte(`{"password":""}`)))
	req.Header.Set("Content-Type", "application/json")

	_, err := decodeDeleteProfileRequest(req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password")
}

func TestProfile_DecodeDeleteProfileRequest_Success(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/api/v1/profile", bytes.NewReader([]byte(`{"password":"password123"}`)))
	req.Header.Set("Content-Type", "application/json")

	result, err := decodeDeleteProfileRequest(req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "password123", result.Password)
}
func TestProfile_GetProfileHandler_EncodeError(_ *testing.T) {
	g := newTestGateway()
	w := &failingHW{ResponseWriter: httptest.NewRecorder()}
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodGet, "/api/v1/profile", nil)
	g.getProfileHandler(w, req)
}

func TestProfile_UpdateProfileHandler_GRPCErrorNon500(t *testing.T) {
	g := newTestGateway()
	g.userClient = &errorUserServiceClient{err: status.Error(codes.NotFound, "not found")}
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodPut, "/api/v1/profile", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()
	g.updateProfileHandler(w, req)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestProfile_UpdateProfileHandler_EncodeError(_ *testing.T) {
	g := newTestGateway()
	w := &failingHW{ResponseWriter: httptest.NewRecorder()}
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodPut, "/api/v1/profile", bytes.NewBufferString(`{}`))
	g.updateProfileHandler(w, req)
}

func TestProfile_DeleteProfileHandler_GRPCError(t *testing.T) {
	g := newTestGateway()
	g.userClient = &errorUserServiceClient{err: status.Error(codes.Internal, "boom")}
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodDelete, "/api/v1/profile", bytes.NewBufferString(`{"password":"p"}`))
	w := httptest.NewRecorder()
	g.deleteProfileHandler(w, req)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestProfile_DeleteProfileHandler_EncodeError(_ *testing.T) {
	g := newTestGateway()
	w := &failingHW{ResponseWriter: httptest.NewRecorder()}
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodDelete, "/api/v1/profile", bytes.NewBufferString(`{"password":"p"}`))
	g.deleteProfileHandler(w, req)
}

func TestProfile_ExportUserDataHandler_Unauthorized(t *testing.T) {
	g := newTestGateway()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile/export", nil)
	g.exportUserDataHandler(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestProfile_ExportUserDataHandler_GRPCError(t *testing.T) {
	g := newTestGateway()
	g.userClient = &errorUserServiceClient{err: status.Error(codes.Internal, "boom")}
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodGet, "/api/v1/profile/export", nil)
	g.exportUserDataHandler(w, req)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestProfile_ExportUserDataHandler_EncodeError(_ *testing.T) {
	g := newTestGateway()
	w := &failingHW{ResponseWriter: httptest.NewRecorder()}
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodGet, "/api/v1/profile/export", nil)
	g.exportUserDataHandler(w, req)
}
