// Package config provides configuration utilities and typed config structs.
package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/logger"
)

const redacted = "[REDACTED]"

var globalViper *viper.Viper

func InitViper(appName string) {
	globalViper = MustLoadViper(appName)
}

func GetViper() *viper.Viper {
	if globalViper == nil {
		return MustLoadViper("app")
	}
	return globalViper
}

type CacheConfig struct {
	Addr     string
	Password string
	DB       int
}

func (c CacheConfig) Validate() error {
	if c.Addr == "" {
		return errors.New("cache addr is required")
	}
	return nil
}

type JWTConfig struct {
	PrivateKeyPEM string
	PublicKeyPEM  string
}

func (c JWTConfig) Validate() error {
	if c.PrivateKeyPEM == "" {
		return errors.New("JWT private key is required")
	}
	if c.PublicKeyPEM == "" {
		return errors.New("JWT public key is required")
	}
	return nil
}

type ServerConfig struct {
	Addr string
}

func (c ServerConfig) Validate() error {
	if c.Addr == "" {
		return errors.New("server addr is required")
	}
	return nil
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func (c DatabaseConfig) Validate() error {
	if c.Host == "" {
		return errors.New("db host is required")
	}
	if c.Port == "" {
		return errors.New("db port is required")
	}
	if c.User == "" {
		return errors.New("db user is required")
	}
	if c.Password == "" {
		return errors.New("db password is required")
	}
	if c.DBName == "" {
		return errors.New("db name is required")
	}
	return nil
}

type Config struct {
	App      string
	Server   ServerConfig
	Database DatabaseConfig
	Cache    CacheConfig
	JWT      JWTConfig
}

func (c Config) Validate() error {
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database: %w", err)
	}
	if err := c.Cache.Validate(); err != nil {
		return fmt.Errorf("cache: %w", err)
	}
	if err := c.JWT.Validate(); err != nil {
		return fmt.Errorf("jwt: %w", err)
	}
	return nil
}

func LoadConfig(v *viper.Viper) Config {
	return Config{
		App:    GetViperString(v, "app.name", "fitpulse"),
		Server: ServerConfig{Addr: GetViperString(v, "server.addr", ":8080")},
		Database: DatabaseConfig{
			Host:     GetViperString(v, "database.host", "localhost"),
			Port:     GetViperString(v, "database.port", "5432"),
			User:     GetViperString(v, "database.user", "postgres"),
			Password: GetViperString(v, "database.password", ""),
			DBName:   GetViperString(v, "database.dbname", "postgres"),
			SSLMode:  GetViperString(v, "database.sslmode", "disable"),
		},
		Cache: CacheConfig{
			Addr:     GetViperString(v, "cache.addr", "localhost:6379"),
			Password: GetViperString(v, "cache.password", ""),
			DB:       GetViperInt(v, "cache.db", 0),
		},
		JWT: JWTConfig{
			PrivateKeyPEM: GetViperString(v, "jwt.private_key_pem", ""),
			PublicKeyPEM:  GetViperString(v, "jwt.public_key_pem", ""),
		},
	}
}

func MustLoadConfig(v *viper.Viper) Config {
	cfg := LoadConfig(v)
	if err := cfg.Validate(); err != nil {
		panic(fmt.Sprintf("invalid configuration: %v", err))
	}
	return cfg
}

func LoadCacheConfig() CacheConfig {
	return CacheConfig{
		Addr:     GetEnv("VALKEY_ADDR", "localhost:6379"),
		Password: GetEnv("VALKEY_PASSWORD"),
		DB:       GetEnvInt("VALKEY_DB", 0),
	}
}

func LoadJWTConfig() JWTConfig {
	return JWTConfig{
		PrivateKeyPEM: GetEnvRequired("JWT_PRIVATE_KEY_PEM"),
		PublicKeyPEM:  GetEnvRequired("JWT_PUBLIC_KEY_PEM"),
	}
}

func LoadServerConfig(envVar, defaultAddr string) ServerConfig {
	return ServerConfig{
		Addr: GetEnv(envVar, defaultAddr),
	}
}

func LogConfig(log *logger.Logger, cfg interface{}) {
	log.Info("configuration loaded", zap.Any("config", redactSecrets(cfg)))
}

func redactSecrets(cfg interface{}) interface{} {
	switch c := cfg.(type) {
	case CacheConfig:
		return struct {
			Addr     string
			Password string
			DB       int
		}{
			Addr:     c.Addr,
			Password: redacted,
			DB:       c.DB,
		}
	case JWTConfig:
		return struct {
			PrivateKeyPEM string
			PublicKeyPEM  string
		}{
			PrivateKeyPEM: redacted,
			PublicKeyPEM:  redacted,
		}
	case DatabaseConfig:
		return struct {
			Host     string
			Port     string
			User     string
			Password string
			DBName   string
			SSLMode  string
		}{
			Host:     c.Host,
			Port:     c.Port,
			User:     c.User,
			Password: redacted,
			DBName:   c.DBName,
			SSLMode:  c.SSLMode,
		}
	case Config:
		return struct {
			App      string
			Server   ServerConfig
			Database interface{}
			Cache    interface{}
			JWT      interface{}
		}{
			App:      c.App,
			Server:   c.Server,
			Database: redactSecrets(c.Database),
			Cache:    redactSecrets(c.Cache),
			JWT:      redactSecrets(c.JWT),
		}
	default:
		return cfg
	}
}

func GetViperString(v *viper.Viper, key, defaultVal string) string {
	if !v.IsSet(key) {
		return defaultVal
	}
	return v.GetString(key)
}

func GetViperInt(v *viper.Viper, key string, defaultVal int) int {
	if !v.IsSet(key) {
		return defaultVal
	}
	return v.GetInt(key)
}

func GetViperBool(v *viper.Viper, key string, defaultVal bool) bool {
	if !v.IsSet(key) {
		return defaultVal
	}
	return v.GetBool(key)
}

func GetViperDuration(v *viper.Viper, key, defaultVal string) time.Duration {
	val := GetViperString(v, key, defaultVal)
	d, err := time.ParseDuration(val)
	if err != nil {
		return time.Duration(0)
	}
	return d
}

func GetViperFloat64(v *viper.Viper, key string, defaultVal float64) float64 {
	if !v.IsSet(key) {
		return defaultVal
	}
	return v.GetFloat64(key)
}
