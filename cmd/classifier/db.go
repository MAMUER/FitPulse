package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// Registers the "pgx" database/sql driver used by sql.Open below.
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/MAMUER/project/internal/config"
)

// connectClassifierDB открывает подключение к PostgreSQL для хранения
// персональных дельт. Возвращает nil, если БД не настроена — тогда
// используется in-memory store (дельты теряются при рестарте).
func connectClassifierDB() *sql.DB {
	host := config.GetEnv("DB_HOST", "")
	if host == "" {
		return nil
	}
	user := config.GetEnv("POSTGRES_USER", "")
	pass := config.GetEnv("POSTGRES_PASSWORD", "")
	name := config.GetEnv("POSTGRES_DB", "")
	if user == "" || pass == "" || name == "" {
		return nil
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host,
		config.GetEnv("DB_PORT", "5432"),
		user,
		pass,
		name,
		config.GetEnv("DB_SSLMODE", "disable"),
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil
	}
	return db
}
