package password

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashArgon2idFormat(t *testing.T) {
	hash, err := HashArgon2id("correct horse battery staple")
	require.NoError(t, err)

	parts := strings.Split(hash, "$")
	require.Len(t, parts, 6)
	assert.Equal(t, "", parts[0])
	assert.Equal(t, "argon2id", parts[1])
	assert.Equal(t, "v=19", parts[2])
	assert.Equal(t, "m=65536,t=3,p=1", parts[3])

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	require.NoError(t, err)
	assert.Len(t, salt, saltLength)

	digest, err := base64.RawStdEncoding.DecodeString(parts[5])
	require.NoError(t, err)
	assert.Len(t, digest, keyLength)
}

func TestVerifyArgon2id(t *testing.T) {
	hash, err := HashArgon2id("correct horse battery staple")
	require.NoError(t, err)

	assert.True(t, VerifyArgon2id(hash, "correct horse battery staple"))
	assert.False(t, VerifyArgon2id(hash, "wrong password"))
	assert.False(t, VerifyArgon2id(hash, ""))
}

func TestVerifyArgon2idRejectsMalformedHashes(t *testing.T) {
	tests := []string{
		"",
		"$argon2id$v=19$m=65536,t=3,p=1$",
		"$argon2id$v=19$m=65536,t=3,p=1$$",
		"$argon2id$v=19$m=65536,t=3,p=1$!!!$AAAA",
		"$argon2id$v=19$m=65536,t=3,p=1$c2FsdA$",
		"$bcrypt$v=19$m=65536,t=3,p=1$c2FsdA$AAAA",
	}

	for _, stored := range tests {
		assert.False(t, VerifyArgon2id(stored, "password"), "stored=%q", stored)
	}
}
