package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnectClassifierDB_NoEnv(t *testing.T) {
	t.Setenv("DB_HOST", "")
	t.Setenv("POSTGRES_USER", "")
	t.Setenv("POSTGRES_PASSWORD", "")
	t.Setenv("POSTGRES_DB", "")

	assert.Nil(t, connectClassifierDB())
}

func TestConnectClassifierDB_MissingCredentials(t *testing.T) {
	t.Setenv("DB_HOST", "127.0.0.1")
	t.Setenv("POSTGRES_USER", "")
	t.Setenv("POSTGRES_PASSWORD", "")
	t.Setenv("POSTGRES_DB", "")

	assert.Nil(t, connectClassifierDB())
}

func TestConnectClassifierDB_Unreachable(t *testing.T) {
	t.Setenv("DB_HOST", "127.0.0.1")
	t.Setenv("DB_PORT", "1")
	t.Setenv("POSTGRES_USER", "u")
	t.Setenv("POSTGRES_PASSWORD", "p")
	t.Setenv("POSTGRES_DB", "d")
	t.Setenv("DB_SSLMODE", "disable")

	// Порт 1 заведомо закрыт — Ping должен провалиться, соединение закрыто.
	assert.Nil(t, connectClassifierDB())
}
