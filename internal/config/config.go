// Package config provides configuration utilities and typed config structs.
package config

import (
	"errors"

	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/logger"
)

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
			Password: "[REDACTED]",
			DB:       c.DB,
		}
	case JWTConfig:
		return struct {
			PrivateKeyPEM string
			PublicKeyPEM  string
		}{
			PrivateKeyPEM: "[REDACTED]",
			PublicKeyPEM:  "[REDACTED]",
		}
	default:
		return cfg
	}
}
