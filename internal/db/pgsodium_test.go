package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetPgsodiumKeyID(t *testing.T) {
	SetPgsodiumKeyID(42)
	assert.Equal(t, int64(42), PgsodiumKeyID())

	SetPgsodiumKeyID(1)
	assert.Equal(t, int64(1), PgsodiumKeyID())

	SetPgsodiumKeyID(0)
	assert.Equal(t, int64(1), PgsodiumKeyID())

	SetPgsodiumKeyID(-5)
	assert.Equal(t, int64(1), PgsodiumKeyID())

	SetPgsodiumKeyID(1)
}

func TestPgsodiumKeyID(t *testing.T) {
	SetPgsodiumKeyID(7)
	assert.Equal(t, int64(7), PgsodiumKeyID())
	SetPgsodiumKeyID(1)
}

func TestPgsodiumKeyringName(t *testing.T) {
	assert.Equal(t, "fitpulse_pii", PgsodiumKeyringName())
}

func TestPgsodiumRandomEncryptParam(t *testing.T) {
	SetPgsodiumKeyID(1)
	sql := PgsodiumRandomEncryptParam(1, 2)
	assert.Contains(t, sql, "pgsodium.crypto_aead_aegis256_encrypt")
	assert.Contains(t, sql, "$1::text")
	assert.Contains(t, sql, ", 1, $2)")
}

func TestPgsodiumDecryptParam_Unique(t *testing.T) {
	SetPgsodiumKeyID(1)
	sql := PgsodiumDecryptParam("u.full_name_encrypted", "u.full_name_nonce", "full_name")
	assert.Contains(t, sql, "pgsodium.crypto_aead_aegis256_decrypt")
	assert.Contains(t, sql, "u.full_name_encrypted")
	assert.Contains(t, sql, "u.full_name_nonce")
	assert.Contains(t, sql, "full_name")
	assert.NotContains(t, sql, "CASE WHEN")
	assert.NotContains(t, sql, "pgsodium.crypto_aead_det_decrypt")
}

func TestPgsodiumRandomEncryptParam_CustomKeyID(t *testing.T) {
	SetPgsodiumKeyID(42)
	sql := PgsodiumRandomEncryptParam(3, 4)
	assert.Contains(t, sql, ", 42, $4)")
	SetPgsodiumKeyID(1)
}
