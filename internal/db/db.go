// Package db provides database connection utilities.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	_ "github.com/lib/pq"

	"github.com/MAMUER/project/internal/config"
	"github.com/MAMUER/project/internal/metrics"
)

// Config holds database connection settings.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// ConnectionString returns a PostgreSQL connection string.
func (c Config) ConnectionString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

// Validate checks that the database configuration is valid.
func (c Config) Validate() error {
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

// LoadConfig loads database configuration from environment variables.
// It supports the _FILE suffix for Docker/Kubernetes secrets.
func LoadConfig() Config {
	return Config{
		Host:     config.GetEnv("DB_HOST", "localhost"),
		Port:     config.GetEnv("DB_PORT", "5432"),
		User:     config.GetEnv("DB_USER", "postgres"),
		Password: config.GetEnvRequired("DB_PASSWORD"),
		DBName:   config.GetEnv("DB_NAME", "postgres"),
		SSLMode:  config.GetEnv("DB_SSL_MODE", "disable"),
	}
}

var poolMetricOnce sync.Once

// NewConnection opens a new PostgreSQL connection and reports pool usage metrics.
func NewConnection(cfg Config) (*sql.DB, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid db config: %w", err)
	}

	connStr := cfg.ConnectionString()

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	poolMetricOnce.Do(func() {
		go func(dbName string) {
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if db == nil {
					continue
				}
				stats := db.Stats()
				usage := float64(stats.InUse) / float64(max(stats.MaxOpenConnections, 1))
				metrics.DBConnectionPoolUsage.WithLabelValues(dbName, "main").Set(usage)
			}
		}(cfg.DBName)
	})

	return db, nil
}
