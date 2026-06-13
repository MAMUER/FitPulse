// Package grpc provides gRPC server utilities and interceptors.
package grpc

import (
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// NewServer creates a gRPC server suitable for internal communication.
// TLS is enabled when GRPC_TLS_CERT_FILE and GRPC_TLS_KEY_FILE are set.
func NewServer() *grpc.Server {
	certFile := os.Getenv("GRPC_TLS_CERT_FILE")
	keyFile := os.Getenv("GRPC_TLS_KEY_FILE")
	creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	if err != nil {
		creds = insecure.NewCredentials()
	}

	s := grpc.NewServer(grpc.Creds(creds))
	grpc_health_v1.RegisterHealthServer(s, health.NewServer())
	return s
}
