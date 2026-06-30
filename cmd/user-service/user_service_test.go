package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	pb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func setupUserService(db *sql.DB) *userServer {
	zapLog, _ := zap.NewDevelopment()
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	privateKeyBytes, _ := x509.MarshalECPrivateKey(privateKey)
	privateKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes}))
	return &userServer{
		db:               db,
		log:              &logger.Logger{Logger: zapLog},
		jwtPrivateKeyPEM: privateKeyPEM,
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
