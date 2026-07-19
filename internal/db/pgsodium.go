package db

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/MAMUER/project/internal/sanitize"
)

// pgsodiumKeyID — идентификатор ключа в keyring pgsodium (таблица pgsodium.key),
// под которым шифруются/расшифровываются PII-поля. Устанавливается при старте
// сервиса функцией EnsurePgsodiumKey (см. cmd/user-service/main.go).
var pgsodiumKeyID int64 = 1

// SetPgsodiumKeyID фиксирует идентификатор активного ключа pgsodium,
// полученный из keyring при инициализации.
func SetPgsodiumKeyID(id int64) {
	if id > 0 {
		pgsodiumKeyID = id
	}
}

// PgsodiumKeyID возвращает текущий идентификатор ключа pgsodium.
func PgsodiumKeyID() int64 {
	return pgsodiumKeyID
}

// keyringName — имя ключа в keyring pgsodium.
const pgsodiumKeyringName = "fitpulse_pii"

// PgsodiumKeyringName возвращает имя ключа в keyring pgsodium.
func PgsodiumKeyringName() string {
	return pgsodiumKeyringName
}

// BlindIndex возвращает lowercase hex SHA256 для поиска без утечки plaintext.
// Используется для полей, где применяется рандомизированное шифрование.
func BlindIndex(plaintext string) string {
	return strings.ToLower(hex.EncodeToString([]byte(sanitize.String(plaintext))))
}

// NicknameHash возвращает lowercase hex SHA256 для поиска по nickname.
func NicknameHash(nickname string) string {
	return BlindIndex(nickname)
}

// GenerateNonce генерирует случайный 12-байтовый nonce для aegis256 AEAD.
func GenerateNonce() ([]byte, error) {
	nonce := make([]byte, 12)
	_, err := rand.Read(nonce)
	return nonce, err
}

// PgsodiumRandomEncryptParam возвращает выражение pgsodium для шифрования значения
// с рандомизированным nonce (aegis256 AEAD).
// Результат: pgsodium.crypto_aead_aegis256_encrypt($N::text, ”, <key_id>, $M)::bytea
func PgsodiumRandomEncryptParam(plaintextParam int, nonceParam int) string {
	return fmt.Sprintf("pgsodium.crypto_aead_aegis256_encrypt($%d::text, '', %d, $%d)", plaintextParam, pgsodiumKeyID, nonceParam)
}

// PgsodiumDecryptParam возвращает выражение для расшифровки колонки,
// зашифрованной с nonce (aegis256 AEAD).
func PgsodiumDecryptParam(ciphertextColumn string, nonceColumn string, alias string) string {
	return fmt.Sprintf("convert_from(pgsodium.crypto_aead_aegis256_decrypt(%s, '', %d, %s), 'UTF8') AS %s", ciphertextColumn, pgsodiumKeyID, nonceColumn, alias)
}
