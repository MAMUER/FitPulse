package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/grpc/credentials"
)

func GetServerTLSCredentials() (credentials.TransportCredentials, error) {
	cfg := LoadConfig()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if cfg.CertFile == "" || cfg.KeyFile == "" {
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	if cfg.CAFile != "" {
		cleanCAFile := filepath.Clean(cfg.CAFile)
		if strings.Contains(cleanCAFile, "..") {
			return nil, errors.New("invalid gRPC CA file path")
		}
		caPem, err := os.ReadFile(cleanCAFile)
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
	cfg := LoadConfig()
	if cfg.CAFile == "" {
		return nil, nil
	}

	cleanCAFile := filepath.Clean(cfg.CAFile)
	if strings.Contains(cleanCAFile, "..") {
		return nil, errors.New("invalid gRPC CA file path")
	}

	caPem, err := os.ReadFile(cleanCAFile)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caPem) {
		return nil, errors.New("failed to append gRPC CA cert to pool")
	}

	tlsConfig := &tls.Config{
		RootCAs:    certPool,
		MinVersion: tls.VersionTLS13,
	}

	return credentials.NewTLS(tlsConfig), nil
}
