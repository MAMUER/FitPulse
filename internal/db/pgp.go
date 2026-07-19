package db

import (
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
	return encryptionKey
}

// EmailHash returns a lowercase SHA256 hex representation for lookup.
func EmailHash(email string) string {
	return strings.ToLower(hex.EncodeToString([]byte(sanitize.String(email))))
}
