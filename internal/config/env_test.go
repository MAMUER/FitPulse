package config

import (
	"errors"
	"os"
	"strings"
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

func TestGetEnvWithFile_PathTraversal(t *testing.T) {
	key := "TEST_KEY_PATH_TRAVERSAL"
	require.NoError(t, os.Setenv(key+"_FILE", "/nonexistent/../etc/passwd"))
	t.Cleanup(func() { require.NoError(t, os.Unsetenv(key+"_FILE")) })

	result := GetEnv(key, "default_value")
	assert.Equal(t, "default_value", result)
}

func TestGetEnvWithFile_AbsError(t *testing.T) {
	key := "TEST_KEY_ABS_ERROR"
	longPath := strings.Repeat("a", 300)
	require.NoError(t, os.Setenv(key+"_FILE", longPath))
	t.Cleanup(func() { require.NoError(t, os.Unsetenv(key+"_FILE")) })

	result := GetEnv(key, "default_value")
	assert.Equal(t, "default_value", result)
}

func TestGetEnvWithFile_AbsErrorInvalidPath(t *testing.T) {
	key := "TEST_KEY_ABS_INVALID"
	require.NoError(t, os.Setenv(key+"_FILE", "C:/foo<bar"))
	t.Cleanup(func() { require.NoError(t, os.Unsetenv(key+"_FILE")) })

	result := GetEnv(key, "default_value")
	assert.Equal(t, "default_value", result)
}

func TestGetEnv_NoDefaultValue(t *testing.T) {
	key := "TEST_KEY_NO_DEFAULT"
	require.NoError(t, os.Unsetenv(key))
	require.NoError(t, os.Unsetenv(key+"_FILE"))
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv(key))
		require.NoError(t, os.Unsetenv(key+"_FILE"))
	})

	require.NoError(t, os.Setenv(key+"_FILE", "C:/foo<bar"))

	result := GetEnv(key)
	assert.Equal(t, "", result)
}

func TestGetEnvWithFile_AbsError_NoDefaultValue(t *testing.T) {
	key := "TEST_KEY_ABS_ERROR_NO_DEFAULT"
	longPath := strings.Repeat("a", 300)
	require.NoError(t, os.Setenv(key+"_FILE", longPath))
	t.Cleanup(func() { require.NoError(t, os.Unsetenv(key+"_FILE")) })

	result := GetEnv(key)
	assert.Equal(t, "", result)
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

type envGetterCase[T any] struct {
	name         string
	key          string
	value        string
	defaultValue T
	expected     T
}

func runEnvGetterCases[T any](t *testing.T, get func(string, T) T, cases []envGetterCase[T]) {
	t.Helper()
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				require.NoError(t, os.Setenv(tt.key, tt.value))
				t.Cleanup(func() { require.NoError(t, os.Unsetenv(tt.key)) })
			} else {
				require.NoError(t, os.Unsetenv(tt.key))
			}

			assert.Equal(t, tt.expected, get(tt.key, tt.defaultValue))
		})
	}
}

func TestEnvTypedGetters(t *testing.T) {
	t.Run("GetEnvInt", func(t *testing.T) {
		runEnvGetterCases(t, GetEnvInt, []envGetterCase[int]{
			{name: "valid int", key: "TEST_INT_VALID", value: "42", defaultValue: 0, expected: 42},
			{name: "invalid int returns default", key: "TEST_INT_INVALID", value: "abc", defaultValue: 10, expected: 10},
			{name: "empty returns default", key: "TEST_INT_EMPTY", value: "", defaultValue: 99, expected: 99},
			{name: "negative int", key: "TEST_INT_NEGATIVE", value: "-5", defaultValue: 0, expected: -5},
		})
	})

	t.Run("GetEnvInt64", func(t *testing.T) {
		runEnvGetterCases(t, GetEnvInt64, []envGetterCase[int64]{
			{name: "valid int64", key: "TEST_INT64_VALID", value: "9223372036854775807", defaultValue: 0, expected: 9223372036854775807},
			{name: "invalid int64 returns default", key: "TEST_INT64_INVALID", value: "abc", defaultValue: 10, expected: 10},
			{name: "empty returns default", key: "TEST_INT64_EMPTY", value: "", defaultValue: 99, expected: 99},
		})
	})

	t.Run("GetEnvBool", func(t *testing.T) {
		runEnvGetterCases(t, GetEnvBool, []envGetterCase[bool]{
			{name: "true", key: "TEST_BOOL_TRUE", value: "true", defaultValue: false, expected: true},
			{name: "false", key: "TEST_BOOL_FALSE", value: "false", defaultValue: true, expected: false},
			{name: "1", key: "TEST_BOOL_1", value: "1", defaultValue: false, expected: true},
			{name: "0", key: "TEST_BOOL_0", value: "0", defaultValue: true, expected: false},
			{name: "yes", key: "TEST_BOOL_YES", value: "yes", defaultValue: false, expected: true},
			{name: "no", key: "TEST_BOOL_NO", value: "no", defaultValue: true, expected: false},
			{name: "on", key: "TEST_BOOL_ON", value: "on", defaultValue: false, expected: true},
			{name: "off", key: "TEST_BOOL_OFF", value: "off", defaultValue: true, expected: false},
			{name: "invalid returns default", key: "TEST_BOOL_INVALID", value: "maybe", defaultValue: true, expected: true},
			{name: "empty returns default", key: "TEST_BOOL_EMPTY", value: "", defaultValue: false, expected: false},
		})
	})

	t.Run("GetEnvDuration", func(t *testing.T) {
		runEnvGetterCases(t, GetEnvDuration, []envGetterCase[time.Duration]{
			{name: "valid seconds", key: "TEST_DURATION_SEC", value: "5s", defaultValue: 0, expected: 5 * time.Second},
			{name: "valid milliseconds", key: "TEST_DURATION_MS", value: "100ms", defaultValue: 0, expected: 100 * time.Millisecond},
			{name: "valid hours", key: "TEST_DURATION_HOUR", value: "1h", defaultValue: 0, expected: time.Hour},
			{name: "invalid returns default", key: "TEST_DURATION_INVALID", value: "abc", defaultValue: 10 * time.Second, expected: 10 * time.Second},
			{name: "empty returns default", key: "TEST_DURATION_EMPTY", value: "", defaultValue: 99 * time.Second, expected: 99 * time.Second},
		})
	})

	t.Run("GetEnvFloat64", func(t *testing.T) {
		runEnvGetterCases(t, GetEnvFloat64, []envGetterCase[float64]{
			{name: "valid float", key: "TEST_FLOAT_VALID", value: "3.14", defaultValue: 0, expected: 3.14},
			{name: "invalid float returns default", key: "TEST_FLOAT_INVALID", value: "abc", defaultValue: 10.5, expected: 10.5},
			{name: "empty returns default", key: "TEST_FLOAT_EMPTY", value: "", defaultValue: 99.9, expected: 99.9},
			{name: "negative float", key: "TEST_FLOAT_NEGATIVE", value: "-2.5", defaultValue: 0, expected: -2.5},
		})
	})
}

func TestGetEnv_AbsPathFailure(t *testing.T) {
	key := "TEST_KEY_ABS_FAILURE"
	require.NoError(t, os.Setenv(key+"_FILE", "some/path"))
	t.Cleanup(func() { require.NoError(t, os.Unsetenv(key+"_FILE")) })

	oldAbsPath := absPath
	absPath = func(string) (string, error) { return "", errors.New("abs failed") }
	t.Cleanup(func() { absPath = oldAbsPath })

	result := GetEnv(key)
	assert.Equal(t, "", result)
}

func TestDefaultValueOrDefault_Empty(t *testing.T) {
	result := defaultValueOrDefault(nil)
	assert.Equal(t, "", result)
}
