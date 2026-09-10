package config

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadViper(t *testing.T) {
	t.Run("returns error when config file has invalid yaml", func(t *testing.T) {
		dir := t.TempDir()
		badFile := dir + string(os.PathSeparator) + "bad.yaml"
		require.NoError(t, os.WriteFile(badFile, []byte(": invalid: [yaml"), 0644))

		oldWd, errWd := os.Getwd()
		require.NoError(t, errWd)
		require.NoError(t, os.Chdir(dir))
		t.Cleanup(func() { require.NoError(t, os.Chdir(oldWd)) })

		_, err := LoadViper("bad")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "read config")
	})

	t.Run("does not return error when config file is missing but environment variables are present", func(t *testing.T) {
		require.NoError(t, os.Setenv("TESTAPP_KEY", "value"))
		t.Cleanup(func() { require.NoError(t, os.Unsetenv("TESTAPP_KEY")) })

		v, err := LoadViper("testapp")

		require.NoError(t, err)
		assert.NotNil(t, v)
	})

	t.Run("does not return error when no config and no env set", func(t *testing.T) {
		oldWd, errWd := os.Getwd()
		require.NoError(t, errWd)
		require.NoError(t, os.Chdir(t.TempDir()))
		t.Cleanup(func() { require.NoError(t, os.Chdir(oldWd)) })

		v, err := LoadViper("nonexistent_app_xyz")

		require.NoError(t, err)
		assert.NotNil(t, v)
	})

	t.Run("sets env prefix correctly", func(t *testing.T) {
		require.NoError(t, os.Setenv("VIPERAPP_HOST", "localhost"))
		t.Cleanup(func() { require.NoError(t, os.Unsetenv("VIPERAPP_HOST")) })

		v, err := LoadViper("viperapp")

		require.NoError(t, err)
		assert.NotNil(t, v)
		assert.Equal(t, "localhost", v.GetString("host"))
	})
}

func TestMustLoadViper(t *testing.T) {
	t.Run("panics when load fails", func(t *testing.T) {
		dir := t.TempDir()
		badFile := dir + string(os.PathSeparator) + "mustbad.yaml"
		require.NoError(t, os.WriteFile(badFile, []byte(": invalid: [yaml"), 0644))

		oldWd, errWd := os.Getwd()
		require.NoError(t, errWd)
		require.NoError(t, os.Chdir(dir))
		t.Cleanup(func() { require.NoError(t, os.Chdir(oldWd)) })

		assert.Panics(t, func() {
			MustLoadViper("mustbad")
		})
	})

	t.Run("returns viper instance when load succeeds", func(t *testing.T) {
		require.NoError(t, os.Setenv("MUSTLOADAPP_KEY", "value"))
		t.Cleanup(func() { require.NoError(t, os.Unsetenv("MUSTLOADAPP_KEY")) })

		v := MustLoadViper("mustloadapp")
		assert.NotNil(t, v)
	})
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		defaultVal string
		setup      func() *viper.Viper
		expected   string
	}{
		{
			name:       "returns value when set",
			key:        "app.name",
			defaultVal: "default",
			setup: func() *viper.Viper {
				v := viper.New()
				v.Set("app.name", "myapp")
				return v
			},
			expected: "myapp",
		},
		{
			name:       "returns default when missing",
			key:        "app.name",
			defaultVal: "default",
			setup: func() *viper.Viper {
				return viper.New()
			},
			expected: "default",
		},
		{
			name:       "returns empty default when missing and default is empty",
			key:        "app.name",
			defaultVal: "",
			setup: func() *viper.Viper {
				return viper.New()
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.setup()
			assert.Equal(t, tt.expected, GetString(v, tt.key, tt.defaultVal))
		})
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		defaultVal int
		setup      func() *viper.Viper
		expected   int
	}{
		{
			name:       "returns value when set",
			key:        "app.port",
			defaultVal: 8080,
			setup: func() *viper.Viper {
				v := viper.New()
				v.Set("app.port", 9090)
				return v
			},
			expected: 9090,
		},
		{
			name:       "returns default when missing",
			key:        "app.port",
			defaultVal: 8080,
			setup: func() *viper.Viper {
				return viper.New()
			},
			expected: 8080,
		},
		{
			name:       "returns zero default when missing and default is zero",
			key:        "app.port",
			defaultVal: 0,
			setup: func() *viper.Viper {
				return viper.New()
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.setup()
			assert.Equal(t, tt.expected, GetInt(v, tt.key, tt.defaultVal))
		})
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		defaultVal bool
		setup      func() *viper.Viper
		expected   bool
	}{
		{
			name:       "returns value when set",
			key:        "app.debug",
			defaultVal: false,
			setup: func() *viper.Viper {
				v := viper.New()
				v.Set("app.debug", true)
				return v
			},
			expected: true,
		},
		{
			name:       "returns default when missing",
			key:        "app.debug",
			defaultVal: true,
			setup: func() *viper.Viper {
				return viper.New()
			},
			expected: true,
		},
		{
			name:       "returns false default when missing",
			key:        "app.debug",
			defaultVal: false,
			setup: func() *viper.Viper {
				return viper.New()
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.setup()
			assert.Equal(t, tt.expected, GetBool(v, tt.key, tt.defaultVal))
		})
	}
}

func TestGetDuration(t *testing.T) {
	t.Run("returns parsed duration when set", func(t *testing.T) {
		v := viper.New()
		v.Set("app.timeout", "5s")

		result := GetDuration(v, "app.timeout", "1s")
		assert.Equal(t, 5*time.Second, result)
	})

	t.Run("returns default when missing", func(t *testing.T) {
		v := viper.New()

		result := GetDuration(v, "app.timeout", "1s")
		assert.Equal(t, 1*time.Second, result)
	})

	t.Run("returns zero duration when invalid value and default", func(t *testing.T) {
		v := viper.New()
		v.Set("app.timeout", "invalid")

		result := GetDuration(v, "app.timeout", "1s")
		assert.Equal(t, time.Duration(0), result)
	})

	t.Run("returns zero duration when invalid value and empty default", func(t *testing.T) {
		v := viper.New()
		v.Set("app.timeout", "invalid")

		result := GetDuration(v, "app.timeout", "")
		assert.Equal(t, time.Duration(0), result)
	})
}

func TestGetFloat64(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		defaultVal float64
		setup      func() *viper.Viper
		expected   float64
	}{
		{
			name:       "returns value when set",
			key:        "app.rate",
			defaultVal: 1.0,
			setup: func() *viper.Viper {
				v := viper.New()
				v.Set("app.rate", 2.5)
				return v
			},
			expected: 2.5,
		},
		{
			name:       "returns default when missing",
			key:        "app.rate",
			defaultVal: 1.0,
			setup: func() *viper.Viper {
				return viper.New()
			},
			expected: 1.0,
		},
		{
			name:       "returns zero default when missing and default is zero",
			key:        "app.rate",
			defaultVal: 0.0,
			setup: func() *viper.Viper {
				return viper.New()
			},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.setup()
			assert.Equal(t, tt.expected, GetFloat64(v, tt.key, tt.defaultVal))
		})
	}
}

func TestGetEnvViper(t *testing.T) {
	t.Run("returns env value when set", func(t *testing.T) {
		require.NoError(t, os.Setenv("CUSTOM_TEST_VAR", "env_value"))
		t.Cleanup(func() { require.NoError(t, os.Unsetenv("CUSTOM_TEST_VAR")) })

		result := GetEnvViper("CUSTOM_TEST_VAR", "default")
		assert.Equal(t, "env_value", result)
	})

	t.Run("returns default when env is not set", func(t *testing.T) {
		require.NoError(t, os.Unsetenv("CUSTOM_TEST_VAR_MISSING"))

		result := GetEnvViper("CUSTOM_TEST_VAR_MISSING", "default")
		assert.Equal(t, "default", result)
	})

	t.Run("returns default when env is empty string", func(t *testing.T) {
		require.NoError(t, os.Setenv("CUSTOM_TEST_VAR_EMPTY", ""))
		t.Cleanup(func() { require.NoError(t, os.Unsetenv("CUSTOM_TEST_VAR_EMPTY")) })

		result := GetEnvViper("CUSTOM_TEST_VAR_EMPTY", "default")
		assert.Equal(t, "default", result)
	})
}

func TestGetEnvRequiredViper(t *testing.T) {
	t.Run("returns env value when set", func(t *testing.T) {
		require.NoError(t, os.Setenv("REQUIRED_TEST_VAR", "env_value"))
		t.Cleanup(func() { require.NoError(t, os.Unsetenv("REQUIRED_TEST_VAR")) })

		result := GetEnvRequiredViper("REQUIRED_TEST_VAR")
		assert.Equal(t, "env_value", result)
	})

	t.Run("panics when env is not set", func(t *testing.T) {
		require.NoError(t, os.Unsetenv("REQUIRED_TEST_VAR_MISSING"))

		assert.Panics(t, func() {
			GetEnvRequiredViper("REQUIRED_TEST_VAR_MISSING")
		})
	})

	t.Run("panics when env is empty string", func(t *testing.T) {
		require.NoError(t, os.Setenv("REQUIRED_TEST_VAR_EMPTY", ""))
		t.Cleanup(func() { require.NoError(t, os.Unsetenv("REQUIRED_TEST_VAR_EMPTY")) })

		assert.Panics(t, func() {
			GetEnvRequiredViper("REQUIRED_TEST_VAR_EMPTY")
		})
	})
}
