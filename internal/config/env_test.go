package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		setupEnv func(string)
		expected string
	}{
		{
			name:     "no env set - returns default",
			key:      "TEST_KEY_EMPTY",
			setupEnv: func(string) {},
			expected: "default_value",
		},
		{
			name:     "env set directly - returns value",
			key:      "TEST_KEY_DIRECT",
			setupEnv: func(key string) { require.NoError(t, os.Setenv(key, "direct_value")) },
			expected: "direct_value",
		},
		{
			name: "both env and _FILE set - returns direct value priority",
			key:  "TEST_KEY_BOTH",
			setupEnv: func(key string) {
				require.NoError(t, os.Setenv(key, "direct"))
				require.NoError(t, os.Setenv(key+"_FILE", "/nonexistent"))
			},
			expected: "direct",
		},
		{
			name:     "default value when nothing set",
			key:      "TEST_KEY_DEFAULT",
			setupEnv: func(string) {},
			expected: "default_value",
		},
		{
			name:     "env value preserved as-is",
			key:      "TEST_KEY_TRIM",
			setupEnv: func(key string) { require.NoError(t, os.Setenv(key, "  trimmed_value  ")) },
			expected: "  trimmed_value  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, os.Unsetenv(tt.key))
			require.NoError(t, os.Unsetenv(tt.key+"_FILE"))
			tt.setupEnv(tt.key)
			t.Cleanup(func() {
				require.NoError(t, os.Unsetenv(tt.key))
				require.NoError(t, os.Unsetenv(tt.key+"_FILE"))
			})
			result := GetEnv(tt.key, "default_value")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvWithFile(t *testing.T) {
	projectDir, err := os.Getwd()
	require.NoError(t, err)
	tmpFile, err := os.CreateTemp(projectDir, "test_secret_*")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, os.Remove(tmpFile.Name())) })

	_, err = tmpFile.WriteString("secret_from_file")
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	key := "TEST_KEY_FROM_FILE"
	require.NoError(t, os.Setenv(key+"_FILE", tmpFile.Name()))
	t.Cleanup(func() { require.NoError(t, os.Unsetenv(key+"_FILE")) })

	result := GetEnv(key, "default_value")
	assert.Equal(t, "secret_from_file", result)
}
