package grpc

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// NewServer creates a gRPC server suitable for internal communication.
// TLS is enabled when GRPC_TLS_CERT_FILE and GRPC_TLS_KEY_FILE are set.
func NewServer(opts ...grpc.ServerOption) *grpc.Server {
	creds, err := GetServerTLSCredentials()
	var serverOpts []grpc.ServerOption
	if err == nil && creds != nil {
		serverOpts = append(serverOpts, grpc.Creds(creds))
	}
	serverOpts = append(serverOpts, opts...)
	s := grpc.NewServer(serverOpts...)
	grpc_health_v1.RegisterHealthServer(s, health.NewServer())
	return s
}
