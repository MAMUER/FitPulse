package db

import (
	"crypto/sha256"
	"encoding/hex"
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
	if testEncryptionKey != nil {
		return testEncryptionKey
	}
	return encryptionKey
}

// EmailHash returns a lowercase SHA256 hex representation for lookup.
// It MUST match the SQL expression `encode(digest(lower(email), 'sha256'), 'hex')`
// used by the migration backfill and pgsodium queries, otherwise lookups by
// email_hash will not match. Empty input returns "".
func EmailHash(email string) string {
	s := strings.ToLower(sanitize.String(email))
	if s == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
