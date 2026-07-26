package grpc

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	s := NewServer()
	assert.NotNil(t, s)
}

func TestHealthCheckRegistered(t *testing.T) {
	s := NewServer()
	defer s.Stop()

	serviceInfo := s.GetServiceInfo()
	_, ok := serviceInfo["grpc.health.v1.Health"]
	assert.True(t, ok, "Health service should be registered")
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		wantError bool
	}{
		{
			name: "empty config is valid (no TLS)",
			cfg: Config{
				CertFile: "",
				KeyFile:  "",
				CAFile:   "",
			},
			wantError: false,
		},
		{
			name: "valid TLS config",
			cfg: Config{
				CertFile: "server.crt",
				KeyFile:  "server.key",
				CAFile:   "ca.crt",
			},
			wantError: false,
		},
		{
			name: "missing cert file",
			cfg: Config{
				CertFile: "",
				KeyFile:  "server.key",
				CAFile:   "ca.crt",
			},
			wantError: true,
		},
		{
			name: "missing key file",
			cfg: Config{
				CertFile: "server.crt",
				KeyFile:  "",
				CAFile:   "ca.crt",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	require.NoError(t, os.Unsetenv("GRPC_TLS_CERT_FILE"))
	require.NoError(t, os.Unsetenv("GRPC_TLS_KEY_FILE"))
	require.NoError(t, os.Unsetenv("GRPC_TLS_CA_FILE"))

	cfg := LoadConfig()
	assert.Empty(t, cfg.CertFile)
	assert.Empty(t, cfg.KeyFile)
	assert.Empty(t, cfg.CAFile)
}

func TestLoadConfigFromEnv(t *testing.T) {
	require.NoError(t, os.Setenv("GRPC_TLS_CERT_FILE", "server.crt"))
	require.NoError(t, os.Setenv("GRPC_TLS_KEY_FILE", "server.key"))
	require.NoError(t, os.Setenv("GRPC_TLS_CA_FILE", "ca.crt"))

	cfg := LoadConfig()
	assert.Equal(t, "server.crt", cfg.CertFile)
	assert.Equal(t, "server.key", cfg.KeyFile)
	assert.Equal(t, "ca.crt", cfg.CAFile)

	require.NoError(t, os.Unsetenv("GRPC_TLS_CERT_FILE"))
	require.NoError(t, os.Unsetenv("GRPC_TLS_KEY_FILE"))
	require.NoError(t, os.Unsetenv("GRPC_TLS_CA_FILE"))
}

func TestGetServerTLSCredentialsNoCert(t *testing.T) {
	require.NoError(t, os.Unsetenv("GRPC_TLS_CERT_FILE"))
	require.NoError(t, os.Unsetenv("GRPC_TLS_KEY_FILE"))

	creds, err := GetServerTLSCredentials()
	assert.NoError(t, err)
	assert.Nil(t, creds)
}

func TestGetServerTLSCredentialsInvalidCert(t *testing.T) {
	require.NoError(t, os.Setenv("GRPC_TLS_CERT_FILE", "nonexistent.crt"))
	require.NoError(t, os.Setenv("GRPC_TLS_KEY_FILE", "nonexistent.key"))
	defer func() {
		require.NoError(t, os.Unsetenv("GRPC_TLS_CERT_FILE"))
		require.NoError(t, os.Unsetenv("GRPC_TLS_KEY_FILE"))
	}()

	creds, err := GetServerTLSCredentials()
	assert.Error(t, err)
	assert.Nil(t, creds)
}

func TestGetClientTLSCredentialsNoCA(t *testing.T) {
	require.NoError(t, os.Unsetenv("GRPC_TLS_CA_FILE"))

	creds, err := GetClientTLSCredentials()
	assert.NoError(t, err)
	assert.Nil(t, creds)
}

func TestGetClientTLSCredentialsInvalidCA(t *testing.T) {
	require.NoError(t, os.Setenv("GRPC_TLS_CA_FILE", "nonexistent.ca"))
	defer func() {
		require.NoError(t, os.Unsetenv("GRPC_TLS_CA_FILE"))
	}()

	creds, err := GetClientTLSCredentials()
	assert.Error(t, err)
	assert.Nil(t, creds)
}
