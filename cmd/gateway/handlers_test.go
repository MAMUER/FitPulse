package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	userpb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/logger"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type mockUserServiceClient struct{}

func (m *mockUserServiceClient) Register(ctx context.Context, req *userpb.RegisterRequest, opts ...grpc.CallOption) (*userpb.RegisterResponse, error) {
	return &userpb.RegisterResponse{UserId: "user-123"}, nil
}
func (m *mockUserServiceClient) RegisterWithInvite(ctx context.Context, req *userpb.RegisterWithInviteRequest, opts ...grpc.CallOption) (*userpb.RegisterResponse, error) {
	return &userpb.RegisterResponse{UserId: "invited-123"}, nil
}
func (m *mockUserServiceClient) ConfirmEmail(ctx context.Context, req *userpb.ConfirmEmailRequest, opts ...grpc.CallOption) (*userpb.ConfirmEmailResponse, error) {
	return &userpb.ConfirmEmailResponse{UserId: "user-123", Message: "ok"}, nil
}
func (m *mockUserServiceClient) Login(ctx context.Context, req *userpb.LoginRequest, opts ...grpc.CallOption) (*userpb.LoginResponse, error) {
	return &userpb.LoginResponse{AccessToken: "token", UserId: "user-123", Role: "client"}, nil
}
func (m *mockUserServiceClient) AuthenticateGoogle(ctx context.Context, req *userpb.AuthenticateGoogleRequest, opts ...grpc.CallOption) (*userpb.LoginResponse, error) {
	return &userpb.LoginResponse{AccessToken: "token", UserId: "user-123"}, nil
}
func (m *mockUserServiceClient) GetProfile(ctx context.Context, req *userpb.GetProfileRequest, opts ...grpc.CallOption) (*userpb.UserProfile, error) {
	return &userpb.UserProfile{UserId: req.UserId, Email: "test@example.com"}, nil
}
func (m *mockUserServiceClient) UpdateProfile(ctx context.Context, req *userpb.UpdateProfileRequest, opts ...grpc.CallOption) (*userpb.UserProfile, error) {
	return &userpb.UserProfile{UserId: req.UserId, Email: "test@example.com"}, nil
}
func (m *mockUserServiceClient) ChangePassword(ctx context.Context, req *userpb.ChangePasswordRequest, opts ...grpc.CallOption) (*userpb.ChangePasswordResponse, error) {
	return &userpb.ChangePasswordResponse{Message: "ok"}, nil
}
func (m *mockUserServiceClient) ChangeEmail(ctx context.Context, req *userpb.ChangeEmailRequest, opts ...grpc.CallOption) (*userpb.ChangeEmailResponse, error) {
	return &userpb.ChangeEmailResponse{Message: "ok"}, nil
}
func (m *mockUserServiceClient) UploadProfilePhoto(ctx context.Context, req *userpb.UploadProfilePhotoRequest, opts ...grpc.CallOption) (*userpb.UploadProfilePhotoResponse, error) {
	return &userpb.UploadProfilePhotoResponse{PhotoUrl: "url"}, nil
}
func (m *mockUserServiceClient) RemoveProfilePhoto(ctx context.Context, req *userpb.RemoveProfilePhotoRequest, opts ...grpc.CallOption) (*userpb.RemoveProfilePhotoResponse, error) {
	return &userpb.RemoveProfilePhotoResponse{Message: "ok"}, nil
}
func (m *mockUserServiceClient) ChangeNickname(ctx context.Context, req *userpb.ChangeNicknameRequest, opts ...grpc.CallOption) (*userpb.ChangeNicknameResponse, error) {
	return &userpb.ChangeNicknameResponse{Message: "ok"}, nil
}
func (m *mockUserServiceClient) ListDevices(ctx context.Context, req *userpb.ListDevicesRequest, opts ...grpc.CallOption) (*userpb.ListDevicesResponse, error) {
	return &userpb.ListDevicesResponse{}, nil
}
func (m *mockUserServiceClient) AddDevice(ctx context.Context, req *userpb.AddDeviceRequest, opts ...grpc.CallOption) (*userpb.AddDeviceResponse, error) {
	return &userpb.AddDeviceResponse{}, nil
}
func (m *mockUserServiceClient) RemoveDevice(ctx context.Context, req *userpb.RemoveDeviceRequest, opts ...grpc.CallOption) (*userpb.RemoveDeviceResponse, error) {
	return &userpb.RemoveDeviceResponse{Message: "ok"}, nil
}
func (m *mockUserServiceClient) SyncDeviceData(ctx context.Context, req *userpb.SyncDeviceDataRequest, opts ...grpc.CallOption) (*userpb.SyncDeviceDataResponse, error) {
	return &userpb.SyncDeviceDataResponse{Message: "ok"}, nil
}
func (m *mockUserServiceClient) GetTrainingStats(ctx context.Context, req *userpb.GetTrainingStatsRequest, opts ...grpc.CallOption) (*userpb.GetTrainingStatsResponse, error) {
	return &userpb.GetTrainingStatsResponse{}, nil
}
func (m *mockUserServiceClient) GetAchievements(ctx context.Context, req *userpb.GetAchievementsRequest, opts ...grpc.CallOption) (*userpb.GetAchievementsResponse, error) {
	return &userpb.GetAchievementsResponse{Achievements: []*userpb.Achievement{}}, nil
}
func (m *mockUserServiceClient) ListUsers(ctx context.Context, req *userpb.ListUsersRequest, opts ...grpc.CallOption) (*userpb.ListUsersResponse, error) {
	return &userpb.ListUsersResponse{}, nil
}
func (m *mockUserServiceClient) ValidateInviteCode(ctx context.Context, req *userpb.ValidateInviteCodeRequest, opts ...grpc.CallOption) (*userpb.ValidateInviteCodeResponse, error) {
	return &userpb.ValidateInviteCodeResponse{IsValid: true}, nil
}
func (m *mockUserServiceClient) SetupTOTP(ctx context.Context, req *userpb.SetupTOTPRequest, opts ...grpc.CallOption) (*userpb.SetupTOTPResponse, error) {
	return &userpb.SetupTOTPResponse{}, nil
}
func (m *mockUserServiceClient) ConfirmTOTP(ctx context.Context, req *userpb.ConfirmTOTPRequest, opts ...grpc.CallOption) (*userpb.ConfirmTOTPResponse, error) {
	return &userpb.ConfirmTOTPResponse{Success: true}, nil
}
func (m *mockUserServiceClient) VerifyTOTP(ctx context.Context, req *userpb.VerifyTOTPRequest, opts ...grpc.CallOption) (*userpb.VerifyTOTPResponse, error) {
	return &userpb.VerifyTOTPResponse{Valid: true}, nil
}
func (m *mockUserServiceClient) DisableTOTP(ctx context.Context, req *userpb.DisableTOTPRequest, opts ...grpc.CallOption) (*userpb.DisableTOTPResponse, error) {
	return &userpb.DisableTOTPResponse{Success: true}, nil
}

func setupGateway() *gateway {
	log := &logger.Logger{Logger: zap.NewNop()}
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	privateKeyBytes, _ := x509.MarshalECPrivateKey(privateKey)
	privateKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes}))
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	publicKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes}))
	return &gateway{
		log:                   log,
		jwtPrivateKeyPEM:      privateKeyPEM,
		jwtPublicKeyPEM:       publicKeyPEM,
		responseSigningSecret: "test-response-secret",
	}
}

func TestRegisterHandler_InvalidJSON(t *testing.T) {
	g := setupGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/register", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")

	g.registerHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterHandler_Success(t *testing.T) {
	g := setupGateway()
	g.userClient = &mockUserServiceClient{}

	w := httptest.NewRecorder()
	reqBody := []byte(`{"email":"test@example.com","password":"password123","full_name":"Test","role":"client"}`)
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/register", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	g.registerHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestLoginHandler_Success(t *testing.T) {
	g := setupGateway()
	g.userClient = &mockUserServiceClient{}

	w := httptest.NewRecorder()
	reqBody := []byte(`{"email":"test@example.com","password":"password123"}`)
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/login", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	g.loginHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "access_token")
}

func TestLoginHandler_InvalidJSON(t *testing.T) {
	g := setupGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/login", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")

	g.loginHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
