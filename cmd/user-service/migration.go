package main

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/db"
	"github.com/MAMUER/project/internal/logger"
)

// _setPgsodiumKeyID is the production setter; replaced in tests via
// the unexported setPgsodiumKeyIDForTest to capture the last-set value.
var _setPgsodiumKeyID = db.SetPgsodiumKeyID

// ensurePgsodiumKey идемпотентно импортирует PII-ключ из DB_ENCRYPTION_KEY
// в keyring pgsodium (таблица pgsodium.key) и фиксирует его идентификатор
// в пакете db для использования в шифровании/расшифровке.
func ensurePgsodiumKey(ctx context.Context, database *sql.DB, log *logger.Logger) error {
	key := db.EncryptionKey()
	if len(key) == 0 {
		return errors.New("DB_ENCRYPTION_KEY not set; pgsodium keyring cannot be initialized")
	}

	var id int64
	err := database.QueryRowContext(ctx, `SELECT id FROM pgsodium.key WHERE name = $1`, db.PgsodiumKeyringName()).Scan(&id)
	if err == nil {
		_setPgsodiumKeyID(id)
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("query pgsodium key: %w", err)
	}

	hexKey := hex.EncodeToString(key)
	err = database.QueryRowContext(ctx,
		`SELECT pgsodium.import_key(CASE WHEN $1 ~ '^[0-9a-fA-F]{64}$' THEN decode($1, 'hex') ELSE convert_to($1, 'UTF8') END, $2)`,
		hexKey, db.PgsodiumKeyringName(),
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("import pgsodium key: %w", err)
	}
	_setPgsodiumKeyID(id)
	log.Info("Imported pgsodium PII key", zap.Int64("key_id", id))
	return nil
}

type pair struct {
	enc   string
	plain string
}

type piiTable struct {
	name  string
	idCol string
	pairs []pair
}

var allowedPiiTables = map[string][]string{
	"users":               {"id", "email", "email_encrypted", "full_name", "full_name_encrypted", "nickname", "nickname_encrypted"},
	"email_verifications": {"id", "email", "email_encrypted", "token", "token_encrypted"},
}

func validatePiiTable(t piiTable) error {
	allowedCols, ok := allowedPiiTables[t.name]
	if !ok {
		return fmt.Errorf("pii migration: table %q is not allowed", t.name)
	}
	allowedSet := make(map[string]bool, len(allowedCols))
	for _, c := range allowedCols {
		allowedSet[c] = true
	}
	if !allowedSet[t.idCol] {
		return fmt.Errorf("pii migration: id column %q is not allowed for table %q", t.idCol, t.name)
	}
	for _, p := range t.pairs {
		if !allowedSet[p.enc] || !allowedSet[p.plain] {
			return fmt.Errorf("pii migration: column %q/%q is not allowed for table %q", p.enc, p.plain, t.name)
		}
	}
	return nil
}

// reencryptPIIFromPgcrypto перекодирует существующие PII-поля,
// зашифрованные ранее через pgcrypto (pgp_sym_encrypt), в pgsodium (libsodium AEAD).
// Строки, уже зашифрованные через pgsodium, пропускаются.
func reencryptPIIFromPgcrypto(ctx context.Context, database *sql.DB, log *logger.Logger) {
	key := string(db.EncryptionKey())
	if key == "" {
		return
	}
	id := db.PgsodiumKeyID()
	if id == 0 {
		return
	}

	tables := []piiTable{
		{"users", "id", []pair{
			{"email_encrypted", "email"},
			{"full_name_encrypted", "full_name"},
			{"nickname_encrypted", "nickname"},
		}},
		{"email_verifications", "id", []pair{
			{"email_encrypted", "email"},
			{"token_encrypted", "token"},
		}},
	}

	for _, t := range tables {
		migrateTablePII(ctx, database, log, t, key, id)
	}
}

func migrateTablePII(ctx context.Context, database *sql.DB, log *logger.Logger, t piiTable, key string, id int64) {
	if err := validatePiiTable(t); err != nil {
		log.Error("Invalid PII migration target", zap.Error(err), zap.String("table", t.name))
		return
	}
	cols := []string{t.idCol}
	for _, p := range t.pairs {
		cols = append(cols, p.enc)
	}
	colList := strings.Join(cols, ", ")
	var selectBuilder strings.Builder
	selectBuilder.WriteString("SELECT ")
	selectBuilder.WriteString(colList)
	selectBuilder.WriteString(" FROM ")
	selectBuilder.WriteString(t.name)
	selectBuilder.WriteString(" WHERE ")
	selectBuilder.WriteString(t.pairs[0].enc)
	selectBuilder.WriteString(" IS NOT NULL")
	// Table/column names are validated by validatePiiTable before this point.
	// Values are passed as parameterized arguments below.
	rows, err := database.QueryContext(ctx, selectBuilder.String()) // nolint:gosimple,staticcheck
	if err != nil {
		log.Error("Failed to scan PII rows for migration", zap.Error(err), zap.String("table", t.name))
		return
	}

	if err := rows.Err(); err != nil {
		log.Error("Failed to iterate PII rows for migration", zap.Error(err), zap.String("table", t.name))
		return
	}

	scanPtrs := make([]interface{}, len(cols))
	rowVals := make([]interface{}, len(cols))
	for i := range scanPtrs {
		scanPtrs[i] = &rowVals[i]
	}

	migrated := int64(0)
	for rows.Next() {
		if err := rows.Scan(scanPtrs...); err != nil {
			log.Error("Failed to scan PII row", zap.Error(err))
			continue
		}
		rowID := fmt.Sprint(rowVals[0])

		if migratePIIRow(ctx, database, log, t, key, id, rowID, rowVals) {
			migrated++
		}
	}
	if rowErr := rows.Err(); rowErr != nil {
		log.Error("Failed to iterate PII rows for migration", zap.Error(rowErr), zap.String("table", t.name))
		return
	}
	if closeErr := rows.Close(); closeErr != nil {
		log.Error("Failed to close rows during PII migration", zap.Error(closeErr), zap.String("table", t.name))
	}
	if migrated > 0 {
		log.Info("Re-encrypted PII from pgcrypto to pgsodium", zap.String("table", t.name), zap.Int64("rows", migrated))
	}
}

func migratePIIRow(ctx context.Context, database *sql.DB, log *logger.Logger, t piiTable, key string, id int64, rowID string, rowVals []interface{}) bool {
	var probe string
	if database.QueryRowContext(ctx,
		fmt.Sprintf("SELECT convert_from(pgsodium.crypto_aead_det_decrypt($1, '', %d), 'UTF8')", id), rowVals[1],
	).Scan(&probe) == nil {
		return false
	}

	setParts := make([]string, 0, len(t.pairs))
	args := make([]interface{}, 0, len(t.pairs)+1)
	ai := 1
	for i, p := range t.pairs {
		var plain sql.NullString
		if dErr := database.QueryRowContext(ctx, "SELECT pgp_sym_decrypt($1, $2)", rowVals[i+1], key).Scan(&plain); dErr != nil || !plain.Valid {
			log.Warn("Failed to pgcrypto-decrypt during PII migration",
				zap.Error(dErr), zap.String("table", t.name), zap.String("col", p.enc))
			return false
		}
		args = append(args, plain.String)
		setParts = append(setParts, fmt.Sprintf("%s = pgsodium.crypto_aead_det_encrypt($%d::text, '', %d)", p.enc, ai, id))
		ai++
	}
	if len(setParts) == 0 {
		return false
	}
	args = append(args, rowID)

	var queryBuilder strings.Builder
	queryBuilder.WriteString("UPDATE ")
	queryBuilder.WriteString(t.name)
	queryBuilder.WriteString(" SET ")
	queryBuilder.WriteString(strings.Join(setParts, ", "))
	queryBuilder.WriteString(" WHERE ")
	queryBuilder.WriteString(t.idCol)
	queryBuilder.WriteString(" = $")
	queryBuilder.WriteString(strconv.Itoa(ai))
	query := queryBuilder.String()

	if _, uErr := database.ExecContext(ctx, query, args...); uErr != nil { // nolint:gosimple,staticcheck
		log.Error("Failed to re-encrypt PII row", zap.Error(uErr), zap.String("table", t.name), zap.String("id", rowID))
		return false
	}
	return true
}

// backfillEncryptedPII зашифровывает открытые PII-поля для существующих записей,
// у которых ещё нет pgsodium-шифротекста.
func backfillEncryptedPII(ctx context.Context, database *sql.DB, log *logger.Logger) {
	key := string(db.EncryptionKey())
	if key == "" {
		log.Warn("DB_ENCRYPTION_KEY not set; skipping PII backfill")
		return
	}
	id := db.PgsodiumKeyID()
	if id == 0 {
		log.Warn("pgsodium key not initialized; skipping PII backfill")
		return
	}

	usersQuery := buildUsersBackfillQuery(id)
	res, err := database.ExecContext(ctx, usersQuery) // nolint:gosimple,staticcheck
	if err != nil {
		log.Error("Failed to backfill PII in users", zap.Error(err))
	} else {
		rows, _ := res.RowsAffected()
		log.Info("PII backfill complete for users", zap.Int64("updated", rows))
	}

	emailVerificationsQuery := buildEmailVerificationsBackfillQuery(id)
	_, err = database.ExecContext(ctx, emailVerificationsQuery) // nolint:gosimple,staticcheck
	if err != nil {
		log.Error("Failed to backfill PII in email_verifications", zap.Error(err))
	}
}

func buildUsersBackfillQuery(id int64) string {
	var q strings.Builder
	q.WriteString("UPDATE users SET ")
	q.WriteString("email_encrypted = pgsodium.crypto_aead_det_encrypt(email::text, '', ")
	q.WriteString(strconv.FormatInt(id, 10))
	q.WriteString("), email_hash = encode(digest(lower(email), 'sha256'), 'hex'), ")
	q.WriteString("full_name_encrypted = pgsodium.crypto_aead_det_encrypt(full_name::text, '', ")
	q.WriteString(strconv.FormatInt(id, 10))
	q.WriteString("), full_name_hash = encode(digest(lower(full_name), 'sha256'), 'hex'), ")
	q.WriteString("nickname_encrypted = pgsodium.crypto_aead_det_encrypt(nickname::text, '', ")
	q.WriteString(strconv.FormatInt(id, 10))
	q.WriteString("), nickname_hash = encode(digest(lower(nickname), 'sha256'), 'hex') ")
	q.WriteString(" WHERE email_encrypted IS NULL")
	return q.String()
}

func buildEmailVerificationsBackfillQuery(id int64) string {
	var q strings.Builder
	q.WriteString("UPDATE email_verifications SET ")
	q.WriteString("email_encrypted = pgsodium.crypto_aead_det_encrypt(email::text, '', ")
	q.WriteString(strconv.FormatInt(id, 10))
	q.WriteString("), token_encrypted = pgsodium.crypto_aead_det_encrypt(token::text, '', ")
	q.WriteString(strconv.FormatInt(id, 10))
	q.WriteString(") WHERE email_encrypted IS NULL")
	return q.String()
}
