package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "postgres",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, "5432", cfg.Port)
	assert.Equal(t, "postgres", cfg.User)
	assert.Equal(t, "postgres", cfg.Password)
	assert.Equal(t, "testdb", cfg.DBName)
	assert.Equal(t, "disable", cfg.SSLMode)
}

func TestConfigConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "default config",
			config: Config{
				Host:     "localhost",
				Port:     "5432",
				User:     "postgres",
				Password: "postgres",
				DBName:   "testdb",
				SSLMode:  "disable",
			},
			expected: "host=localhost port=5432 user=postgres password=postgres dbname=testdb sslmode=disable",
		},
		{
			name: "custom host and port",
			config: Config{
				Host:     "db.example.com",
				Port:     "5433",
				User:     "appuser",
				Password: "secret",
				DBName:   "production",
				SSLMode:  "require",
			},
			expected: "host=db.example.com port=5433 user=appuser password=secret dbname=production sslmode=require",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.config.ConnectionString()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestNewConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "postgres",
		DBName:   "postgres",
		SSLMode:  "disable",
	}

	db, err := NewConnection(cfg)
	if err != nil {
		t.Skip("PostgreSQL not available, skipping test")
	}
	defer func() { _ = db.Close() }()

	assert.NotNil(t, db)
	err = db.PingContext(context.Background())
	assert.NoError(t, err)
}

func TestNewConnectionWithInvalidConfig(t *testing.T) {
	cfg := Config{
		Host:     "nonexistent-host",
		Port:     "5432",
		User:     "invalid",
		Password: "invalid",
		DBName:   "invalid",
		SSLMode:  "disable",
	}

	db, err := NewConnection(cfg)
	assert.Error(t, err)
	assert.Nil(t, db)
}

func TestConnectionPoolSettings(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "postgres",
		DBName:   "postgres",
		SSLMode:  "disable",
	}

	db, err := NewConnection(cfg)
	if err != nil {
		t.Skip("PostgreSQL not available, skipping test")
	}
	defer func() { _ = db.Close() }()

	assert.Equal(t, 25, db.Stats().MaxOpenConnections)
	assert.Equal(t, 10, db.Stats().MaxIdleClosed)
}

func TestBlindIndex(t *testing.T) {
	assert.Equal(t, "54657374", BlindIndex("Test"))
	assert.Equal(t, "54657374", BlindIndex("Test"))
	assert.NotEqual(t, BlindIndex("Test"), BlindIndex("test"))
}

func TestNicknameHash(t *testing.T) {
	assert.Equal(t, NicknameHash("alice"), NicknameHash("alice"))
	assert.NotEqual(t, NicknameHash("alice"), NicknameHash("bob"))
}

func TestGenerateNonce(t *testing.T) {
	nonce, err := GenerateNonce()
	assert.NoError(t, err)
	assert.Len(t, nonce, 12)

	nonce2, err := GenerateNonce()
	assert.NoError(t, err)
	assert.NotEqual(t, nonce, nonce2)
}

func TestPgsodiumDecryptDualParam(t *testing.T) {
	SetPgsodiumKeyID(1)
	sql := PgsodiumDecryptDualParam("u.full_name_encrypted", "u.full_name_nonce", "full_name")
	assert.Contains(t, sql, "CASE WHEN length(u.full_name_nonce) >= 12")
	assert.Contains(t, sql, "pgsodium.crypto_aead_aegis256_decrypt")
	assert.Contains(t, sql, "pgsodium.crypto_aead_det_decrypt")
}
