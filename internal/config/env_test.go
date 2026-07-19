package config

import (
	"os"
	"testing"
	"time"

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

func TestGetEnvRequired(t *testing.T) {
	t.Run("returns value when set", func(t *testing.T) {
		require.NoError(t, os.Setenv("TEST_REQUIRED_KEY", "value"))
		t.Cleanup(func() { require.NoError(t, os.Unsetenv("TEST_REQUIRED_KEY")) })

		val := GetEnvRequired("TEST_REQUIRED_KEY")
		assert.Equal(t, "value", val)
	})

	t.Run("panics when not set", func(t *testing.T) {
		require.NoError(t, os.Unsetenv("TEST_REQUIRED_MISSING"))
		assert.Panics(t, func() {
			GetEnvRequired("TEST_REQUIRED_MISSING")
		})
	})
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue int
		expected     int
	}{
		{"valid int", "TEST_INT_VALID", "42", 0, 42},
		{"invalid int returns default", "TEST_INT_INVALID", "abc", 10, 10},
		{"empty returns default", "TEST_INT_EMPTY", "", 99, 99},
		{"negative int", "TEST_INT_NEGATIVE", "-5", 0, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				require.NoError(t, os.Setenv(tt.key, tt.value))
				t.Cleanup(func() { require.NoError(t, os.Unsetenv(tt.key)) })
			} else {
				require.NoError(t, os.Unsetenv(tt.key))
			}

			result := GetEnvInt(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvInt64(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue int64
		expected     int64
	}{
		{"valid int64", "TEST_INT64_VALID", "9223372036854775807", 0, 9223372036854775807},
		{"invalid int64 returns default", "TEST_INT64_INVALID", "abc", 10, 10},
		{"empty returns default", "TEST_INT64_EMPTY", "", 99, 99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				require.NoError(t, os.Setenv(tt.key, tt.value))
				t.Cleanup(func() { require.NoError(t, os.Unsetenv(tt.key)) })
			} else {
				require.NoError(t, os.Unsetenv(tt.key))
			}

			result := GetEnvInt64(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue bool
		expected     bool
	}{
		{"true", "TEST_BOOL_TRUE", "true", false, true},
		{"false", "TEST_BOOL_FALSE", "false", true, false},
		{"1", "TEST_BOOL_1", "1", false, true},
		{"0", "TEST_BOOL_0", "0", true, false},
		{"yes", "TEST_BOOL_YES", "yes", false, true},
		{"no", "TEST_BOOL_NO", "no", true, false},
		{"on", "TEST_BOOL_ON", "on", false, true},
		{"off", "TEST_BOOL_OFF", "off", true, false},
		{"invalid returns default", "TEST_BOOL_INVALID", "maybe", true, true},
		{"empty returns default", "TEST_BOOL_EMPTY", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				require.NoError(t, os.Setenv(tt.key, tt.value))
				t.Cleanup(func() { require.NoError(t, os.Unsetenv(tt.key)) })
			} else {
				require.NoError(t, os.Unsetenv(tt.key))
			}

			result := GetEnvBool(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue time.Duration
		expected     time.Duration
	}{
		{"valid seconds", "TEST_DURATION_SEC", "5s", 0, 5 * time.Second},
		{"valid milliseconds", "TEST_DURATION_MS", "100ms", 0, 100 * time.Millisecond},
		{"valid hours", "TEST_DURATION_HOUR", "1h", 0, time.Hour},
		{"invalid returns default", "TEST_DURATION_INVALID", "abc", 10 * time.Second, 10 * time.Second},
		{"empty returns default", "TEST_DURATION_EMPTY", "", 99 * time.Second, 99 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				require.NoError(t, os.Setenv(tt.key, tt.value))
				t.Cleanup(func() { require.NoError(t, os.Unsetenv(tt.key)) })
			} else {
				require.NoError(t, os.Unsetenv(tt.key))
			}

			result := GetEnvDuration(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvFloat64(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue float64
		expected     float64
	}{
		{"valid float", "TEST_FLOAT_VALID", "3.14", 0, 3.14},
		{"invalid float returns default", "TEST_FLOAT_INVALID", "abc", 10.5, 10.5},
		{"empty returns default", "TEST_FLOAT_EMPTY", "", 99.9, 99.9},
		{"negative float", "TEST_FLOAT_NEGATIVE", "-2.5", 0, -2.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				require.NoError(t, os.Setenv(tt.key, tt.value))
				t.Cleanup(func() { require.NoError(t, os.Unsetenv(tt.key)) })
			} else {
				require.NoError(t, os.Unsetenv(tt.key))
			}

			result := GetEnvFloat64(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}
