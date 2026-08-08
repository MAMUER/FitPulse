package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/auth/jwt"
	"github.com/MAMUER/project/internal/logger"
)

func setupUserService(db *sql.DB) *userServer {
	zapLog, _ := zap.NewDevelopment()
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	privateKeyBytes, _ := x509.MarshalECPrivateKey(privateKey)
	privateKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes}))
	return &userServer{
		db:            db,
		log:           &logger.Logger{Logger: zapLog},
		tokenProvider: jwt.NewJWTAdapter(privateKeyPEM, ""),
	}
}

func TestUserServer_Register_InvalidInput(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	s := setupUserService(db)

	tests := []struct {
		name    string
		req     *pb.RegisterRequest
		wantMsg string
	}{
		{
			name:    "empty email",
			req:     &pb.RegisterRequest{Email: "", Password: "password123", FullName: "Test", Role: "client"},
			wantMsg: "email is required",
		},
		{
			name:    "short password",
			req:     &pb.RegisterRequest{Email: "test@example.com", Password: "short", FullName: "Test", Role: "client"},
			wantMsg: "password must be at least",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.Register(context.Background(), tt.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.InvalidArgument, st.Code())
			assert.Contains(t, st.Message(), tt.wantMsg)
		})
	}
}

func TestUserServer_GetProfile_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT u.id").WithArgs("missing-id").WillReturnError(sql.ErrNoRows)

	s := setupUserService(db)

	_, err = s.GetProfile(context.Background(), &pb.GetProfileRequest{UserId: "missing-id"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestUserServer_ChangePassword_Validation(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	s := setupUserService(db)

	tests := []struct {
		name     string
		req      *pb.ChangePasswordRequest
		wantMsg  string
		wantCode codes.Code
	}{
		{
			name:     "empty user_id",
			req:      &pb.ChangePasswordRequest{UserId: "", CurrentPassword: "old", NewPassword: "NewPass1"},
			wantMsg:  "user_id is required",
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "empty current_password",
			req:      &pb.ChangePasswordRequest{UserId: "user-1", CurrentPassword: "", NewPassword: "NewPass1"},
			wantMsg:  "current_password is required",
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "empty new_password",
			req:      &pb.ChangePasswordRequest{UserId: "user-1", CurrentPassword: "old", NewPassword: ""},
			wantMsg:  "new_password is required",
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "short new_password",
			req:      &pb.ChangePasswordRequest{UserId: "user-1", CurrentPassword: "old", NewPassword: "short"},
			wantMsg:  "new password must be at least 8 characters",
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "weak new_password",
			req:      &pb.ChangePasswordRequest{UserId: "user-1", CurrentPassword: "old", NewPassword: "nouppercase1"},
			wantMsg:  "new password must contain uppercase, lowercase, and digit",
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.ChangePassword(context.Background(), tt.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, tt.wantCode, st.Code())
			assert.Contains(t, st.Message(), tt.wantMsg)
		})
	}
}

func TestUserServer_ChangePassword_UserNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT password_hash FROM users WHERE id = \\$1").WithArgs("missing-id").WillReturnError(sql.ErrNoRows)

	s := setupUserService(db)

	_, err = s.ChangePassword(context.Background(), &pb.ChangePasswordRequest{
		UserId: "missing-id", CurrentPassword: "old", NewPassword: "NewPass1",
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestUserServer_Register_UserAlreadyExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM users WHERE email_hash = \\$1\\)").WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	s := setupUserService(db)

	_, err = s.Register(context.Background(), &pb.RegisterRequest{
		Email: "test@example.com", Password: "password123", FullName: "Test User", Role: "client",
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.AlreadyExists, st.Code())
	assert.Contains(t, st.Message(), "user already exists")
}

func TestUserServer_ConfirmEmail_InvalidToken(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(`SELECT user_id, .* FROM email_verifications WHERE token = \$1`).WithArgs(sqlmock.AnyArg()).WillReturnError(sql.ErrNoRows)

	s := setupUserService(db)

	_, err = s.ConfirmEmail(context.Background(), &pb.ConfirmEmailRequest{Token: "invalid-token"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "invalid verification token")
}

func TestUserServer_ConfirmEmail_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(`SELECT user_id, .* FROM email_verifications WHERE token = \$1`).WithArgs(sqlmock.AnyArg()).WillReturnError(errors.New("database error"))

	s := setupUserService(db)

	_, err = s.ConfirmEmail(context.Background(), &pb.ConfirmEmailRequest{Token: "some-token"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Contains(t, st.Message(), "database error")
}
