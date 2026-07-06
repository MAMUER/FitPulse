package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
	"strings"

	"google.golang.org/grpc/credentials"
)

func GetServerTLSCredentials() (credentials.TransportCredentials, error) {
	certFile := os.Getenv("GRPC_TLS_CERT_FILE")
	keyFile := os.Getenv("GRPC_TLS_KEY_FILE")
	caFile := os.Getenv("GRPC_TLS_CA_FILE")

	if certFile == "" || keyFile == "" {
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

		tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
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
		return nil, nil
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
		MinVersion: tls.VersionTLS13,
	}

	return credentials.NewTLS(tlsConfig), nil
}
