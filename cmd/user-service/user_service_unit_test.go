package main

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/config"
)

func TestContainsUpperCase(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"empty", "", false},
		{"lowercase", "hello", false},
		{"uppercase", "HELLO", true},
		{"mixed", "Hello", true},
		{"with digit", "Hello1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsUpperCase(tt.s)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestContainsLowerCase(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"empty", "", false},
		{"uppercase", "HELLO", false},
		{"lowercase", "hello", true},
		{"mixed", "Hello", true},
		{"with digit", "HELLO1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsLowerCase(tt.s)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestContainsDigit(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"empty", "", false},
		{"letters only", "hello", false},
		{"with digit", "hello1", true},
		{"all digits", "12345", true},
		{"mixed", "a1b2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsDigit(tt.s)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractLocalPart(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{"simple email", "user@example.com", "user"},
		{"with dots", "john.doe@example.com", "john.doe"},
		{"multiple @", "user@sub@domain.com", "user"},
		{"no @", "user", "user"},
		{"empty", "", ""},
		{"trailing @", "user@", "user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractLocalPart(tt.email)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		fallback string
		want     string
	}{
		{"env set", "TEST_KEY", "value", "default", "value"},
		{"env empty", "TEST_KEY", "", "default", "default"},
		{"env unset", "UNSET_KEY", "", "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" || tt.envKey == "UNSET_KEY" {
				if tt.envValue != "" {
					t.Setenv(tt.envKey, tt.envValue)
				}
			}
			got := config.GetEnv(tt.envKey, tt.fallback)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHashPasswordArgon2id(t *testing.T) {
	hash, err := hashPasswordArgon2id("password123")
	assert.NoError(t, err)
	assert.Contains(t, hash, "$argon2id$")
	assert.Contains(t, hash, "m=65536,t=3,p=1")
}

func TestVerifyPasswordArgon2id(t *testing.T) {
	hash, err := hashPasswordArgon2id("password123")
	require.NoError(t, err)

	assert.True(t, verifyPasswordArgon2id(hash, "password123"))
	assert.False(t, verifyPasswordArgon2id(hash, "wrongpassword"))
	assert.False(t, verifyPasswordArgon2id("invalid", "password123"))
	assert.False(t, verifyPasswordArgon2id("$argon2id$v=19$m=65536,t=3,p=1$salt", "password123"))
}

func TestVerifyPasswordArgon2idFormat(t *testing.T) {
	hash, err := hashPasswordArgon2id("testpassword")
	require.NoError(t, err)

	parts := strings.Split(hash, "$")
	require.Len(t, parts, 6, "хеш должен содержать 6 частей, разделенных $")
	assert.Equal(t, "argon2id", parts[1])
	assert.Equal(t, "v=19", parts[2])
	assert.Equal(t, "m=65536,t=3,p=1", parts[3])

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	require.NoError(t, err)
	assert.Len(t, salt, 16, "соль должна быть 16 байт")

	storedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	require.NoError(t, err)
	assert.Len(t, storedHash, 32, "хеш должен быть 32 байта")
}

func TestVerifyPasswordArgon2idDifferentPasswords(t *testing.T) {
	hash1, err := hashPasswordArgon2id("password1")
	require.NoError(t, err)
	hash2, err := hashPasswordArgon2id("password2")
	require.NoError(t, err)

	assert.True(t, verifyPasswordArgon2id(hash1, "password1"))
	assert.False(t, verifyPasswordArgon2id(hash1, "password2"))
	assert.True(t, verifyPasswordArgon2id(hash2, "password2"))
	assert.False(t, verifyPasswordArgon2id(hash2, "password1"))
	assert.NotEqual(t, hash1, hash2, "разные пароли должны давать разные хеши")
}

func TestVerifyPasswordArgon2idEdgeCases(t *testing.T) {
	assert.False(t, verifyPasswordArgon2id("", "password"))
	assert.False(t, verifyPasswordArgon2id("$argon2id$v=19$m=65536,t=3,p=1$", "password"))
	assert.False(t, verifyPasswordArgon2id("$argon2id$v=19$m=65536,t=3,p=1$invalid$invalid", "password"))
	assert.False(t, verifyPasswordArgon2id("$argon2id$v=19$m=65536,t=3,p=1$invalid$invalid", ""))
}

func TestHashPasswordArgon2idUniqueSalt(t *testing.T) {
	hash1, err := hashPasswordArgon2id("samepassword")
	require.NoError(t, err)
	hash2, err := hashPasswordArgon2id("samepassword")
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "хеши одного пароля должны отличаться из-за уникальной соли")
}

func TestToString(t *testing.T) {
	assert.Equal(t, "", toString(nil))
	assert.Equal(t, "hello", toString(strPtr("hello")))
}

func strPtr(s string) *string {
	return &s
}
