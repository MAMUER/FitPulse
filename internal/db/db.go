// Package db provides database connection utilities.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	// Register PostgreSQL driver for database/sql.
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

const (
	databasePingAttempts = 10
	databasePingInterval = time.Second
	databasePingTimeout  = 15 * time.Second
)

// NewConnection opens a new PostgreSQL connection using database/sql and reports pool usage metrics.
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

	pingCtx, cancel := context.WithTimeout(context.Background(), databasePingTimeout)
	defer cancel()
	if err := pingWithRetry(pingCtx, db.PingContext, databasePingAttempts, databasePingInterval); err != nil {
		_ = db.Close()
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

func pingWithRetry(ctx context.Context, ping func(context.Context) error, attempts int, interval time.Duration) error {
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := ping(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if attempt == attempts {
			break
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("database ping interrupted after %d attempts: %w", attempt, errors.Join(ctx.Err(), lastErr))
		case <-timer.C:
		}
	}

	return fmt.Errorf("database ping failed after %d attempts: %w", attempts, lastErr)
}

// PgxConfig returns a pgx connection config from the standard Config.
func (c Config) PgxConfig() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode)
}

// NewPgxPool opens a new pgx connection pool and reports pool usage metrics.
func NewPgxPool(cfg Config) (*pgxpool.Pool, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid db config: %w", err)
	}

	connStr := cfg.PgxConfig()

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	poolMetricOnce.Do(func() {
		go func(dbName string) {
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if pool == nil {
					continue
				}
				stats := pool.Stat()
				usage := float64(stats.AcquireCount()) / float64(max(int64(stats.MaxConns()), 1))
				metrics.DBConnectionPoolUsage.WithLabelValues(dbName, "main").Set(usage)
			}
		}(cfg.DBName)
	})

	return pool, nil
}
