package db

import (
	"fmt"

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
