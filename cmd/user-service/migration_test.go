package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/db"
	"github.com/MAMUER/project/internal/logger"
)

// ---------------------------------------------------------------------------
// seq driver — custom driver for ensurePgsodiumKey tests.
//
// Go 1.26's DB.QueryRowContext internally calls DB.QueryContext, which calls
// driver.QueryContext.  sqlmock intercepts QueryContext but its regex matcher
// cannot reliably match the complex pgsodium.import_key CASE SQL.
// We therefore use a custom driver that returns pre-configured results.
// ---------------------------------------------------------------------------

// rower mirrors the unexported driver.Row interface.
type rower interface {
	Scan(dest ...interface{}) error
}

// _singleIDRow scans a fixed int64 on first Scan call.
type _singleIDRow struct {
	v       int64
	scanned bool
}

func (r *_singleIDRow) Scan(dest ...interface{}) error {
	if !r.scanned {
		r.scanned = true
		if len(dest) > 0 {
			if p, ok := dest[0].(*int64); ok {
				*p = r.v
				return nil
			}
		}
	}
	return nil
}

// _errRow always returns the stored error.
type _errRow struct{ err error }

func (r *_errRow) Scan(_ ...interface{}) error { return r.err }

// _seqRows implements driver.Rows backed by a single-row result set.
// Each call to Next either returns the stored values (first call) or
// io.EOF (subsequent calls).  New instances must be created via
// noRows() or singleRow(col, val) to ensure a fresh done=false state.
type _seqRows struct {
	cols []string
	vals []driver.Value
	exErr error
	done  bool
}

// _noRows implements driver.Rows for queries that return zero rows.
// First call to Next returns io.EOF; subsequent calls also return io.EOF.
// Implements driver.RowsNextResultSet to avoid doClose=true on EOF.
type _noRows struct {
	cols []string
	done bool
}

func (r *_noRows) Columns() []string { return r.cols }

func (r *_noRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	return io.EOF
}

func (r *_noRows) Close() error { return nil }

// _hasNoNext implements driver.RowsNextResultSet: always false.
type _hasNoNext struct{}

func (_hasNoNext) HasNextResultSet() bool { return false }

// noRows returns a driver.Rows that signals "no rows" on first Next call.
// The returned rows also implement driver.RowsNextResultSet so that
// database/sql does not call Close() on the inner driver.Rows when EOF
// is reached (avoiding a double-close / nil rowsi panic).
func noRows(cols ...string) driver.Rows {
	return &_noRowsColWrapper{rows: &_noRows{cols: cols}}
}

// _noRowsColWrapper wraps _noRows with driver.RowsNextResultSet.
type _noRowsColWrapper struct {
	rows *_noRows
}

func (w *_noRowsColWrapper) Columns() []string  { return w.rows.Columns() }
func (w *_noRowsColWrapper) Next(dest []driver.Value) error { return w.rows.Next(dest) }
func (w *_noRowsColWrapper) Close() error                       { return w.rows.Close() }
func (w *_noRowsColWrapper) HasNextResultSet() bool             { return false }

// singleRow returns a _seqRows that yields one row with the given column/value.
func singleRow(col string, val driver.Value) *_seqRows {
	return &_seqRows{
		cols: []string{col},
		vals: []driver.Value{val},
	}
}

// Next satisfies driver.Rows.Next.
// Returns io.EOF when vals is empty (no rows) or when already exhausted.
func (r *_seqRows) Next(dest []driver.Value) error {
	if len(r.vals) == 0 {
		return io.EOF
	}
	if r.done {
		return io.EOF
	}
	r.done = true
	if r.exErr != nil {
		return r.exErr
	}
	for i, v := range r.vals {
		if i < len(dest) {
			dest[i] = v
		}
	}
	return nil
}

func (r *_seqRows) Columns() []string { return r.cols }

func (r *_seqRows) Close() error { return nil }

// _seqResult describes one expected driver call.
type _seqResult struct {
	// qcRows is returned by QueryContext.  Must be a fresh _seqRows per call.
	qcRows driver.Rows
	// qrRow is returned by QueryRowContext.
	qrRow rower
	// qcErr is the error returned by QueryContext (overrides qcRows).
	qcErr error
	// qrErr is the error embedded in the returned row for QueryRowContext.
	qrErr error
}

// _seqDrv / _seqC / _seqCtor: driver registered as "seq_test_driver".
type _seqDrv struct{}

func (d *_seqDrv) Open(name string) (driver.Conn, error) {
	return nil, fmt.Errorf("use OpenConnector")
}

func (d *_seqDrv) OpenConnector(name string) (driver.Connector, error) {
	return &_seqCtor{}, nil
}

type _seqCtor struct{}

func (c *_seqCtor) Driver() driver.Driver { return &_seqDrv{} }

func (c *_seqCtor) Connect(ctx context.Context) (driver.Conn, error) {
	return &_seqC{}, nil
}

type _seqC struct {
	results []_seqResult
	idx     int
}

func (c *_seqC) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}

func (c *_seqC) Close() error { return nil }

func (c *_seqC) Begin() (driver.Tx, error) {
	return nil, errors.New("not implemented")
}

func (c *_seqC) Exec(query string, args []driver.Value) (driver.Result, error) {
	return nil, errors.New("use ExecContext")
}

func (c *_seqC) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return nil, errors.New("not implemented")
}

func (c *_seqC) Ping(ctx context.Context) error { return nil }

func (c *_seqC) PingContext(ctx context.Context) error { return nil }

func (c *_seqC) Query(query string, args []driver.Value) (driver.Rows, error) {
	return nil, errors.New("use QueryContext")
}

// QueryContext returns the next pre-configured driver.Rows or error.
// Never returns io.EOF as an error (causes nil rowsi panic in database/sql);
// use noRows() for empty result sets.
func (c *_seqC) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.idx >= len(c.results) {
		return nil, fmt.Errorf("unexpected query %q (no more expectations)", query)
	}
	r := c.results[c.idx]
	c.idx++
	if r.qcErr != nil {
		return nil, r.qcErr
	}
	if r.qcRows == nil {
		return noRows(), nil
	}
	return r.qcRows, nil
}

// QueryRowContext returns the next pre-configured rower.
func (c *_seqC) QueryRowContext(ctx context.Context, query string, args []driver.NamedValue) rower {
	if c.idx >= len(c.results) {
		return &_errRow{err: fmt.Errorf("unexpected query %q (no more expectations)", query)}
	}
	r := c.results[c.idx]
	c.idx++
	if r.qrErr != nil {
		return &_errRow{err: r.qrErr}
	}
	return r.qrRow
}

func (c *_seqC) CheckNamedValue(nv *driver.NamedValue) error { return nil }

func (c *_seqC) ResetSession(ctx context.Context) error { return nil }

func (c *_seqC) IsValid() bool { return true }

// _connectorWithConn wraps a driver.Conn as a driver.Connector for sql.OpenDB.
type _connectorWithConn struct {
	name string
	conn driver.Conn
}

func (c *_connectorWithConn) Driver() driver.Driver { return &_seqDrv{} }

func (c *_connectorWithConn) Connect(ctx context.Context) (driver.Conn, error) {
	return c.conn, nil
}

var seqDriverCounter int

func init() {
	sql.Register("seq_test_driver", &_seqDrv{})
}

// newSeqDB opens a *sql.DB backed by the seq driver.
// A unique driver name suffix is used per call to avoid connector caching
// by database/sql (which caches by driver name and would share state).
func newSeqDB(results ..._seqResult) *sql.DB {
	seqDriverCounter++
	name := fmt.Sprintf("seq_test_driver_%d", seqDriverCounter)
	sql.Register(name, &_seqDrv{})
	conn := &_seqC{results: results}
	ctor := &_connectorWithConn{conn: conn, name: name}
	return sql.OpenDB(ctor)
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

const testEncryptionKeyHex = "a0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

var defaultSetPgsodiumKeyID func(id int64)

func resetTestEnv(t *testing.T) {
	t.Helper()
	db.ResetTestEncryptionKey()
	db.ResetPgsodiumKeyID()
	db.ResetLastSetPgsodiumKeyIDForTest()
	_setPgsodiumKeyID = defaultSetPgsodiumKeyID
}

func installTestKeySetter(t *testing.T) {
	t.Helper()
	_setPgsodiumKeyID = db.SetPgsodiumKeyIDForTest
}

func newTestLogger(t *testing.T) *logger.Logger {
	t.Helper()
	return logger.New("test")
}

func TestMain(m *testing.M) {
	defaultSetPgsodiumKeyID = _setPgsodiumKeyID
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// validatePiiTable tests
// ---------------------------------------------------------------------------

func TestValidatePiiTable(t *testing.T) {
	tests := []struct {
		name    string
		input   piiTable
		wantErr bool
		errMsg  string
	}{
		{name: "valid users single pair", input: piiTable{name: "users", idCol: "id", pairs: []pair{{enc: "email_encrypted", plain: "email"}}}, wantErr: false},
		{name: "valid email_verifications", input: piiTable{name: "email_verifications", idCol: "id", pairs: []pair{{enc: "token_encrypted", plain: "token"}}}, wantErr: false},
		{name: "valid users all pairs", input: piiTable{name: "users", idCol: "id", pairs: []pair{{enc: "email_encrypted", plain: "email"}, {enc: "full_name_encrypted", plain: "full_name"}, {enc: "nickname_encrypted", plain: "nickname"}}}, wantErr: false},
		{name: "disallowed table", input: piiTable{name: "secret_data", idCol: "id", pairs: []pair{{enc: "ssn_encrypted", plain: "ssn"}}}, wantErr: true, errMsg: `pii migration: table "secret_data" is not allowed`},
		{name: "disallowed id col", input: piiTable{name: "users", idCol: "uuid", pairs: []pair{{enc: "email_encrypted", plain: "email"}}}, wantErr: true, errMsg: `pii migration: id column "uuid" is not allowed`},
		{name: "disallowed enc col", input: piiTable{name: "users", idCol: "id", pairs: []pair{{enc: "phone_encrypted", plain: "phone"}}}, wantErr: true, errMsg: `pii migration: column "phone_encrypted"/"phone" is not allowed`},
		{name: "disallowed plain col", input: piiTable{name: "users", idCol: "id", pairs: []pair{{enc: "email_encrypted", plain: "phone"}}}, wantErr: true, errMsg: `pii migration: column "email_encrypted"/"phone" is not allowed`},
		{name: "disallowed ev id col", input: piiTable{name: "email_verifications", idCol: "user_id", pairs: []pair{{enc: "email_encrypted", plain: "email"}}}, wantErr: true, errMsg: `pii migration: id column "user_id" is not allowed`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePiiTable(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ensurePgsodiumKey tests
// ---------------------------------------------------------------------------

func TestEnsurePgsodiumKey(t *testing.T) {
	t.Run("returns error when encryption key is empty", func(t *testing.T) {
		resetTestEnv(t)
		installTestKeySetter(t)
		defer resetTestEnv(t)

		ctx := context.Background()
		mockDB, _, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		log := newTestLogger(t)
		err = ensurePgsodiumKey(ctx, mockDB, log)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DB_ENCRYPTION_KEY not set")
		assert.Equal(t, int64(0), db.GetLastSetPgsodiumKeyIDForTest())
	})

	t.Run("returns nil and sets keyring ID when key already exists", func(t *testing.T) {
		resetTestEnv(t)
		db.SetTestEncryptionKey([]byte(testEncryptionKeyHex))
		installTestKeySetter(t)
		defer resetTestEnv(t)

		ctx := context.Background()
		// Call 1 (QueryContext → single row id=42): keyring found
		mockDB := newSeqDB(
			_seqResult{qcRows: singleRow("id", int64(42))},
		)
		defer mockDB.Close()

		log := newTestLogger(t)
		err := ensurePgsodiumKey(ctx, mockDB, log)
		assert.NoError(t, err)
		assert.Equal(t, int64(42), db.GetLastSetPgsodiumKeyIDForTest())
	})

	t.Run("imports key and sets keyring ID when not yet in keyring", func(t *testing.T) {
		resetTestEnv(t)
		db.SetTestEncryptionKey([]byte(testEncryptionKeyHex))
		installTestKeySetter(t)
		defer resetTestEnv(t)

		ctx := context.Background()
		// Call 1 (QueryContext → Next→io.EOF → sql.ErrNoRows): keyring miss
		// Call 2 (QueryRowContext): import_key returns id=7
		mockDB := newSeqDB(
			_seqResult{qcRows: noRows("id")},
			_seqResult{qrRow: &_singleIDRow{v: 7}},
		)
		defer mockDB.Close()

		log := newTestLogger(t)
		err := ensurePgsodiumKey(ctx, mockDB, log)
		assert.NoError(t, err)
		assert.Equal(t, int64(7), db.GetLastSetPgsodiumKeyIDForTest())
	})

	t.Run("returns error on unexpected query error for keyring lookup", func(t *testing.T) {
		resetTestEnv(t)
		db.SetTestEncryptionKey([]byte(testEncryptionKeyHex))
		installTestKeySetter(t)
		defer resetTestEnv(t)

		ctx := context.Background()
		// QueryContext returns connection error directly
		mockDB := newSeqDB(
			_seqResult{qcErr: errors.New("connection lost")},
		)
		defer mockDB.Close()

		log := newTestLogger(t)
		err := ensurePgsodiumKey(ctx, mockDB, log)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "query pgsodium key")
		assert.Equal(t, int64(0), db.GetLastSetPgsodiumKeyIDForTest())
	})

	t.Run("returns error when key import fails", func(t *testing.T) {
		resetTestEnv(t)
		db.SetTestEncryptionKey([]byte(testEncryptionKeyHex))
		installTestKeySetter(t)
		defer resetTestEnv(t)

		ctx := context.Background()
		// Call 1 (QueryContext → io.EOF → sql.ErrNoRows): keyring miss
		// Call 2 (QueryRowContext): import_key returns error
		mockDB := newSeqDB(
			_seqResult{qcRows: noRows("id")},
			_seqResult{qrErr: errors.New("import failed")},
		)
		defer mockDB.Close()

		log := newTestLogger(t)
		err := ensurePgsodiumKey(ctx, mockDB, log)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "import pgsodium key")
		assert.Equal(t, int64(0), db.GetLastSetPgsodiumKeyIDForTest())
	})
}

// TestSetPgsodiumKeyIDForTest verifies the test spy records the last-set ID.
func TestSetPgsodiumKeyIDForTest(t *testing.T) {
	db.ResetPgsodiumKeyID()
	db.ResetLastSetPgsodiumKeyIDForTest()
	defer db.ResetPgsodiumKeyID()
	defer db.ResetLastSetPgsodiumKeyIDForTest()

	db.SetPgsodiumKeyIDForTest(55)
	assert.Equal(t, int64(55), db.PgsodiumKeyID())
	assert.Equal(t, int64(55), db.GetLastSetPgsodiumKeyIDForTest())

	db.SetPgsodiumKeyIDForTest(0)
	assert.Equal(t, int64(55), db.PgsodiumKeyID())
	assert.Equal(t, int64(0), db.GetLastSetPgsodiumKeyIDForTest())

	db.SetPgsodiumKeyIDForTest(-1)
	assert.Equal(t, int64(55), db.PgsodiumKeyID())
	assert.Equal(t, int64(-1), db.GetLastSetPgsodiumKeyIDForTest())

	db.ResetLastSetPgsodiumKeyIDForTest()
	assert.Equal(t, int64(0), db.GetLastSetPgsodiumKeyIDForTest())
}

// ---------------------------------------------------------------------------
// buildUsersBackfillQuery tests
// ---------------------------------------------------------------------------

func TestBuildUsersBackfillQuery(t *testing.T) {
	query := buildUsersBackfillQuery(99)
	assert.Contains(t, query, "UPDATE users SET")
	assert.Contains(t, query, "email_encrypted = pgsodium.crypto_aead_det_encrypt(email::text, '', 99)")
	assert.Contains(t, query, "email_hash = encode(digest(lower(email), 'sha256'), 'hex')")
	assert.Contains(t, query, "full_name_encrypted = pgsodium.crypto_aead_det_encrypt(full_name::text, '', 99)")
	assert.Contains(t, query, "full_name_hash = encode(digest(lower(full_name), 'sha256'), 'hex')")
	assert.Contains(t, query, "nickname_encrypted = pgsodium.crypto_aead_det_encrypt(nickname::text, '', 99)")
	assert.Contains(t, query, "nickname_hash = encode(digest(lower(nickname), 'sha256'), 'hex')")
	assert.Contains(t, query, "WHERE email_encrypted IS NULL")
}

// ---------------------------------------------------------------------------
// buildEmailVerificationsBackfillQuery tests
// ---------------------------------------------------------------------------

func TestBuildEmailVerificationsBackfillQuery(t *testing.T) {
	query := buildEmailVerificationsBackfillQuery(5)
	assert.Contains(t, query, "UPDATE email_verifications SET")
	assert.Contains(t, query, "email_encrypted = pgsodium.crypto_aead_det_encrypt(email::text, '', 5)")
	assert.Contains(t, query, "token_encrypted = pgsodium.crypto_aead_det_encrypt(token::text, '', 5)")
	assert.Contains(t, query, "WHERE email_encrypted IS NULL")
}

// ---------------------------------------------------------------------------
// backfillEncryptedPII tests
// ---------------------------------------------------------------------------

func TestBackfillEncryptedPII_SkipsWhenKeyEmpty(t *testing.T) {
	resetTestEnv(t)
	defer resetTestEnv(t)

	ctx := context.Background()
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	log := newTestLogger(t)
	backfillEncryptedPII(ctx, mockDB, log)
}

func TestBackfillEncryptedPII_SkipsWhenKeyIDZero(t *testing.T) {
	resetTestEnv(t)
	db.SetTestEncryptionKey([]byte(testEncryptionKeyHex))
	defer resetTestEnv(t)

	ctx := context.Background()
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	log := newTestLogger(t)
	backfillEncryptedPII(ctx, mockDB, log)
}

// usersBackfillPattern matches the SQL produced by buildUsersBackfillQuery(id).
var usersBackfillPattern = `^UPDATE users SET email_encrypted = pgsodium\.crypto_aead_det_encrypt\(email::text, '', \d+\), email_hash = encode\(digest\(lower\(email\), 'sha256'\), 'hex'\), full_name_encrypted = pgsodium\.crypto_aead_det_encrypt\(full_name::text, '', \d+\), full_name_hash = encode\(digest\(lower\(full_name\), 'sha256'\), 'hex'\), nickname_encrypted = pgsodium\.crypto_aead_det_encrypt\(nickname::text, '', \d+\), nickname_hash = encode\(digest\(lower\(nickname\), 'sha256'\), 'hex'\)\s+WHERE email_encrypted IS NULL$`

// evBackfillPattern matches the SQL produced by buildEmailVerificationsBackfillQuery(id).
var evBackfillPattern = `^UPDATE email_verifications SET email_encrypted = pgsodium\.crypto_aead_det_encrypt\(email::text, '', \d+\), token_encrypted = pgsodium\.crypto_aead_det_encrypt\(token::text, '', \d+\)\s+WHERE email_encrypted IS NULL$`

func TestBackfillEncryptedPII_ExecutesQueries(t *testing.T) {
	resetTestEnv(t)
	db.SetTestEncryptionKey([]byte(testEncryptionKeyHex))
	db.SetPgsodiumKeyID(11)
	defer resetTestEnv(t)

	ctx := context.Background()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	mock.ExpectExec(usersBackfillPattern).WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(evBackfillPattern).WillReturnResult(sqlmock.NewResult(0, 1))

	log := newTestLogger(t)
	backfillEncryptedPII(ctx, mockDB, log)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBackfillEncryptedPII_HandlesExecError(t *testing.T) {
	resetTestEnv(t)
	db.SetTestEncryptionKey([]byte(testEncryptionKeyHex))
	db.SetPgsodiumKeyID(11)
	defer resetTestEnv(t)

	ctx := context.Background()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	mock.ExpectExec(usersBackfillPattern).WillReturnError(errors.New("query failed"))

	log := newTestLogger(t)
	assert.NotPanics(t, func() {
		backfillEncryptedPII(ctx, mockDB, log)
	})
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// reencryptPIIFromPgcrypto tests
// ---------------------------------------------------------------------------

func TestReencryptPIIFromPgcrypto_SkipsWhenKeyEmpty(t *testing.T) {
	resetTestEnv(t)
	defer resetTestEnv(t)

	ctx := context.Background()
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	log := newTestLogger(t)
	reencryptPIIFromPgcrypto(ctx, mockDB, log)
}

func TestReencryptPIIFromPgcrypto_SkipsWhenKeyIDZero(t *testing.T) {
	resetTestEnv(t)
	db.SetTestEncryptionKey([]byte(testEncryptionKeyHex))
	defer resetTestEnv(t)

	ctx := context.Background()
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	log := newTestLogger(t)
	reencryptPIIFromPgcrypto(ctx, mockDB, log)
}
