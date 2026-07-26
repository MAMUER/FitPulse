package testcontainers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePort(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"standard port", "5432", 5432},
		{"small port", "1", 1},
		{"large port", "65535", 65535},
		{"invalid port", "not-a-port", 0},
		{"empty string", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePort(tt.input)
			assert.Equal(t, tt.expected, result)
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
