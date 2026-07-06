package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MAMUER/project/internal/config"
)

func TestGetEnv(t *testing.T) {
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
			got := config.GetEnv(tt.envKey, tt.fallback)
			assert.Equal(t, tt.want, got)
		})
	}
}
