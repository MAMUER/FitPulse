package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultEnv(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		fallback string
		want     string
	}{
		{"env set", "TEST_KEY", "value", "default", "value"},
		{"env empty", "TEST_KEY", "", "default", "default"},
		{"env unset", "UNSET_KEY", "", "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envKey != "UNSET_KEY" {
				t.Setenv(tt.envKey, tt.envValue)
			}
			got := defaultEnv(tt.envKey, tt.fallback)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEnvBool(t *testing.T) {
	tests := []struct {
		name   string
		envKey string
		value  string
		want   bool
	}{
		{"true", "TEST_BOOL", "true", true},
		{"True", "TEST_BOOL", "True", true},
		{"1", "TEST_BOOL", "1", true},
		{"false", "TEST_BOOL", "false", false},
		{"0", "TEST_BOOL", "0", false},
		{"empty", "UNSET_BOOL", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name != "empty" {
				t.Setenv(tt.envKey, tt.value)
			}
			got := envBool(tt.envKey)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRedisAddress(t *testing.T) {
	t.Run("default redis host", func(t *testing.T) {
		t.Setenv("REDIS_HOST", "")
		got := redisAddress()
		assert.Equal(t, "redis:6379", got)
	})

	t.Run("custom redis host", func(t *testing.T) {
		t.Setenv("REDIS_HOST", "custom-redis")
		got := redisAddress()
		assert.Equal(t, "custom-redis:6379", got)
	})
}

func TestPublicHostFromBaseURL(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		wantEmpty bool
		wantHost  string
	}{
		{"empty base URL", "", true, ""},
		{"invalid URL", "not-a-url", true, ""},
		{"valid URL", "https://example.com", false, "example.com"},
		{"URL with port", "https://example.com:8443", false, "example.com:8443"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := publicHostFromBaseURL(tt.baseURL)
			if tt.wantEmpty {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tt.wantHost, got)
			}
		})
	}
}

func TestReadSecretFile_InvalidPath(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{"dot path", ".", true},
		{"separator path", string(filepath.Separator), true},
		{"empty path", "", true},
		{"nonexistent file", "/nonexistent/path/to/file.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := readSecretFile(tt.filePath)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
