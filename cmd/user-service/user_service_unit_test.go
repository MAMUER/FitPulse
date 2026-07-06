package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MAMUER/project/internal/config"
)

func TestContainsUpperCase(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"empty", "", false},
		{"lowercase", "hello", false},
		{"uppercase", "HELLO", true},
		{"mixed", "Hello", true},
		{"with digit", "Hello1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsUpperCase(tt.s)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestContainsLowerCase(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"empty", "", false},
		{"uppercase", "HELLO", false},
		{"lowercase", "hello", true},
		{"mixed", "Hello", true},
		{"with digit", "HELLO1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsLowerCase(tt.s)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestContainsDigit(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"empty", "", false},
		{"letters only", "hello", false},
		{"with digit", "hello1", true},
		{"all digits", "12345", true},
		{"mixed", "a1b2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsDigit(tt.s)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractLocalPart(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{"simple email", "user@example.com", "user"},
		{"with dots", "john.doe@example.com", "john.doe"},
		{"multiple @", "user@sub@domain.com", "user"},
		{"no @", "user", "user"},
		{"empty", "", ""},
		{"trailing @", "user@", "user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractLocalPart(tt.email)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
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
			if tt.envValue != "" || tt.envKey == "UNSET_KEY" {
				if tt.envValue != "" {
					t.Setenv(tt.envKey, tt.envValue)
				}
			}
			got := config.GetEnv(tt.envKey, tt.fallback)
			assert.Equal(t, tt.want, got)
		})
	}
}
