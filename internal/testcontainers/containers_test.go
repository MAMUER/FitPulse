package testcontainers

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestParsePort(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    int
		expectError bool
	}{
		{"standard port", "5432", 5432, false},
		{"small port", "1", 1, false},
		{"large port", "65535", 65535, false},
		{"invalid port", "not-a-port", 0, true},
		{"empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, err := parsePort(tt.input)
			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, port)
		})
	}
}

func TestResolveHost_Localhost(t *testing.T) {
	result := ResolveHost(t, "localhost")
	assert.Equal(t, "localhost", result)
}

func TestResolveHost_127_0_0_1(t *testing.T) {
	result := ResolveHost(t, "127.0.0.1")
	assert.Equal(t, "127.0.0.1", result)
}

func TestResolveHost_OtherHost(t *testing.T) {
	result := ResolveHost(t, "some-host")
	assert.Equal(t, "some-host", result)
}

func TestRetryStart_SuccessFirstAttempt(t *testing.T) {
	calls := 0
	res, err := retryStart(t, 3, "test", func() (string, error) {
		calls++
		return "ok", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", res)
	assert.Equal(t, 1, calls)
}

func TestRetryStart_SuccessAfterRetries(t *testing.T) {
	calls := 0
	res, err := retryStart(t, 3, "test", func() (string, error) {
		calls++
		if calls < 3 {
			return "", errors.New("temporary failure")
		}
		return "finally", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "finally", res)
	assert.Equal(t, 3, calls)
}

func TestRetryStart_AllAttemptsFail(t *testing.T) {
	calls := 0
	boom := errors.New("boom")
	res, err := retryStart(t, 2, "test", func() (string, error) {
		calls++
		return "", boom
	})
	require.ErrorIs(t, err, boom)
	assert.Empty(t, res)
	assert.Equal(t, 2, calls)
}

func TestRetryStart_ZeroAttempts(t *testing.T) {
	calls := 0
	res, err := retryStart(t, 0, "test", func() (string, error) {
		calls++
		return "never", nil
	})
	require.NoError(t, err)
	assert.Empty(t, res)
	assert.Equal(t, 0, calls)
}

func TestMustParsePort(t *testing.T) {
	assert.Equal(t, 5432, mustParsePort(t, "5432"))
	assert.Equal(t, 1, mustParsePort(t, "1"))
	assert.Equal(t, 65535, mustParsePort(t, "65535"))
}
