package db

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/MAMUER/project/internal/config"
	"github.com/MAMUER/project/internal/sanitize"
)

var encryptionKey = func() []byte {
	key := config.GetEnv("DB_ENCRYPTION_KEY", "")
	if key == "" {
		return nil
	}
	if len(key) == 64 && isHex(key) {
		b, err := hex.DecodeString(key)
		if err == nil {
			return b
		}
	}
	return []byte(key)
}()

func isHex(s string) bool {
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

// EncryptionKey returns the database PII encryption key.
// Returns nil if DB_ENCRYPTION_KEY is not configured.
func EncryptionKey() []byte {
	return encryptionKey
}

// PGPEncryptSQL returns a pgcrypto expression for encrypting a column value with a literal key.
// Usage: INSERT INTO t (enc_col) VALUES (PGPEncryptSQL('plaintext', 'key'))
func PGPEncryptSQL(plaintext, key string) string {
	plaintext = sanitize.String(plaintext)
	return fmt.Sprintf("pgp_sym_encrypt(%s, %s)", quoteString(plaintext), quoteString(key))
}

// PGPDecryptSQL returns a pgcrypto expression for decrypting a column with a literal key.
// Usage: SELECT PGPDecryptSQL(enc_col, 'key') AS plaintext FROM t
func PGPDecryptSQL(column, key, alias string) string {
	return fmt.Sprintf("pgp_sym_decrypt(%s, %s) AS %s", column, quoteString(key), alias)
}

// PGPEncryptParam returns a pgcrypto expression for encrypting using a parameter placeholder.
// Usage: VALUES (..., PGPEncryptParam(2, 3), ...) -> pgp_sym_encrypt($2, $3)
func PGPEncryptParam(plaintextParam, keyParam int) string {
	return fmt.Sprintf("pgp_sym_encrypt($%d, $%d)", plaintextParam, keyParam)
}

// PGPDecryptParam returns a pgcrypto expression for decrypting using a parameter placeholder.
// Usage: SELECT ..., PGPDecryptParam("col", 1, "alias") FROM t
func PGPDecryptParam(column string, keyParam int, alias string) string {
	return fmt.Sprintf("pgp_sym_decrypt(%s, $%d) AS %s", column, keyParam, alias)
}

// EmailHash returns a lowercase SHA256 hex representation for lookup.
func EmailHash(email string) string {
	return strings.ToLower(hex.EncodeToString([]byte(sanitize.String(email))))
}

func quoteString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
