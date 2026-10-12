package config

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/logger"
)

func TestCacheConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    CacheConfig
		wantError bool
	}{
		{
			name:      "valid config",
			config:    CacheConfig{Addr: "localhost:6379", Password: "secret", DB: 0},
			wantError: false,
		},
		{
			name:      "empty addr",
			config:    CacheConfig{Addr: "", Password: "secret", DB: 0},
			wantError: true,
		},
		{
			name:      "empty password is valid",
			config:    CacheConfig{Addr: "localhost:6379", Password: "", DB: 0},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestJWTConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    JWTConfig
		wantError bool
	}{
		{
			name:      "valid config",
			config:    JWTConfig{PrivateKeyPEM: "private", PublicKeyPEM: "public"},
			wantError: false,
		},
		{
			name:      "empty private key",
			config:    JWTConfig{PrivateKeyPEM: "", PublicKeyPEM: "public"},
			wantError: true,
		},
		{
			name:      "empty public key",
			config:    JWTConfig{PrivateKeyPEM: "private", PublicKeyPEM: ""},
			wantError: true,
		},
		{
			name:      "both empty",
			config:    JWTConfig{PrivateKeyPEM: "", PublicKeyPEM: ""},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    ServerConfig
		wantError bool
	}{
		{
			name:      "valid config",
			config:    ServerConfig{Addr: ":8080"},
			wantError: false,
		},
		{
			name:      "empty addr",
			config:    ServerConfig{Addr: ""},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoadCacheConfig(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		require.NoError(t, os.Unsetenv("VALKEY_ADDR"))
		require.NoError(t, os.Unsetenv("VALKEY_PASSWORD"))
		require.NoError(t, os.Unsetenv("VALKEY_DB"))

		cfg := LoadCacheConfig()
		assert.Equal(t, "localhost:6379", cfg.Addr)
		assert.Empty(t, cfg.Password)
		assert.Equal(t, 0, cfg.DB)
	})

	t.Run("custom values", func(t *testing.T) {
		require.NoError(t, os.Setenv("VALKEY_ADDR", "valkey:6379"))
		require.NoError(t, os.Setenv("VALKEY_PASSWORD", "secret"))
		require.NoError(t, os.Setenv("VALKEY_DB", "1"))
		t.Cleanup(func() {
			require.NoError(t, os.Unsetenv("VALKEY_ADDR"))
			require.NoError(t, os.Unsetenv("VALKEY_PASSWORD"))
			require.NoError(t, os.Unsetenv("VALKEY_DB"))
		})

		cfg := LoadCacheConfig()
		assert.Equal(t, "valkey:6379", cfg.Addr)
		assert.Equal(t, "secret", cfg.Password)
		assert.Equal(t, 1, cfg.DB)
	})
}

func TestLoadJWTConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		require.NoError(t, os.Setenv("JWT_PRIVATE_KEY_PEM", "private-key"))
		require.NoError(t, os.Setenv("JWT_PUBLIC_KEY_PEM", "public-key"))
		t.Cleanup(func() {
			require.NoError(t, os.Unsetenv("JWT_PRIVATE_KEY_PEM"))
			require.NoError(t, os.Unsetenv("JWT_PUBLIC_KEY_PEM"))
		})

		cfg := LoadJWTConfig()
		assert.Equal(t, "private-key", cfg.PrivateKeyPEM)
		assert.Equal(t, "public-key", cfg.PublicKeyPEM)
	})

	t.Run("panics on missing private key", func(t *testing.T) {
		require.NoError(t, os.Unsetenv("JWT_PRIVATE_KEY_PEM"))
		require.NoError(t, os.Unsetenv("JWT_PUBLIC_KEY_PEM"))
		assert.Panics(t, func() {
			LoadJWTConfig()
		})
	})
}

func TestLoadServerConfig(t *testing.T) {
	t.Run("default value", func(t *testing.T) {
		require.NoError(t, os.Unsetenv("TEST_SERVER_ADDR"))
		cfg := LoadServerConfig("TEST_SERVER_ADDR", ":8080")
		assert.Equal(t, ":8080", cfg.Addr)
	})

	t.Run("custom value", func(t *testing.T) {
		require.NoError(t, os.Setenv("TEST_SERVER_ADDR", ":9090"))
		t.Cleanup(func() { require.NoError(t, os.Unsetenv("TEST_SERVER_ADDR")) })

		cfg := LoadServerConfig("TEST_SERVER_ADDR", ":8080")
		assert.Equal(t, ":9090", cfg.Addr)
	})
}

func TestRedactSecrets(t *testing.T) {
	t.Run("redacts cache password", func(t *testing.T) {
		cfg := CacheConfig{Addr: "localhost:6379", Password: "secret", DB: 0}
		redacted := redactSecrets(cfg)
		cacheCfg, ok := redacted.(struct {
			Addr     string
			Password string
			DB       int
		})
		require.True(t, ok)
		assert.Equal(t, "localhost:6379", cacheCfg.Addr)
		assert.Equal(t, "[REDACTED]", cacheCfg.Password)
		assert.Equal(t, 0, cacheCfg.DB)
	})

	t.Run("redacts JWT keys", func(t *testing.T) {
		cfg := JWTConfig{PrivateKeyPEM: "private", PublicKeyPEM: "public"}
		redacted := redactSecrets(cfg)
		jwtCfg, ok := redacted.(struct {
			PrivateKeyPEM string
			PublicKeyPEM  string
		})
		require.True(t, ok)
		assert.Equal(t, "[REDACTED]", jwtCfg.PrivateKeyPEM)
		assert.Equal(t, "[REDACTED]", jwtCfg.PublicKeyPEM)
	})

	t.Run("returns config as-is for unknown type", func(t *testing.T) {
		type CustomConfig struct {
			Value string
		}
		cfg := CustomConfig{Value: "test"}
		redacted := redactSecrets(cfg)
		assert.Equal(t, cfg, redacted)
	})
}

func TestDatabaseConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    DatabaseConfig
		wantError bool
	}{
		{
			name:      "valid config",
			config:    DatabaseConfig{Host: "localhost", Port: "5432", User: "postgres", Password: "secret", DBName: "mydb", SSLMode: "disable"},
			wantError: false,
		},
		{
			name:      "empty host",
			config:    DatabaseConfig{Port: "5432", User: "postgres", Password: "secret", DBName: "mydb"},
			wantError: true,
		},
		{
			name:      "empty password",
			config:    DatabaseConfig{Host: "localhost", Port: "5432", User: "postgres", DBName: "mydb"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := Config{
			App:    "test",
			Server: ServerConfig{Addr: ":8080"},
			Database: DatabaseConfig{
				Host:     "localhost",
				Port:     "5432",
				User:     "postgres",
				Password: "secret",
				DBName:   "mydb",
				SSLMode:  "disable",
			},
			Cache: CacheConfig{Addr: "localhost:6379", DB: 0},
			JWT:   JWTConfig{PrivateKeyPEM: "private", PublicKeyPEM: "public"},
		}
		require.NoError(t, cfg.Validate())
	})

	t.Run("invalid server", func(t *testing.T) {
		cfg := Config{Server: ServerConfig{Addr: ""}}
		require.Error(t, cfg.Validate())
	})

	t.Run("invalid database", func(t *testing.T) {
		cfg := Config{Database: DatabaseConfig{}}
		require.Error(t, cfg.Validate())
	})
}

func TestLoadConfigFromViper(t *testing.T) {
	v := viper.New()
	v.Set("app.name", "testapp")
	v.Set("server.addr", ":9090")
	v.Set("database.host", "db.example.com")
	v.Set("database.port", "5433")
	v.Set("database.user", "admin")
	v.Set("database.password", "s3cret")
	v.Set("database.dbname", "appdb")
	v.Set("database.sslmode", "require")
	v.Set("cache.addr", "redis:6379")
	v.Set("cache.password", "redispass")
	v.Set("cache.db", 2)
	v.Set("jwt.private_key_pem", "priv")
	v.Set("jwt.public_key_pem", "pub")

	cfg := LoadConfig(v)
	assert.Equal(t, "testapp", cfg.App)
	assert.Equal(t, ":9090", cfg.Server.Addr)
	assert.Equal(t, "db.example.com", cfg.Database.Host)
	assert.Equal(t, "redis:6379", cfg.Cache.Addr)
	assert.Equal(t, 2, cfg.Cache.DB)
}

func TestViperHelpers(t *testing.T) {
	v := viper.New()
	v.Set("existing_str", "hello")
	v.Set("existing_int", 42)
	v.Set("existing_bool", true)
	v.Set("existing_duration", "5s")
	v.Set("existing_float", 3.14)

	assert.Equal(t, "hello", GetViperString(v, "existing_str", "default"))
	assert.Equal(t, "default", GetViperString(v, "missing_str", "default"))
	assert.Equal(t, 42, GetViperInt(v, "existing_int", 0))
	assert.Equal(t, 0, GetViperInt(v, "missing_int", 0))
	assert.Equal(t, true, GetViperBool(v, "existing_bool", false))
	assert.Equal(t, false, GetViperBool(v, "missing_bool", false))
	assert.Equal(t, 5*time.Second, GetViperDuration(v, "existing_duration", "1s"))
	assert.Equal(t, 1*time.Second, GetViperDuration(v, "missing_duration", "1s"))
	assert.Equal(t, 3.14, GetViperFloat64(v, "existing_float", 0.0))
	assert.Equal(t, 0.0, GetViperFloat64(v, "missing_float", 0.0))
}

func TestGetViperDurationInvalidValue(t *testing.T) {
	v := viper.New()
	v.Set("invalid_duration", "not-a-duration")

	assert.Equal(t, time.Duration(0), GetViperDuration(v, "invalid_duration", "1s"))
}

func TestDatabaseConfigValidateMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(cfg *DatabaseConfig)
		wantErr string
	}{
		{
			name:    "missing host",
			mutate:  func(cfg *DatabaseConfig) { cfg.Host = "" },
			wantErr: "db host is required",
		},
		{
			name:    "missing port",
			mutate:  func(cfg *DatabaseConfig) { cfg.Port = "" },
			wantErr: "db port is required",
		},
		{
			name:    "missing user",
			mutate:  func(cfg *DatabaseConfig) { cfg.User = "" },
			wantErr: "db user is required",
		},
		{
			name:    "missing password",
			mutate:  func(cfg *DatabaseConfig) { cfg.Password = "" },
			wantErr: "db password is required",
		},
		{
			name:    "missing dbname",
			mutate:  func(cfg *DatabaseConfig) { cfg.DBName = "" },
			wantErr: "db name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DatabaseConfig{
				Host:     "localhost",
				Port:     "5432",
				User:     "postgres",
				Password: "secret",
				DBName:   "mydb",
				SSLMode:  "disable",
			}
			tt.mutate(&cfg)

			err := cfg.Validate()
			require.Error(t, err)
			assert.Equal(t, tt.wantErr, err.Error())
		})
	}
}

func TestConfigValidateNestedErrors(t *testing.T) {
	validConfig := func() Config {
		return Config{
			App:    "fitpulse",
			Server: ServerConfig{Addr: ":8080"},
			Database: DatabaseConfig{
				Host:     "localhost",
				Port:     "5432",
				User:     "postgres",
				Password: "secret",
				DBName:   "mydb",
				SSLMode:  "disable",
			},
			Cache: CacheConfig{Addr: "localhost:6379", DB: 0},
			JWT:   JWTConfig{PrivateKeyPEM: "private", PublicKeyPEM: "public"},
		}
	}

	t.Run("invalid cache", func(t *testing.T) {
		cfg := validConfig()
		cfg.Cache.Addr = ""

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cache: cache addr is required")
	})

	t.Run("invalid jwt", func(t *testing.T) {
		cfg := validConfig()
		cfg.JWT.PrivateKeyPEM = ""

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "jwt: JWT private key is required")
	})

	t.Run("invalid database", func(t *testing.T) {
		cfg := validConfig()
		cfg.Database.DBName = ""

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database: db name is required")
	})

	t.Run("valid config", func(t *testing.T) {
		require.NoError(t, validConfig().Validate())
	})
}

func TestRedactSecretsDatabaseConfig(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "supersecret",
		DBName:   "mydb",
		SSLMode:  "disable",
	}

	redacted := redactSecrets(cfg)
	dbCfg, ok := redacted.(struct {
		Host     string
		Port     string
		User     string
		Password string
		DBName   string
		SSLMode  string
	})
	require.True(t, ok)
	assert.Equal(t, "localhost", dbCfg.Host)
	assert.Equal(t, "5432", dbCfg.Port)
	assert.Equal(t, "postgres", dbCfg.User)
	assert.Equal(t, "mydb", dbCfg.DBName)
	assert.Equal(t, "disable", dbCfg.SSLMode)
	assert.Equal(t, "[REDACTED]", dbCfg.Password)
}

func TestRedactSecretsFullConfig(t *testing.T) {
	cfg := Config{
		App:    "fitpulse",
		Server: ServerConfig{Addr: ":8080"},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "postgres",
			Password: "supersecret",
			DBName:   "mydb",
			SSLMode:  "disable",
		},
		Cache: CacheConfig{Addr: "localhost:6379", Password: "redissecret", DB: 1},
		JWT:   JWTConfig{PrivateKeyPEM: "private", PublicKeyPEM: "public"},
	}

	redacted := redactSecrets(cfg)
	redactedCfg, ok := redacted.(struct {
		App      string
		Server   ServerConfig
		Database interface{}
		Cache    interface{}
		JWT      interface{}
	})
	require.True(t, ok)
	assert.Equal(t, "fitpulse", redactedCfg.App)
	assert.Equal(t, ":8080", redactedCfg.Server.Addr)

	database, ok := redactedCfg.Database.(struct {
		Host     string
		Port     string
		User     string
		Password string
		DBName   string
		SSLMode  string
	})
	require.True(t, ok)
	assert.Equal(t, "[REDACTED]", database.Password)

	cache, ok := redactedCfg.Cache.(struct {
		Addr     string
		Password string
		DB       int
	})
	require.True(t, ok)
	assert.Equal(t, "localhost:6379", cache.Addr)
	assert.Equal(t, "[REDACTED]", cache.Password)
	assert.Equal(t, 1, cache.DB)

	jwt, ok := redactedCfg.JWT.(struct {
		PrivateKeyPEM string
		PublicKeyPEM  string
	})
	require.True(t, ok)
	assert.Equal(t, "[REDACTED]", jwt.PrivateKeyPEM)
	assert.Equal(t, "[REDACTED]", jwt.PublicKeyPEM)
}

func TestLoadConfigDefaultsFromEmptyViper(t *testing.T) {
	cfg := LoadConfig(viper.New())

	assert.Equal(t, "fitpulse", cfg.App)
	assert.Equal(t, ":8080", cfg.Server.Addr)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, "5432", cfg.Database.Port)
	assert.Equal(t, "postgres", cfg.Database.User)
	assert.Empty(t, cfg.Database.Password)
	assert.Equal(t, "postgres", cfg.Database.DBName)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.Equal(t, "localhost:6379", cfg.Cache.Addr)
	assert.Empty(t, cfg.Cache.Password)
	assert.Equal(t, 0, cfg.Cache.DB)
	assert.Empty(t, cfg.JWT.PrivateKeyPEM)
	assert.Empty(t, cfg.JWT.PublicKeyPEM)
}

func TestMustLoadConfig(t *testing.T) {
	t.Run("returns valid config", func(t *testing.T) {
		v := viper.New()
		v.Set("server.addr", ":8080")
		v.Set("database.host", "localhost")
		v.Set("database.port", "5432")
		v.Set("database.user", "postgres")
		v.Set("database.password", "secret")
		v.Set("database.dbname", "mydb")
		v.Set("cache.addr", "localhost:6379")
		v.Set("jwt.private_key_pem", "private")
		v.Set("jwt.public_key_pem", "public")

		cfg := MustLoadConfig(v)
		require.NoError(t, cfg.Validate())
		assert.Equal(t, ":8080", cfg.Server.Addr)
	})

	t.Run("panics on invalid config", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = MustLoadConfig(viper.New())
		})
	})
}

func TestGetViperWithoutInit(t *testing.T) {
	previous := globalViper
	globalViper = nil
	t.Cleanup(func() { globalViper = previous })

	v := GetViper()
	require.NotNil(t, v)
}

func TestInitViperSetsGlobalViper(t *testing.T) {
	InitViper("app")
	require.NotNil(t, globalViper)
	assert.Same(t, globalViper, GetViper())
}

func TestLogConfig(t *testing.T) {
	cfg := Config{
		App:    "fitpulse",
		Server: ServerConfig{Addr: ":8080"},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "postgres",
			Password: "supersecret",
			DBName:   "mydb",
			SSLMode:  "disable",
		},
		Cache: CacheConfig{Addr: "localhost:6379", Password: "redissecret", DB: 0},
		JWT:   JWTConfig{PrivateKeyPEM: "private", PublicKeyPEM: "public"},
	}

	t.Run("logs full config", func(t *testing.T) {
		LogConfig(logger.New("test"), cfg)
	})

	t.Run("logs cache config", func(t *testing.T) {
		LogConfig(logger.New("test"), cfg.Cache)
	})

	t.Run("logs jwt config", func(t *testing.T) {
		LogConfig(logger.New("test"), cfg.JWT)
	})

	t.Run("logs database config", func(t *testing.T) {
		LogConfig(logger.New("test"), cfg.Database)
	})

	t.Run("logs unknown config", func(t *testing.T) {
		LogConfig(logger.New("test"), "not-a-config")
	})
}
