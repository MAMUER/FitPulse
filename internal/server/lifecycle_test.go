package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	t.Run("returns config with provided ports", func(t *testing.T) {
		cfg := DefaultConfig("8080", "9090")
		assert.Equal(t, "8080", cfg.Port)
		assert.Equal(t, "9090", cfg.MetricsPort)
	})

	t.Run("returns default timeouts", func(t *testing.T) {
		cfg := DefaultConfig("8080", "9090")
		assert.Equal(t, 15*time.Second, cfg.ReadTimeout)
		assert.Equal(t, 30*time.Second, cfg.WriteTimeout)
		assert.Equal(t, 60*time.Second, cfg.IdleTimeout)
		assert.Equal(t, 10*time.Second, cfg.ShutdownTimeout)
	})
}

func TestConfigValidation(t *testing.T) {
	t.Run("empty ports are accepted", func(t *testing.T) {
		cfg := DefaultConfig("", "")
		assert.Equal(t, "", cfg.Port)
		assert.Equal(t, "", cfg.MetricsPort)
	})

	t.Run("custom timeouts are preserved", func(t *testing.T) {
		cfg := Config{
			Port:            "8080",
			MetricsPort:     "9090",
			ReadTimeout:     5 * time.Second,
			WriteTimeout:    10 * time.Second,
			IdleTimeout:     15 * time.Second,
			ShutdownTimeout: 3 * time.Second,
		}
		assert.Equal(t, 5*time.Second, cfg.ReadTimeout)
		assert.Equal(t, 10*time.Second, cfg.WriteTimeout)
		assert.Equal(t, 15*time.Second, cfg.IdleTimeout)
		assert.Equal(t, 3*time.Second, cfg.ShutdownTimeout)
	})
}
