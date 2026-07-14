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
//
// ВНИМАНИЕ: crypto_aead_det_encrypt — детерминированный AEAD (libsodium).
// Одинаковый plaintext + ключ = одинаковый ciphertext. Это уязвимо к частотному
// анализу при компрометации дампа/бэкапа БД.
//
// Детерминированное шифрование оправдано ТОЛЬКО для полей, где требуется
// точный lookup без расшифровки (токены верификации).
//
// Для PII (email, full_name, nickname) используется рандомизированное
// шифрование + blind index (HMAC-индекс для поиска).
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

// PgsodiumEncryptParam возвращает выражение pgsodium для шифрования значения
// параметра-плейсхолдера с детерминированным AEAD (libsodium).
// Результат: pgsodium.crypto_aead_det_encrypt($N, ”, <key_id>)::bytea
func PgsodiumEncryptParam(plaintextParam int) string {
	return fmt.Sprintf("pgsodium.crypto_aead_det_encrypt($%d::text, '', %d)", plaintextParam, pgsodiumKeyID)
}

// PgsodiumDecryptParam возвращает выражение pgsodium для расшифровки колонки
// с детерминированным AEAD. Результат приводится к text через convert_from.
func PgsodiumDecryptParam(column string, alias string) string {
	return fmt.Sprintf("convert_from(pgsodium.crypto_aead_det_decrypt(%s, '', %d), 'UTF8') AS %s", column, pgsodiumKeyID, alias)
}

// PgsodiumEncryptSQL возвращает выражение pgsodium для шифрования литерала.
func PgsodiumEncryptSQL(plaintext string) string {
	plaintext = sanitize.String(plaintext)
	return fmt.Sprintf("pgsodium.crypto_aead_det_encrypt(%s, '', %d)", quoteString(plaintext), pgsodiumKeyID)
}

// PgsodiumDecryptSQL возвращает выражение pgsodium для расшифровки колонки-литерала.
func PgsodiumDecryptSQL(column, alias string) string {
	return fmt.Sprintf("convert_from(pgsodium.crypto_aead_det_decrypt(%s, '', %d), 'UTF8') AS %s", column, pgsodiumKeyID, alias)
}

// keyringName — имя ключа в pgsodium.key, под которым хранится PII-ключ.
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
// Результат: pgsodium.crypto_aead_aegis256_encrypt($N, ”, <key_id>, $M)::bytea
func PgsodiumRandomEncryptParam(plaintextParam int, nonceParam int) string {
	return fmt.Sprintf("pgsodium.crypto_aead_aegis256_encrypt($%d::text, '', %d, $%d)", plaintextParam, pgsodiumKeyID, nonceParam)
}

// PgsodiumDecryptDualParam возвращает выражение для расшифровки колонки,
// зашифрованной либо детерминированно (legacy), либо с nonce (aegis256).
func PgsodiumDecryptDualParam(ciphertextColumn string, nonceColumn string, alias string) string {
	return fmt.Sprintf("CASE WHEN length(%s) >= 12 THEN convert_from(pgsodium.crypto_aead_aegis256_decrypt(%s, '', %d, %s), 'UTF8') ELSE convert_from(pgsodium.crypto_aead_det_decrypt(%s, '', %d), 'UTF8') END AS %s",
		nonceColumn, ciphertextColumn, pgsodiumKeyID, nonceColumn, ciphertextColumn, pgsodiumKeyID, alias)
}
