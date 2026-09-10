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

// testEncryptionKey allows tests to override the encryption key returned by
// EncryptionKey().  Production code always sees the real key unless a test
// explicitly calls SetTestEncryptionKey.
var testEncryptionKey []byte

// SetTestEncryptionKey overrides the encryption key for the duration of a test.
// Call ResetTestEncryptionKey (typically via defer) to restore production state.
func SetTestEncryptionKey(key []byte) {
	testEncryptionKey = key
}

// ResetTestEncryptionKey clears any test override.
func ResetTestEncryptionKey() {
	testEncryptionKey = nil
}

// SetPgsodiumKeyID фиксирует идентификатор активного ключа pgsodium,
// полученный из keyring при инициализации.
func SetPgsodiumKeyID(id int64) {
	if id > 0 {
		pgsodiumKeyID = id
	}
}

// ResetPgsodiumKeyID сбрасывает идентификатор ключа pgsodium в 0.
// Используется в тестах для возврата к начальному состоянию.
func ResetPgsodiumKeyID() {
	pgsodiumKeyID = 0
}

// lastSetPgsodiumKeyIDForTest запоминает последний ID, переданный в
// SetPgsodiumKeyID во время теста.
var lastSetPgsodiumKeyIDForTest int64

// SetPgsodiumKeyIDForTest устанавливает pgsodiumKeyID и запоминает значение
// для проверки в тестах.  Используется только в тестах.
func SetPgsodiumKeyIDForTest(id int64) {
	if id > 0 {
		pgsodiumKeyID = id
	}
	lastSetPgsodiumKeyIDForTest = id
}

// ResetLastSetPgsodiumKeyIDForTest сбрасывает lastSetPgsodiumKeyIDForTest в 0.
// Используется в тестах.
func ResetLastSetPgsodiumKeyIDForTest() {
	lastSetPgsodiumKeyIDForTest = 0
}

// GetLastSetPgsodiumKeyIDForTest возвращает последний ID, установленный через
// SetPgsodiumKeyIDForTest.  Используется в тестах для проверки.
func GetLastSetPgsodiumKeyIDForTest() int64 {
	return lastSetPgsodiumKeyIDForTest
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
func PgsodiumRandomEncryptParam(plaintextParam, nonceParam int) string {
	return fmt.Sprintf("pgsodium.crypto_aead_aegis256_encrypt($%d::text, '', %d, $%d)", plaintextParam, pgsodiumKeyID, nonceParam)
}

// PgsodiumDecryptParam возвращает выражение для расшифровки колонки,
// зашифрованной с nonce (aegis256 AEAD).
func PgsodiumDecryptParam(ciphertextColumn, nonceColumn, alias string) string {
	return fmt.Sprintf("convert_from(pgsodium.crypto_aead_aegis256_decrypt(%s, '', %d, %s), 'UTF8') AS %s", ciphertextColumn, pgsodiumKeyID, nonceColumn, alias)
}
