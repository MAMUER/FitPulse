package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    CacheConfig
		wantError bool
	}{
		{
			name:      "valid config",
			config:    CacheConfig{Addr: "localhost:6379", Password: "secret", DB: 0},
			wantError: false,
		},
		{
			name:      "empty addr",
			config:    CacheConfig{Addr: "", Password: "secret", DB: 0},
			wantError: true,
		},
		{
			name:      "empty password is valid",
			config:    CacheConfig{Addr: "localhost:6379", Password: "", DB: 0},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestJWTConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    JWTConfig
		wantError bool
	}{
		{
			name:      "valid config",
			config:    JWTConfig{PrivateKeyPEM: "private", PublicKeyPEM: "public"},
			wantError: false,
		},
		{
			name:      "empty private key",
			config:    JWTConfig{PrivateKeyPEM: "", PublicKeyPEM: "public"},
			wantError: true,
		},
		{
			name:      "empty public key",
			config:    JWTConfig{PrivateKeyPEM: "private", PublicKeyPEM: ""},
			wantError: true,
		},
		{
			name:      "both empty",
			config:    JWTConfig{PrivateKeyPEM: "", PublicKeyPEM: ""},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    ServerConfig
		wantError bool
	}{
		{
			name:      "valid config",
			config:    ServerConfig{Addr: ":8080"},
			wantError: false,
		},
		{
			name:      "empty addr",
			config:    ServerConfig{Addr: ""},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoadCacheConfig(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		require.NoError(t, os.Unsetenv("VALKEY_ADDR"))
		require.NoError(t, os.Unsetenv("VALKEY_PASSWORD"))
		require.NoError(t, os.Unsetenv("VALKEY_DB"))

		cfg := LoadCacheConfig()
		assert.Equal(t, "localhost:6379", cfg.Addr)
		assert.Empty(t, cfg.Password)
		assert.Equal(t, 0, cfg.DB)
	})

	t.Run("custom values", func(t *testing.T) {
		require.NoError(t, os.Setenv("VALKEY_ADDR", "valkey:6379"))
		require.NoError(t, os.Setenv("VALKEY_PASSWORD", "secret"))
		require.NoError(t, os.Setenv("VALKEY_DB", "1"))
		t.Cleanup(func() {
			require.NoError(t, os.Unsetenv("VALKEY_ADDR"))
			require.NoError(t, os.Unsetenv("VALKEY_PASSWORD"))
			require.NoError(t, os.Unsetenv("VALKEY_DB"))
		})

		cfg := LoadCacheConfig()
		assert.Equal(t, "valkey:6379", cfg.Addr)
		assert.Equal(t, "secret", cfg.Password)
		assert.Equal(t, 1, cfg.DB)
	})
}

func TestLoadJWTConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		require.NoError(t, os.Setenv("JWT_PRIVATE_KEY_PEM", "private-key"))
		require.NoError(t, os.Setenv("JWT_PUBLIC_KEY_PEM", "public-key"))
		t.Cleanup(func() {
			require.NoError(t, os.Unsetenv("JWT_PRIVATE_KEY_PEM"))
			require.NoError(t, os.Unsetenv("JWT_PUBLIC_KEY_PEM"))
		})

		cfg := LoadJWTConfig()
		assert.Equal(t, "private-key", cfg.PrivateKeyPEM)
		assert.Equal(t, "public-key", cfg.PublicKeyPEM)
	})

	t.Run("panics on missing private key", func(t *testing.T) {
		require.NoError(t, os.Unsetenv("JWT_PRIVATE_KEY_PEM"))
		require.NoError(t, os.Unsetenv("JWT_PUBLIC_KEY_PEM"))
		assert.Panics(t, func() {
			LoadJWTConfig()
		})
	})
}

func TestLoadServerConfig(t *testing.T) {
	t.Run("default value", func(t *testing.T) {
		require.NoError(t, os.Unsetenv("TEST_SERVER_ADDR"))
		cfg := LoadServerConfig("TEST_SERVER_ADDR", ":8080")
		assert.Equal(t, ":8080", cfg.Addr)
	})

	t.Run("custom value", func(t *testing.T) {
		require.NoError(t, os.Setenv("TEST_SERVER_ADDR", ":9090"))
		t.Cleanup(func() { require.NoError(t, os.Unsetenv("TEST_SERVER_ADDR")) })

		cfg := LoadServerConfig("TEST_SERVER_ADDR", ":8080")
		assert.Equal(t, ":9090", cfg.Addr)
	})
}

func TestRedactSecrets(t *testing.T) {
	t.Run("redacts cache password", func(t *testing.T) {
		cfg := CacheConfig{Addr: "localhost:6379", Password: "secret", DB: 0}
		redacted := redactSecrets(cfg)
		cacheCfg, ok := redacted.(struct {
			Addr     string
			Password string
			DB       int
		})
		require.True(t, ok)
		assert.Equal(t, "localhost:6379", cacheCfg.Addr)
		assert.Equal(t, "[REDACTED]", cacheCfg.Password)
		assert.Equal(t, 0, cacheCfg.DB)
	})

	t.Run("redacts JWT keys", func(t *testing.T) {
		cfg := JWTConfig{PrivateKeyPEM: "private", PublicKeyPEM: "public"}
		redacted := redactSecrets(cfg)
		jwtCfg, ok := redacted.(struct {
			PrivateKeyPEM string
			PublicKeyPEM  string
		})
		require.True(t, ok)
		assert.Equal(t, "[REDACTED]", jwtCfg.PrivateKeyPEM)
		assert.Equal(t, "[REDACTED]", jwtCfg.PublicKeyPEM)
	})
}
