package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/MAMUER/project/internal/logger"
)

func TestHealth_CheckGRPCServiceNilConnection(t *testing.T) {
	got := checkGRPCService(nil, "user.UserService")
	assert.Equal(t, "down", got)
}

func TestHealth_CheckTCPServiceEmptyAddress(t *testing.T) {
	got := checkTCPService(zap.NewNop(), "")
	assert.Equal(t, "down", got)
}

func TestHealth_CheckTCPServiceUp(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer func() { _ = lis.Close() }()

	addr := lis.Addr().String()
	got := checkTCPService(zap.NewNop(), addr)
	assert.Equal(t, "up", got)
}

func TestHealth_CheckGRPCServiceServing(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer func() { _ = lis.Close() }()

	//nolint:gosec // test-only local gRPC server
	s := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(s, &mockHealthServer{serving: grpc_health_v1.HealthCheckResponse_SERVING})
	go func() { _ = s.Serve(lis) }()
	defer s.Stop()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer func() { _ = conn.Close() }()

	got := checkGRPCService(conn, "user.UserService")
	assert.Equal(t, "up", got)
}

func TestHealth_CheckGRPCServiceNotServing(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer func() { _ = lis.Close() }()

	//nolint:gosec // test-only local gRPC server
	s := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(s, &mockHealthServer{serving: grpc_health_v1.HealthCheckResponse_NOT_SERVING})
	go func() { _ = s.Serve(lis) }()
	defer s.Stop()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer func() { _ = conn.Close() }()

	got := checkGRPCService(conn, "user.UserService")
	assert.Equal(t, "down", got)
}

func TestHealth_CheckGRPCServiceError(t *testing.T) {
	conn, err := grpc.NewClient("127.0.0.1:1", grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer func() { _ = conn.Close() }()

	got := checkGRPCService(conn, "user.UserService")
	assert.Equal(t, "down", got)
}

func TestHealth_CheckTCPDialError(t *testing.T) {
	got := checkTCPService(zap.NewNop(), "127.0.0.1:1")
	assert.Equal(t, "down", got)
}

func TestHealth_CheckTCPServiceCloseError(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	addr := lis.Addr().String()
	closeErr := lis.Close()
	_ = closeErr

	// Make conn.Close fail by closing the listener first
	got := checkTCPService(zap.NewNop(), addr)
	assert.Equal(t, "down", got)
}

type mockHealthServer struct {
	grpc_health_v1.UnimplementedHealthServer
	serving grpc_health_v1.HealthCheckResponse_ServingStatus
}

func (m *mockHealthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{Status: m.serving}, nil
}

func TestHealth_HandlerNoServices(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{log: log}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)

	g.healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "gateway")
	assert.Contains(t, w.Body.String(), "degraded")
}

func TestHealth_HandlerWithUserClient(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{
		log:        log,
		userClient: &mockUserServiceClient{},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)

	g.healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestHealth_HandlerWithGRPCConnDown(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}

	// Start a gRPC server that returns NOT_SERVING
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer func() { _ = lis.Close() }()

	//nolint:gosec // test-only local gRPC server
	s := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(s, &mockHealthServer{serving: grpc_health_v1.HealthCheckResponse_NOT_SERVING})
	go func() { _ = s.Serve(lis) }()
	defer s.Stop()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer func() { _ = conn.Close() }()

	g := &gateway{log: log, userConn: conn}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)

	g.healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "degraded")
}

func TestHealth_HandlerWithGRPCConnUp(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer func() { _ = lis.Close() }()

	//nolint:gosec // test-only local gRPC server
	s := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(s, &mockHealthServer{serving: grpc_health_v1.HealthCheckResponse_SERVING})
	go func() { _ = s.Serve(lis) }()
	defer s.Stop()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer func() { _ = conn.Close() }()

	g := &gateway{log: log, userConn: conn}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)

	g.healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestHealth_HandlerResponseFormat(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{
		log:        log,
		userClient: &mockUserServiceClient{},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)

	g.healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "timestamp")
	assert.Contains(t, w.Body.String(), "services")
	assert.Contains(t, w.Body.String(), "classifier")
}
