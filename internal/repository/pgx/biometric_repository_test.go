package pgx

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/domain/entity"
)

type mockDBForBiometric struct {
	queryRowFunc func(context.Context, string, ...interface{}) pgx.Row
	execFunc     func(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	beginFunc    func(context.Context, pgx.TxOptions) (pgx.Tx, error)
	queryFunc    func(context.Context, string, ...any) (pgx.Rows, error)
}

func (m *mockDBForBiometric) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, query, args...)
	}
	return &mockRowsForBiometric{}, nil
}

type mockRowsForBiometric struct {
	scanErr   error
	rowsErr   error
	nextCount int
}

func (m *mockRowsForBiometric) Close() {}
func (m *mockRowsForBiometric) Next() bool {
	if m.nextCount > 0 {
		m.nextCount--
		return true
	}
	return false
}
func (m *mockRowsForBiometric) Err() error        { return m.rowsErr }
func (m *mockRowsForBiometric) Scan(...any) error { return m.scanErr }
func (m *mockRowsForBiometric) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}
func (m *mockRowsForBiometric) Columns() []string   { return nil }
func (m *mockRowsForBiometric) RawValues() [][]byte { return nil }
func (m *mockRowsForBiometric) Conn() *pgx.Conn     { return nil }
func (m *mockRowsForBiometric) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}
func (m *mockRowsForBiometric) TypeMap() *pgtype.Map   { return nil }
func (m *mockRowsForBiometric) Values() ([]any, error) { return nil, nil }

func (m *mockDBForBiometric) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(ctx, query, args...)
	}
	return &mockRowForBiometric{}
}

func (m *mockDBForBiometric) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, query, args...)
	}
	return pgconn.CommandTag{}, nil
}

func (m *mockDBForBiometric) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	if m.beginFunc != nil {
		return m.beginFunc(ctx, txOptions)
	}
	return &mockTxForBiometric{}, nil
}

type mockRowForBiometric struct {
	scanErr error
}

func (m *mockRowForBiometric) Scan(dest ...any) error {
	if m.scanErr != nil {
		return m.scanErr
	}
	for i := range dest {
		switch v := dest[i].(type) {
		case *string:
			*v = "test-value"
		case *float64:
			*v = 75.0
		case *time.Time:
			*v = time.Now()
		}
	}
	return nil
}

type mockTxForBiometric struct{}

func (m *mockTxForBiometric) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("INSERT 2"), nil
}

func (m *mockTxForBiometric) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

func (m *mockTxForBiometric) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return &mockRowForBiometric{}
}

func (m *mockTxForBiometric) Begin(ctx context.Context) (pgx.Tx, error) {
	return m, nil
}

func (m *mockTxForBiometric) Commit(ctx context.Context) error {
	return nil
}

func (m *mockTxForBiometric) Rollback(ctx context.Context) error {
	return nil
}

func (m *mockTxForBiometric) Conn() *pgx.Conn {
	return nil
}

func (m *mockTxForBiometric) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (m *mockTxForBiometric) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}

func (m *mockTxForBiometric) LargeObjects() pgx.LargeObjects {
	var lo pgx.LargeObjects
	return lo
}

func (m *mockTxForBiometric) PgConn() *pgconn.PgConn {
	return nil
}

func (m *mockTxForBiometric) Close() error {
	return nil
}

func (m *mockTxForBiometric) Ping(ctx context.Context) error {
	return nil
}

func (m *mockTxForBiometric) TypeMap() *pgtype.Map {
	return nil
}

func (m *mockTxForBiometric) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

func TestBiometricRepositoryPGX_BatchCreate_CommitError(t *testing.T) {
	ctx := context.Background()
	mockDB := &mockDBForBiometric{
		beginFunc: func(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
			return &mockTxForBiometricWithCommitError{}, nil
		},
	}
	repo := NewBiometricRepositoryPGX(mockDB)

	now := time.Now()
	records := []*entity.BiometricRecord{
		{ID: "r1", UserID: "u1", MetricType: "heart_rate", Value: 75.0, Timestamp: now, DeviceType: "watch", Source: "test", CreatedAt: now},
	}

	_, err := repo.BatchCreate(ctx, records)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to commit transaction")
}

func TestBiometricRepositoryPGX_BatchCreate_BeginError(t *testing.T) {
	ctx := context.Background()
	mockDB := &mockDBForBiometric{
		beginFunc: func(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
			return nil, assert.AnError
		},
	}
	repo := NewBiometricRepositoryPGX(mockDB)

	now := time.Now()
	records := []*entity.BiometricRecord{{ID: "r1", UserID: "u1", MetricType: "hr", Value: 1, Timestamp: now}}
	_, err := repo.BatchCreate(ctx, records)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to begin transaction")
}

func TestBiometricRepositoryPGX_BatchCreate_ExecError(t *testing.T) {
	ctx := context.Background()
	mockDB := &mockDBForBiometric{
		beginFunc: func(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
			return &mockTxExecErr{}, nil
		},
	}
	repo := NewBiometricRepositoryPGX(mockDB)

	now := time.Now()
	records := []*entity.BiometricRecord{{ID: "r1", UserID: "u1", MetricType: "hr", Value: 1, Timestamp: now}}
	_, err := repo.BatchCreate(ctx, records)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to insert biometric record")
}

type mockTxExecErr struct{}

func (m *mockTxExecErr) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, assert.AnError
}
func (m *mockTxExecErr) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (m *mockTxExecErr) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return &mockRowForBiometric{}
}
func (m *mockTxExecErr) Begin(ctx context.Context) (pgx.Tx, error) { return m, nil }
func (m *mockTxExecErr) Commit(ctx context.Context) error          { return nil }
func (m *mockTxExecErr) Rollback(ctx context.Context) error        { return nil }
func (m *mockTxExecErr) Conn() *pgx.Conn                           { return nil }
func (m *mockTxExecErr) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (m *mockTxExecErr) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults { return nil }
func (m *mockTxExecErr) LargeObjects() pgx.LargeObjects                               { var lo pgx.LargeObjects; return lo }
func (m *mockTxExecErr) PgConn() *pgconn.PgConn                                       { return nil }
func (m *mockTxExecErr) Close() error                                                 { return nil }
func (m *mockTxExecErr) Ping(ctx context.Context) error                               { return nil }
func (m *mockTxExecErr) TypeMap() *pgtype.Map                                         { return nil }
func (m *mockTxExecErr) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

func TestBiometricRepositoryPGX_BatchCreate_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	mockDB := &mockDBForBiometric{
		beginFunc: func(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
			return &mockTxForBiometric{}, nil
		},
	}
	repo := NewBiometricRepositoryPGX(mockDB)

	now := time.Now()
	records := []*entity.BiometricRecord{{ID: "r1", UserID: "u1", MetricType: "hr", Value: 1, Timestamp: now}}
	_, err := repo.BatchCreate(ctx, records)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request canceled")
}

func TestBiometricRepositoryPGX_Create_Error(t *testing.T) {
	ctx := context.Background()
	repo := &BiometricRepositoryPGX{db: &mockDBQueryRowErr{err: assert.AnError}}
	now := time.Now()
	_, err := repo.Create(ctx, &entity.BiometricRecord{ID: "1", UserID: "u1", MetricType: "hr", Value: 1, Timestamp: now})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to insert biometric record")
}

type mockDBQueryRowErr struct {
	err error
}

func (m *mockDBQueryRowErr) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (m *mockDBQueryRowErr) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return &mockRowForBiometric{scanErr: m.err}
}
func (m *mockDBQueryRowErr) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (m *mockDBQueryRowErr) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return &mockTxForBiometric{}, nil
}

func TestBiometricRepositoryPGX_GetByUserID_Error(t *testing.T) {
	ctx := context.Background()
	repo := &BiometricRepositoryPGX{db: &mockDBQueryErr{}}
	_, err := repo.GetByUserID(ctx, "u1", "", 10, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get biometric records")
}

type mockDBQueryErr struct{}

func (m *mockDBQueryErr) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return nil, assert.AnError
}
func (m *mockDBQueryErr) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return &mockRowForBiometric{}
}
func (m *mockDBQueryErr) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (m *mockDBQueryErr) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return &mockTxForBiometric{}, nil
}

func TestBiometricRepositoryPGX_GetLatest_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &BiometricRepositoryPGX{db: &mockDBQueryRowErr{err: pgx.ErrNoRows}}
	_, err := repo.GetLatest(ctx, "u1", "hr")
	require.Error(t, err)
}

func TestBiometricRepositoryPGX_GetLatest_Error(t *testing.T) {
	ctx := context.Background()
	repo := &BiometricRepositoryPGX{db: &mockDBQueryRowErr{err: assert.AnError}}
	_, err := repo.GetLatest(ctx, "u1", "hr")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get latest biometric record")
}

func TestBiometricRepositoryPGX_Update_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &BiometricRepositoryPGX{db: &mockDBQueryRowErr{err: pgx.ErrNoRows}}
	now := time.Now()
	_, err := repo.Update(ctx, &entity.BiometricRecord{ID: "1", Value: 1, Timestamp: now})
	require.Error(t, err)
}

func TestBiometricRepositoryPGX_Update_Error(t *testing.T) {
	ctx := context.Background()
	repo := &BiometricRepositoryPGX{db: &mockDBQueryRowErr{err: assert.AnError}}
	now := time.Now()
	_, err := repo.Update(ctx, &entity.BiometricRecord{ID: "1", Value: 1, Timestamp: now})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update biometric record")
}

func TestBiometricRepositoryPGX_Delete_Error(t *testing.T) {
	ctx := context.Background()
	repo := &BiometricRepositoryPGX{db: &mockDBExecErr{}}
	err := repo.Delete(ctx, "1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete biometric record")
}

type mockDBExecErr struct{}

func (m *mockDBExecErr) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (m *mockDBExecErr) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return &mockRowForBiometric{}
}
func (m *mockDBExecErr) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, assert.AnError
}
func (m *mockDBExecErr) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return &mockTxForBiometric{}, nil
}

func TestBiometricRepositoryPGX_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &BiometricRepositoryPGX{db: &mockDBQueryRowErr{err: pgx.ErrNoRows}}
	_, err := repo.GetByID(ctx, "1")
	require.Error(t, err)
}

func TestBiometricRepositoryPGX_GetByID_Error(t *testing.T) {
	ctx := context.Background()
	repo := &BiometricRepositoryPGX{db: &mockDBQueryRowErr{err: assert.AnError}}
	_, err := repo.GetByID(ctx, "1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get biometric record")
}

func TestBiometricRepositoryPGX_CountByUser_Error(t *testing.T) {
	ctx := context.Background()
	repo := &BiometricRepositoryPGX{db: &mockDBQueryRowErr{err: assert.AnError}}
	_, err := repo.CountByUser(ctx, "u1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to count biometric records")
}

func TestBiometricRepositoryPGX_DeleteByUser_Error(t *testing.T) {
	ctx := context.Background()
	repo := &BiometricRepositoryPGX{db: &mockDBExecErr{}}
	_, err := repo.DeleteByUser(ctx, "u1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete biometric records")
}

func TestBiometricRepositoryPGX_GetMetricsSummary_QueryError(t *testing.T) {
	ctx := context.Background()
	repo := &BiometricRepositoryPGX{db: &mockDBQueryErr{}}
	_, err := repo.GetMetricsSummary(ctx, "u1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get metrics summary")
}

func TestBiometricRepositoryPGX_GetMetricsSummary_ScanError(t *testing.T) {
	ctx := context.Background()
	repo := &BiometricRepositoryPGX{db: &mockDBQueryRowsScanErr{}}
	_, err := repo.GetMetricsSummary(ctx, "u1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to scan metrics summary")
}

func TestBiometricRepositoryPGX_GetMetricsSummary_RowsErr(t *testing.T) {
	ctx := context.Background()
	repo := &BiometricRepositoryPGX{db: &mockDBQueryRowsErr{}}
	_, err := repo.GetMetricsSummary(ctx, "u1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to iterate metrics summary")
}

type mockRowsSimple struct {
	next bool
	scan func(dest ...any) error
	err  error
}

func (m *mockRowsSimple) Close()                                       {}
func (m *mockRowsSimple) Err() error                                   { return m.err }
func (m *mockRowsSimple) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (m *mockRowsSimple) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (m *mockRowsSimple) Next() bool {
	if m.next {
		m.next = false
		return true
	}
	return false
}
func (m *mockRowsSimple) Scan(dest ...any) error {
	if m.scan != nil {
		return m.scan(dest...)
	}
	return nil
}
func (m *mockRowsSimple) Values() ([]interface{}, error) { return nil, nil }
func (m *mockRowsSimple) RawValues() [][]byte            { return nil }
func (m *mockRowsSimple) Conn() *pgx.Conn                { return nil }
func (m *mockRowsSimple) TypeMap() *pgtype.Map           { return nil }

type mockDBQueryRowsScanErr struct{}

func (m *mockDBQueryRowsScanErr) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return &mockRowsSimple{next: true, scan: func(dest ...any) error { return assert.AnError }}, nil
}
func (m *mockDBQueryRowsScanErr) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return &mockRowForBiometric{}
}
func (m *mockDBQueryRowsScanErr) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (m *mockDBQueryRowsScanErr) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return &mockTxForBiometric{}, nil
}

type mockDBQueryRowsErr struct{}

func (m *mockDBQueryRowsErr) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return &mockRowsSimple{next: false, err: assert.AnError}, nil
}
func (m *mockDBQueryRowsErr) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return &mockRowForBiometric{}
}
func (m *mockDBQueryRowsErr) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (m *mockDBQueryRowsErr) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return &mockTxForBiometric{}, nil
}

type mockTxForBiometricWithCommitError struct{}

func (m *mockTxForBiometricWithCommitError) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (m *mockTxForBiometricWithCommitError) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

func (m *mockTxForBiometricWithCommitError) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return &mockRowForBiometric{}
}

func (m *mockTxForBiometricWithCommitError) Begin(ctx context.Context) (pgx.Tx, error) {
	return m, nil
}

func (m *mockTxForBiometricWithCommitError) Commit(ctx context.Context) error {
	return assert.AnError
}

func (m *mockTxForBiometricWithCommitError) Rollback(ctx context.Context) error {
	return nil
}

func (m *mockTxForBiometricWithCommitError) Conn() *pgx.Conn {
	return nil
}

func (m *mockTxForBiometricWithCommitError) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (m *mockTxForBiometricWithCommitError) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}

func (m *mockTxForBiometricWithCommitError) LargeObjects() pgx.LargeObjects {
	var lo pgx.LargeObjects
	return lo
}

func (m *mockTxForBiometricWithCommitError) PgConn() *pgconn.PgConn {
	return nil
}

func (m *mockTxForBiometricWithCommitError) Close() error {
	return nil
}

func (m *mockTxForBiometricWithCommitError) Ping(ctx context.Context) error {
	return nil
}

func (m *mockTxForBiometricWithCommitError) TypeMap() *pgtype.Map {
	return nil
}

func (m *mockTxForBiometricWithCommitError) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Additional tests for uncovered lines in biometric_repository.go
// ---------------------------------------------------------------------------

func TestBiometricRepositoryPGX_Insert_ReturnsRecord(t *testing.T) {
	repo := NewBiometricRepositoryPGX(&mockDBForBiometric{
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			return &mockRowForBiometric{scanErr: nil}
		},
	})
	ctx := context.Background()

	record := &entity.BiometricRecord{
		UserID:     "user-1",
		MetricType: "heart_rate",
		Value:      72.0,
		DeviceType: "watch",
		Timestamp:  time.Now(),
	}

	result, err := repo.Create(ctx, record)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "user-1", result.UserID)
}

func TestBiometricRepositoryPGX_BatchCreate_ReturnsInsertedCount(t *testing.T) {
	repo := NewBiometricRepositoryPGX(&mockDBForBiometric{
		beginFunc: func(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
			return &mockTxForBiometric{}, nil
		},
		execFunc: func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("INSERT 2"), nil
		},
	})
	ctx := context.Background()

	records, err := repo.BatchCreate(ctx, []*entity.BiometricRecord{
		{UserID: "user-1", MetricType: "heart_rate", Value: 72.5},
		{UserID: "user-1", MetricType: "heart_rate", Value: 73.0},
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, records)
}

func TestBiometricRepositoryPGX_GetByUserID_ReturnsRecords(t *testing.T) {
	repo := NewBiometricRepositoryPGX(&mockDBForBiometric{
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			return &mockRowForBiometric{}
		},
		execFunc: func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, nil
		},
		queryFunc: func(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
			return &mockRowsForBiometric{nextCount: 1}, nil
		},
	})
	ctx := context.Background()

	records, err := repo.GetByUserID(ctx, "user-1", "heart_rate", 10, 0)
	assert.NoError(t, err)
	assert.NotNil(t, records)
	assert.IsType(t, []*entity.BiometricRecord{}, records)
}

func TestBiometricRepositoryPGX_GetLatest_ReturnsRecord(t *testing.T) {
	repo := NewBiometricRepositoryPGX(&mockDBForBiometric{
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			return &mockRowForBiometric{}
		},
	})
	ctx := context.Background()

	record, err := repo.GetLatest(ctx, "user-1", "heart_rate")
	assert.NoError(t, err)
	assert.NotNil(t, record)
}

func TestBiometricRepositoryPGX_Update_ReturnsRecord(t *testing.T) {
	repo := NewBiometricRepositoryPGX(&mockDBForBiometric{
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			return &mockRowForBiometric{}
		},
	})
	ctx := context.Background()

	record := &entity.BiometricRecord{
		ID:         "r1",
		UserID:     "user-1",
		MetricType: "heart_rate",
		Value:      75.0,
		Timestamp:  time.Now(),
	}

	result, err := repo.Update(ctx, record)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBiometricRepositoryPGX_Delete_ReturnsNil(t *testing.T) {
	repo := NewBiometricRepositoryPGX(&mockDBForBiometric{
		execFunc: func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, nil
		},
	})
	ctx := context.Background()

	err := repo.Delete(ctx, "r1")
	assert.NoError(t, err)
}

func TestBiometricRepositoryPGX_GetByID_ReturnsRecord(t *testing.T) {
	repo := NewBiometricRepositoryPGX(&mockDBForBiometric{
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			return &mockRowForBiometric{}
		},
	})
	ctx := context.Background()

	record, err := repo.GetByID(ctx, "r1")
	assert.NoError(t, err)
	assert.NotNil(t, record)
}

func TestBiometricRepositoryPGX_CountByUser_ReturnsCount(t *testing.T) {
	repo := NewBiometricRepositoryPGX(&mockDBForBiometric{
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			return &mockRowForBiometric{}
		},
	})
	ctx := context.Background()

	count, err := repo.CountByUser(ctx, "user-1")
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestBiometricRepositoryPGX_DeleteByUser_ReturnsCount(t *testing.T) {
	repo := NewBiometricRepositoryPGX(&mockDBForBiometric{
		execFunc: func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("DELETE 3"), nil
		},
	})
	ctx := context.Background()

	count, err := repo.DeleteByUser(ctx, "user-1")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestBiometricRepositoryPGX_GetMetricsSummary_ReturnsSummary(t *testing.T) {
	repo := NewBiometricRepositoryPGX(&mockDBForBiometric{
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			return &mockRowForBiometric{}
		},
	})
	ctx := context.Background()

	summary, err := repo.GetMetricsSummary(ctx, "user-1")
	assert.NoError(t, err)
	assert.NotNil(t, summary)
}
func TestBiometricRepositoryPGX_GetMetricsSummary_WithRows(t *testing.T) {
	repo := NewBiometricRepositoryPGX(&mockDBForBiometric{
		queryFunc: func(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
			return &mockRowsForBiometric{nextCount: 1}, nil
		},
	})
	ctx := context.Background()

	summary, err := repo.GetMetricsSummary(ctx, "user-1")
	assert.NoError(t, err)
	assert.NotNil(t, summary)
}
