// Package grpc provides gRPC server utilities with optional mutual TLS.
package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"google.golang.org/grpc/credentials"
)

//nolint:gochecknoglobals
var certCache struct {
	serverCreds credentials.TransportCredentials
	clientCreds credentials.TransportCredentials
	serverOnce  sync.Once
	clientOnce  sync.Once
	serverErr   error
	clientErr   error
}

const (
	defaultCertFile       = "/etc/grpc-tls/tls.crt"
	defaultKeyFile        = "/etc/grpc-tls/tls.key"
	defaultCAFile         = "/etc/grpc-tls/ca.crt"
	defaultClientCertFile = "/etc/grpc-tls/client.crt"
	defaultClientKeyFile  = "/etc/grpc-tls/client.key"
)

func getTLSPath(envKey, defaultPath string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath
	}
	return ""
}

func loadServerTLSCredentials() (credentials.TransportCredentials, error) {
	certFile := getTLSPath("GRPC_TLS_CERT_FILE", defaultCertFile)
	keyFile := getTLSPath("GRPC_TLS_KEY_FILE", defaultKeyFile)
	caFile := getTLSPath("GRPC_TLS_CA_FILE", defaultCAFile)

	if certFile == "" || keyFile == "" {
		return nil, fmt.Errorf("GRPC_TLS_CERT_FILE and GRPC_TLS_KEY_FILE must be set for gRPC TLS")
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load gRPC server TLS cert/key: %w", err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	if caFile != "" {
		caFile = filepath.Clean(caFile)
		if strings.Contains(caFile, "..") {
			return nil, fmt.Errorf("invalid gRPC CA file path")
		}
		caPem, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read gRPC CA file: %w", err)
		}
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caPem) {
			return nil, fmt.Errorf("failed to append gRPC CA cert to pool")
		}
		tlsCfg.ClientCAs = certPool
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return credentials.NewTLS(tlsCfg), nil
}

func loadClientTLSCredentials() (credentials.TransportCredentials, error) {
	caFile := getTLSPath("GRPC_TLS_CA_FILE", defaultCAFile)
	certFile := getTLSPath("GRPC_TLS_CLIENT_CERT_FILE", defaultClientCertFile)
	keyFile := getTLSPath("GRPC_TLS_CLIENT_KEY_FILE", defaultClientKeyFile)

	if caFile == "" {
		return nil, fmt.Errorf("GRPC_TLS_CA_FILE must be set for gRPC client TLS")
	}

	caFile = filepath.Clean(caFile)
	if strings.Contains(caFile, "..") {
		return nil, fmt.Errorf("invalid gRPC CA file path")
	}
	caPem, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read gRPC CA file: %w", err)
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caPem) {
		return nil, fmt.Errorf("failed to append gRPC CA cert to pool")
	}

	tlsCfg := &tls.Config{
		RootCAs: certPool,
	}

	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load gRPC client TLS cert/key: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	return credentials.NewTLS(tlsCfg), nil
}

// GetServerTLSCredentials returns cached gRPC server TLS credentials.
// Returns nil and error if TLS is not configured.
func GetServerTLSCredentials() (credentials.TransportCredentials, error) {
	certCache.serverOnce.Do(func() {
		certCache.serverCreds, certCache.serverErr = loadServerTLSCredentials()
	})
	if certCache.serverErr != nil {
		return nil, certCache.serverErr
	}
	return certCache.serverCreds, nil
}

// GetClientTLSCredentials returns cached gRPC client TLS credentials.
// Returns nil and no error if mTLS is not configured.
func GetClientTLSCredentials() (credentials.TransportCredentials, error) {
	certCache.clientOnce.Do(func() {
		certCache.clientCreds, certCache.clientErr = loadClientTLSCredentials()
	})
	if certCache.clientErr != nil {
		return nil, certCache.clientErr
	}
	return certCache.clientCreds, nil
}
