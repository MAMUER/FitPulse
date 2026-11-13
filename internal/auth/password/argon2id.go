// Package password implements the Argon2id password hashing format used by
// the user service: a PHC string with m=65536, t=3, p=1, a 16-byte salt and a
// 32-byte hash, both encoded with unpadded standard base64.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2idParams = "m=65536,t=3,p=1"
	saltLength     = 16
	keyLength      = 32
	argon2Time     = 3
	argon2Memory   = 64 * 1024
	argon2Threads  = 1
)

// HashArgon2id returns a PHC-formatted Argon2id hash of plain.
func HashArgon2id(plain string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(plain), salt, argon2Time, argon2Memory, argon2Threads, keyLength)

	return "$argon2id$v=19$" + argon2idParams + "$" +
		base64.RawStdEncoding.EncodeToString(salt) + "$" +
		base64.RawStdEncoding.EncodeToString(hash), nil
}

// VerifyArgon2id reports whether plain matches the stored PHC-formatted hash.
func VerifyArgon2id(stored, plain string) bool {
	parts := strings.Split(stored, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return false
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(hash) == 0 || len(hash) > keyLength {
		return false
	}

	hashLen := len(hash)
	if uint64(hashLen) > uint64(^uint32(0)) {
		return false
	}

	computed := argon2.IDKey([]byte(plain), salt, argon2Time, argon2Memory, argon2Threads, uint32(hashLen))

	return subtle.ConstantTimeCompare(hash, computed) == 1
}
