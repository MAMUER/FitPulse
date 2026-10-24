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
}

func (m *mockDBForBiometric) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

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
	return nil
}

type mockTxForBiometric struct{}

func (m *mockTxForBiometric) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
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
