package db

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlindIndexFormat(t *testing.T) {
	got := BlindIndex("Test@Example.COM")
	expected := strings.ToLower(hex.EncodeToString([]byte("Test@Example.COM")))

	assert.Equal(t, expected, got)
	assert.Equal(t, strings.ToLower(got), got)
	assert.NotEmpty(t, got)
	assert.Equal(t, got, BlindIndex("Test@Example.COM"))
}

func TestPgsodiumKeyringName(t *testing.T) {
	assert.Equal(t, "fitpulse_pii", PgsodiumKeyringName())
}

func TestSetPgsodiumKeyID(t *testing.T) {
	defer ResetPgsodiumKeyID()

	SetPgsodiumKeyID(7)
	assert.Equal(t, int64(7), PgsodiumKeyID())

	// Некорректный id (<= 0) не должен перезаписывать текущий.
	SetPgsodiumKeyID(0)
	assert.Equal(t, int64(7), PgsodiumKeyID())

	ResetPgsodiumKeyID()
	assert.Equal(t, int64(0), PgsodiumKeyID())
}

func TestSetPgsodiumKeyIDForTest(t *testing.T) {
	defer ResetLastSetPgsodiumKeyIDForTest()
	defer ResetPgsodiumKeyID()

	SetPgsodiumKeyIDForTest(42)
	assert.Equal(t, int64(42), PgsodiumKeyID())
	assert.Equal(t, int64(42), GetLastSetPgsodiumKeyIDForTest())

	SetPgsodiumKeyIDForTest(-1)
	assert.Equal(t, int64(42), PgsodiumKeyID(), "некорректный id не должен менять активный ключ")
	// lastSet запоминает любое переданное значение, включая некорректное.
	assert.Equal(t, int64(-1), GetLastSetPgsodiumKeyIDForTest())

	ResetLastSetPgsodiumKeyIDForTest()
	assert.Equal(t, int64(0), GetLastSetPgsodiumKeyIDForTest())
}

func TestPgsodiumRandomEncryptParam(t *testing.T) {
	defer ResetPgsodiumKeyID()
	SetPgsodiumKeyID(3)

	got := PgsodiumRandomEncryptParam(1, 2)
	assert.Equal(t, "pgsodium.crypto_aead_aegis256_encrypt($1::text, '', 3, $2)", got)
}

func TestPgsodiumRefreshTokenInsertQuery(t *testing.T) {
	defer ResetPgsodiumKeyID()
	SetPgsodiumKeyID(9)

	insert := PgsodiumRefreshTokenInsertQuery()
	require.Contains(t, insert, "INSERT INTO refresh_tokens (token_hash, token_encrypted, token_nonce, user_id, expires_at) VALUES ($1,")
	require.Contains(t, insert, "pgsodium.crypto_aead_aegis256_encrypt($2::text, '', 9, $3)")
	require.Contains(t, insert, "$4, $5)")
}

func TestPgsodiumRefreshTokenSelectQuery(t *testing.T) {
	defer ResetPgsodiumKeyID()
	SetPgsodiumKeyID(9)

	sel := PgsodiumRefreshTokenSelectQuery()
	require.Contains(t, sel, "SELECT id, user_id,")
	require.Contains(t, sel, "pgsodium.crypto_aead_aegis256_decrypt(token_encrypted, '', 9, token_nonce)")
	require.Contains(t, sel, "AS token")
	require.Contains(t, sel, "WHERE token_hash = $1 AND revoked = FALSE AND expires_at > NOW()")
}
