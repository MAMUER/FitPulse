package middleware

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/MAMUER/project/internal/auth/jwt"
)

var (
	testPrivateKeyPEMGRPC string
	testPublicKeyPEMGRPC  string
)

func init() {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	b, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		panic(err)
	}
	testPrivateKeyPEMGRPC = string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b}))
	pub, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		panic(err)
	}
	testPublicKeyPEMGRPC = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pub}))
}

func generateValidTokenGRPC() string {
	token, err := jwt.GenerateAccessToken("user-123", "test@example.com", "client", testPrivateKeyPEMGRPC, 15*time.Minute)
	if err != nil {
		panic(err)
	}
	return token
}

func TestGRPCAuthInterceptor_ValidToken(t *testing.T) {
	log := zap.NewNop()
	interceptor := GRPCAuthInterceptor(testPublicKeyPEMGRPC, log)

	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		userID := ctx.Value(UserIDKey)
		role := ctx.Value(RoleKey)
		assert.Equal(t, "user-123", userID)
		assert.Equal(t, "client", role)
		return "ok", nil
	}

	md := metadata.New(map[string]string{
		authMetadataKey: "Bearer " + generateValidTokenGRPC(),
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test.Method"}, handler)

	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
	assert.True(t, handlerCalled)
}

func TestGRPCAuthInterceptor_MissingMetadata(t *testing.T) {
	log := zap.NewNop()
	interceptor := GRPCAuthInterceptor(testPublicKeyPEMGRPC, log)

	resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test.Method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	})

	assert.Nil(t, resp)
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "missing metadata")
}

func TestGRPCAuthInterceptor_MissingAuthHeader(t *testing.T) {
	log := zap.NewNop()
	interceptor := GRPCAuthInterceptor(testPublicKeyPEMGRPC, log)

	md := metadata.New(map[string]string{})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test.Method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	})

	assert.Nil(t, resp)
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "missing authorization header")
}

func TestGRPCAuthInterceptor_InvalidFormat(t *testing.T) {
	log := zap.NewNop()
	interceptor := GRPCAuthInterceptor(testPublicKeyPEMGRPC, log)

	md := metadata.New(map[string]string{
		authMetadataKey: "InvalidFormat",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test.Method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	})

	assert.Nil(t, resp)
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "invalid authorization format")
}

func TestGRPCAuthInterceptor_WrongPrefix(t *testing.T) {
	log := zap.NewNop()
	interceptor := GRPCAuthInterceptor(testPublicKeyPEMGRPC, log)

	md := metadata.New(map[string]string{
		authMetadataKey: "Basic token",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test.Method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	})

	assert.Nil(t, resp)
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "invalid authorization format")
}

func TestGRPCAuthInterceptor_InvalidToken(t *testing.T) {
	log := zap.NewNop()
	interceptor := GRPCAuthInterceptor(testPublicKeyPEMGRPC, log)

	md := metadata.New(map[string]string{
		authMetadataKey: "Bearer invalid.token.string",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test.Method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	})

	assert.Nil(t, resp)
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "invalid token")
}

func TestGRPCAuthInterceptor_EmptyAuthValue(t *testing.T) {
	log := zap.NewNop()
	interceptor := GRPCAuthInterceptor(testPublicKeyPEMGRPC, log)

	md := metadata.New(map[string]string{
		authMetadataKey: "",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test.Method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	})

	assert.Nil(t, resp)
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "missing authorization header")
}
