// Package grpc provides gRPC server utilities with optional mutual TLS.
package grpc

import (
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/MAMUER/project/internal/config"
)

// Config holds gRPC TLS configuration.
type Config struct {
	CertFile string
	KeyFile  string
	CAFile   string
}

// Validate checks that the gRPC configuration is valid.
func (c Config) Validate() error {
	if c.CertFile == "" && c.KeyFile == "" {
		return nil
	}
	if c.CertFile == "" {
		return errors.New("GRPC_TLS_CERT_FILE is required when TLS is enabled")
	}
	if c.KeyFile == "" {
		return errors.New("GRPC_TLS_KEY_FILE is required when TLS is enabled")
	}
	return nil
}

// LoadConfig loads gRPC TLS configuration from environment variables.
// Supports _FILE suffix for Docker/Kubernetes secrets.
func LoadConfig() Config {
	return Config{
		CertFile: config.GetEnv("GRPC_TLS_CERT_FILE"),
		KeyFile:  config.GetEnv("GRPC_TLS_KEY_FILE"),
		CAFile:   config.GetEnv("GRPC_TLS_CA_FILE"),
	}
}

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

// NewClient creates a gRPC client connection with optional TLS.
// If TLS is not configured, it falls back to insecure connection.
func NewClient(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	creds, err := GetClientTLSCredentials()
	var dialOpts []grpc.DialOption
	if err != nil {
		return nil, err
	}
	if creds != nil {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	dialOpts = append(dialOpts, opts...)
	return grpc.NewClient(target, dialOpts...)
}
