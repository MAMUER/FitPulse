package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/MAMUER/project/internal/sanitize"
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

	// pgsodium 3.x stores a UUID in key.id and the numeric key identifier in
	// key.key_id. The server root key is provided by the pgsodium_getkey script,
	// so a named derived key is enough; importing raw key material is neither
	// required nor supported (there is no pgsodium.import_key in 3.x).
	var id int64
	err := database.QueryRowContext(ctx,
		`SELECT key_id FROM pgsodium.key WHERE name = $1 AND key_type = 'aead-det'`,
		db.PgsodiumKeyringName(),
	).Scan(&id)
	if err == nil {
		_setPgsodiumKeyID(id)
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("query pgsodium key: %w", err)
	}

	err = database.QueryRowContext(ctx,
		`SELECT (pgsodium.create_key('aead-det'::pgsodium.key_type, $1)).key_id`,
		db.PgsodiumKeyringName(),
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
	nonce string
}

type piiTable struct {
	name  string
	idCol string
	pairs []pair
}

var allowedPiiTables = map[string][]string{
	"users":               {"id", "email", "email_encrypted", "email_nonce", "full_name", "full_name_encrypted", "full_name_nonce", "nickname", "nickname_encrypted", "nickname_nonce"},
	"email_verifications": {"id", "email", "email_encrypted", "email_nonce", "token", "token_encrypted", "token_nonce"},
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
		if p.nonce != "" && !allowedSet[p.nonce] {
			return fmt.Errorf("pii migration: nonce column %q is not allowed for table %q", p.nonce, t.name)
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
			{"email_encrypted", "email", "email_nonce"},
			{"full_name_encrypted", "full_name", "full_name_nonce"},
			{"nickname_encrypted", "nickname", "nickname_nonce"},
		}},
		{"email_verifications", "id", []pair{
			{"email_encrypted", "email", "email_nonce"},
			{"token_encrypted", "token", "token_nonce"},
		}},
	}

	for _, t := range tables {
		migrateTablePII(ctx, database, log, t, key, id)
	}
}

type piiMigrationContext struct {
	database *sql.DB
	log      *logger.Logger
	t        piiTable
	key      string
	id       int64
}

func migrateTablePII(ctx context.Context, database *sql.DB, log *logger.Logger, t piiTable, key string, id int64) {
	if err := validatePiiTable(t); err != nil {
		log.Error("Invalid PII migration target", zap.Error(err), zap.String("table", sanitize.LogString(t.name)))
		return
	}
	cols := []string{t.idCol}
	for _, p := range t.pairs {
		cols = append(cols, p.enc)
		if p.nonce != "" {
			cols = append(cols, p.nonce)
		}
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
	rows, err := database.QueryContext(ctx, selectBuilder.String()) // NOSONAR go:S2077
	if err != nil {
		log.Error("Failed to scan PII rows for migration", zap.Error(err), zap.String("table", sanitize.LogString(t.name)))
		return
	}

	if err := rows.Err(); err != nil {
		log.Error("Failed to iterate PII rows for migration", zap.Error(err), zap.String("table", sanitize.LogString(t.name)))
		return
	}

	scanPtrs := make([]interface{}, len(cols))
	rowVals := make([]interface{}, len(cols))
	for i := range scanPtrs {
		scanPtrs[i] = &rowVals[i]
	}

	migCtx := piiMigrationContext{
		database: database,
		log:      log,
		t:        t,
		key:      key,
		id:       id,
	}
	migrated := int64(0)
	for rows.Next() {
		if err := rows.Scan(scanPtrs...); err != nil {
			log.Error("Failed to scan PII row", zap.Error(err))
			continue
		}
		rowID := fmt.Sprint(rowVals[0])

		if migratePIIRow(ctx, migCtx, rowID, rowVals) {
			migrated++
		}
	}
	if rowErr := rows.Err(); rowErr != nil {
		log.Error("Failed to iterate PII rows for migration", zap.Error(rowErr), zap.String("table", sanitize.LogString(t.name)))
		return
	}
	if closeErr := rows.Close(); closeErr != nil {
		log.Error("Failed to close rows during PII migration", zap.Error(closeErr), zap.String("table", sanitize.LogString(t.name)))
	}
	if migrated > 0 {
		log.Info("Re-encrypted PII from pgcrypto to pgsodium", zap.String("table", sanitize.LogString(t.name)), zap.Int64("rows", migrated))
	}
}

func piiRowIsPgsodium(ctx context.Context, database *sql.DB, keyID int64, pairs []pair, rowVals []interface{}) bool {
	var probe string
	for i, p := range pairs {
		enc := rowVals[1+i*2]
		if database.QueryRowContext(ctx,
			fmt.Sprintf("SELECT convert_from(pgsodium.crypto_aead_det_decrypt($1, '', %d), 'UTF8')", keyID), enc,
		).Scan(&probe) == nil {
			return true
		}
		if p.nonce != "" && database.QueryRowContext(ctx,
			fmt.Sprintf("SELECT convert_from(pgsodium.crypto_aead_aegis256_decrypt($1, '', %d, $2), 'UTF8')", keyID), enc, rowVals[2+i*2],
		).Scan(&probe) == nil {
			return true
		}
	}
	return false
}

func decryptPIIRow(ctx context.Context, database *sql.DB, log *logger.Logger, t piiTable, key string, rowVals []interface{}) ([]string, bool) {
	values := make([]string, 0, len(t.pairs))
	for i, p := range t.pairs {
		var plain sql.NullString
		if dErr := database.QueryRowContext(ctx, "SELECT pgp_sym_decrypt($1, $2)", rowVals[1+i*2], key).Scan(&plain); dErr != nil || !plain.Valid {
			if dErr != nil && strings.Contains(dErr.Error(), "invalid byte sequence for encoding") {
				log.Debug("Skipping legacy PII row with non-UTF8 pgcrypto payload",
					zap.String("table", sanitize.LogString(t.name)), zap.String("col", sanitize.LogString(p.enc)))
			} else {
				log.Warn("Failed to pgcrypto-decrypt during PII migration",
					zap.Error(dErr), zap.String("table", sanitize.LogString(t.name)), zap.String("col", sanitize.LogString(p.enc)))
			}
			return nil, false
		}
		values = append(values, plain.String)
	}
	return values, true
}

func buildPIIUpdate(t piiTable, keyID int64, plainValues []string) (string, []interface{}) {
	setParts := make([]string, 0, len(t.pairs))
	args := make([]interface{}, 0, len(t.pairs)+1)
	for i, p := range t.pairs {
		setParts = append(setParts, fmt.Sprintf("%s = pgsodium.crypto_aead_det_encrypt($%d::text, '', %d)", p.enc, i+1, keyID))
		args = append(args, plainValues[i])
	}
	return strings.Join(setParts, ", "), args
}

func migratePIIRow(ctx context.Context, migCtx piiMigrationContext, rowID string, rowVals []interface{}) bool {
	// Skip rows already encrypted with pgsodium. The seed-admin and the app
	// use nonce-based crypto_aead_aegis256 ciphertexts, which the old
	// deterministic-only probe did not recognize; it then fell through to
	// pgp_sym_decrypt and logged a false warning on every start.
	if piiRowIsPgsodium(ctx, migCtx.database, migCtx.id, migCtx.t.pairs, rowVals) {
		return false
	}
	plainValues, ok := decryptPIIRow(ctx, migCtx.database, migCtx.log, migCtx.t, migCtx.key, rowVals)
	if !ok {
		return false
	}
	setParts, args := buildPIIUpdate(migCtx.t, migCtx.id, plainValues)
	args = append(args, rowID)
	var queryBuilder strings.Builder
	queryBuilder.WriteString("UPDATE ")
	queryBuilder.WriteString(migCtx.t.name)
	queryBuilder.WriteString(" SET ")
	queryBuilder.WriteString(setParts)
	queryBuilder.WriteString(" WHERE ")
	queryBuilder.WriteString(migCtx.t.idCol)
	queryBuilder.WriteString(" = $")
	queryBuilder.WriteString(strconv.Itoa(len(args)))
	query := queryBuilder.String()
	if _, uErr := migCtx.database.ExecContext(ctx, query, args...); uErr != nil { // NOSONAR go:S2077
		migCtx.log.Error("Failed to re-encrypt PII row", zap.Error(uErr), zap.String("table", sanitize.LogString(migCtx.t.name)), zap.String("id", sanitize.LogString(rowID)))
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

	usersColumns := []string{"email", "email_encrypted", "email_hash", "full_name", "full_name_encrypted", "full_name_hash", "nickname", "nickname_encrypted", "nickname_hash"}
	if hasColumns, err := tableHasColumns(ctx, database, "users", usersColumns); err != nil {
		log.Error("Failed to inspect users columns for PII backfill", zap.Error(err))
	} else if !hasColumns {
		log.Info("Skipping legacy PII backfill for users: required legacy columns are absent")
	} else {
		usersQuery := buildUsersBackfillQuery(id)
		res, execErr := database.ExecContext(ctx, usersQuery) // NOSONAR go:S2077
		if execErr != nil {
			log.Error("Failed to backfill PII in users", zap.Error(execErr))
		} else {
			rows, rowsErr := res.RowsAffected()
			if rowsErr != nil {
				log.Warn("Failed to count backfilled PII rows in users", zap.Error(rowsErr))
			}
			log.Info("PII backfill complete for users", zap.Int64("updated", rows))
		}
	}

	emailVerificationColumns := []string{"email", "email_encrypted", "email_hash", "token", "token_encrypted"}
	if hasColumns, err := tableHasColumns(ctx, database, "email_verifications", emailVerificationColumns); err != nil {
		log.Error("Failed to inspect email_verifications columns for PII backfill", zap.Error(err))
	} else if !hasColumns {
		log.Info("Skipping legacy PII backfill for email_verifications: required legacy columns are absent")
	} else {
		emailVerificationsQuery := buildEmailVerificationsBackfillQuery(id)
		if _, execErr := database.ExecContext(ctx, emailVerificationsQuery); execErr != nil { // NOSONAR go:S2077
			log.Error("Failed to backfill PII in email_verifications", zap.Error(execErr))
		}
	}
}

func tableHasColumns(ctx context.Context, database *sql.DB, table string, required []string) (hasColumns bool, err error) {
	rows, err := database.QueryContext(ctx,
		"SELECT column_name FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = $1",
		table,
	)
	if err != nil {
		return false, err
	}
	defer func() {
		if closeErr := rows.Close(); err == nil {
			err = closeErr
		}
	}()

	columns := make(map[string]struct{}, len(required))
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return false, err
		}
		columns[column] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}

	for _, column := range required {
		if _, ok := columns[column]; !ok {
			return false, nil
		}
	}
	return true, nil
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
