// Package config provides configuration utilities including environment variable loading with _FILE suffix support.
package config

import (
	"os"
	"strings"
)

// GetEnv returns environment variable value.
// If KEY_FILE env var is set, reads the file content (trimmed).
// Otherwise returns KEY value.
// If neither is set, returns defaultValue (if provided).
func GetEnv(key string, defaultValue ...string) string {
	fileKey := key + "_FILE"
	if filePath := os.Getenv(fileKey); filePath != "" {
		data, err := os.ReadFile(filePath) //nolint:gosec // G304: intentional file read for Kubernetes secrets
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	if val := os.Getenv(key); val != "" {
		return val
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}
