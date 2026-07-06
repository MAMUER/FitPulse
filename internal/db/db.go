// Package db provides database connection utilities.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"sync"
	"time"

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

var poolMetricOnce sync.Once

// NewConnection opens a new PostgreSQL connection and reports pool usage metrics.
func NewConnection(cfg Config) (*sql.DB, error) {
	if cfg.User == "" {
		cfg.User = "postgres"
	}
	if cfg.Password == "" {
		return nil, errors.New("POSTGRES_PASSWORD environment variable is required")
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
