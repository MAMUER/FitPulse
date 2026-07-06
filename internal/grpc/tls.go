// Package grpc provides gRPC server utilities with optional mutual TLS.
package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"google.golang.org/grpc/credentials"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// GetServerTLSCredentials returns TLS credentials for the gRPC server.
func GetServerTLSCredentials() (credentials.TransportCredentials, error) {
	certFile := os.Getenv("GRPC_TLS_CERT_FILE")
	keyFile := os.Getenv("GRPC_TLS_KEY_FILE")
	caFile := os.Getenv("GRPC_TLS_CA_FILE")

	if certFile == "" || keyFile == "" {
		return nil, nil // No TLS configured
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	if caFile != "" {
		if strings.Contains(caFile, "..") {
			return nil, errors.New("invalid gRPC CA file path")
		}
		caPem, err := os.ReadFile(caFile)
		if err != nil {
			return nil, err
		}
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caPem) {
			return nil, errors.New("failed to append gRPC CA cert to pool")
		}
		tlsConfig.ClientCAs = certPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return credentials.NewTLS(tlsConfig), nil
}

func GetClientTLSCredentials() (credentials.TransportCredentials, error) {
	caFile := os.Getenv("GRPC_TLS_CA_FILE")
	if caFile == "" {
		return nil, nil // No TLS configured
	}

	if strings.Contains(caFile, "..") {
		return nil, errors.New("invalid gRPC CA file path")
	}

	caPem, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caPem) {
		return nil, errors.New("failed to append gRPC CA cert to pool")
	}

	tlsConfig := &tls.Config{
		RootCAs:    certPool,
		MinVersion: tls.VersionTLS12,
	}

	return credentials.NewTLS(tlsConfig), nil
}
