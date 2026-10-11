package testcontainers

import (
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
