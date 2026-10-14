package main

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntToInt32(t *testing.T) {
	assert.Equal(t, int32(42), intToInt32(42))
	assert.Equal(t, int32(0), intToInt32(0))
	assert.Equal(t, int32(math.MaxInt32), intToInt32(math.MaxInt32))
	assert.Equal(t, int32(math.MinInt32), intToInt32(math.MinInt32))
	// За пределами диапазона int32 возвращается 0.
	assert.Equal(t, int32(0), intToInt32(math.MaxInt32+1))
	assert.Equal(t, int32(0), intToInt32(math.MinInt32-1))
}

func TestPointerConversions(t *testing.T) {
	s := "hello"
	assert.Equal(t, "hello", toString(&s))
	assert.Equal(t, "", toString(nil))

	i := int32(7)
	assert.Equal(t, int32(7), toInt32(&i))
	assert.Equal(t, int32(0), toInt32(nil))

	f := 3.5
	assert.Equal(t, 3.5, toFloat64(&f))
	assert.Equal(t, float64(0), toFloat64(nil))

	f32 := float32(1.5)
	assert.Equal(t, float32(1.5), toFloat32(&f32))
	assert.Equal(t, float32(0), toFloat32(nil))

	got := float64Ptr(9.25)
	require.NotNil(t, got)
	assert.Equal(t, 9.25, *got)
}

func TestHashAndVerifyPasswordArgon2id(t *testing.T) {
	hashed, err := hashPasswordArgon2id("correct horse battery staple")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(hashed, "$argon2id$v=19$"))

	assert.True(t, verifyPasswordArgon2id(hashed, "correct horse battery staple"))
	assert.False(t, verifyPasswordArgon2id(hashed, "wrong password"))
}

func TestVerifyPasswordArgon2idInvalidInput(t *testing.T) {
	// Неправильная структура.
	assert.False(t, verifyPasswordArgon2id("not-a-hash", "x"))
	// Не base64.
	assert.False(t, verifyPasswordArgon2id("$argon2id$v=19$m=65536,t=3,p=1$!!!notbase64!!!$alsobad", "x"))
	// Битый base64 в соли.
	assert.False(t, verifyPasswordArgon2id("$argon2id$v=19$m=65536,t=3,p=1$@@@@$AAAA", "x"))
}

func TestBuildDBConfigFromEnv(t *testing.T) {
	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("DB_PORT", "6543")
	t.Setenv("POSTGRES_USER", "fitpulse_app")
	t.Setenv("POSTGRES_PASSWORD", "s3cret")
	t.Setenv("POSTGRES_DB", "fitness_platform")
	t.Setenv("DB_SSLMODE", "require")

	cfg := buildDBConfig()
	assert.Equal(t, "db.internal", cfg.Host)
	assert.Equal(t, "6543", cfg.Port)
	assert.Equal(t, "fitpulse_app", cfg.User)
	assert.Equal(t, "s3cret", cfg.Password)
	assert.Equal(t, "fitness_platform", cfg.DBName)
	assert.Equal(t, "require", cfg.SSLMode)
}

func TestBuildDBConfigDefaults(t *testing.T) {
	cfg := buildDBConfig()
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, "5432", cfg.Port)
	assert.Equal(t, "disable", cfg.SSLMode)
	assert.Equal(t, "fitness", cfg.DBName)
}
