package pgx

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
)

type mockRow struct {
	scanFunc func(dest ...interface{}) error
}

func (m *mockRow) Scan(dest ...interface{}) error {
	return m.scanFunc(dest...)
}

type mockRows struct {
	rows      [][]interface{}
	idx       int
	scanFunc  func(dest ...interface{}) error
	err       error
	closeErr  error
}

func newMockRows(rows [][]interface{}, scanFunc func(dest ...interface{}) error) *mockRows {
	return &mockRows{rows: rows, scanFunc: scanFunc}
}

func (m *mockRows) Next() bool {
	if m.idx < len(m.rows) {
		m.idx++
		return true
	}
	return false
}

func (m *mockRows) Scan(dest ...interface{}) error {
	if m.scanFunc != nil {
		return m.scanFunc(dest...)
	}
	if m.idx-1 < len(m.rows) {
		row := m.rows[m.idx-1]
		for i, d := range dest {
			if i < len(row) {
				switch ptr := d.(type) {
				case *string:
					if s, ok := row[i].(string); ok {
						*ptr = s
					}
				case *int:
					if n, ok := row[i].(int); ok {
						*ptr = n
					}
				case *int64:
					if n, ok := row[i].(int64); ok {
						*ptr = n
					}
				case *bool:
					if b, ok := row[i].(bool); ok {
						*ptr = b
					}
				case *time.Time:
					if t, ok := row[i].(time.Time); ok {
						*ptr = t
					}
				case *[]byte:
					if b, ok := row[i].([]byte); ok {
						*ptr = b
					}
				default:
					return fmt.Errorf("unsupported scan type: %T", d)
				}
			}
		}
		return nil
	}
	return errors.New("no more rows")
}

func (m *mockRows) Err() error {
	return m.err
}

func (m *mockRows) Close() {
	// no-op
}

func (m *mockRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (m *mockRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (m *mockRows) Values() ([]any, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRows) RawValues() [][]byte {
	return nil
}

func (m *mockRows) Conn() *pgx.Conn {
	return nil
}

type mockTx struct {
	execFunc  func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	commitErr error
	rollbackErr error
}

func (m *mockTx) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return m.execFunc(ctx, sql, args...)
}

func (m *mockTx) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTx) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return &mockRow{scanFunc: func(dest ...interface{}) error {
		return errors.New("not implemented")
	}}
}

func (m *mockTx) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTx) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTx) Commit(ctx context.Context) error {
	return m.commitErr
}

func (m *mockTx) Rollback(ctx context.Context) error {
	return m.rollbackErr
}

func (m *mockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("not implemented")
}

func (m *mockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}

func (m *mockTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (m *mockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTx) Conn() *pgx.Conn {
	return nil
}

type mockDB struct {
	queryFunc      func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error)
	queryRowFunc   func(ctx context.Context, query string, args ...interface{}) pgx.Row
	execFunc       func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error)
	beginTxFunc    func(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

func (m *mockDB) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return m.queryFunc(ctx, query, args...)
}

func (m *mockDB) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return m.queryRowFunc(ctx, query, args...)
}

func (m *mockDB) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return m.execFunc(ctx, query, args...)
}

func (m *mockDB) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return m.beginTxFunc(ctx, txOptions)
}

func setupDeviceRepo(t *testing.T) (*DeviceRepositoryPGX, *mockDB) {
	t.Helper()
	mock := &mockDB{}
	repo := NewDeviceRepositoryPGX(mock)
	return repo, mock
}

func TestDeviceRepositoryPGX_List_Success(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()
	now := time.Now()

	mock.queryFunc = func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
		assert.Contains(t, query, "SELECT id")
		assert.Equal(t, []interface{}{"user-1"}, args)
		return newMockRows([][]interface{}{
			{"dev-1", "user-1", "watch", "Apple Watch", true, now},
		}, nil), nil
	}

	result, err := repo.List(ctx, "user-1")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "dev-1", result[0].ID)
	assert.Equal(t, "Apple Watch", result[0].DeviceName)
}

func TestDeviceRepositoryPGX_List_QueryError(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.queryFunc = func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
		return nil, assert.AnError
	}

	result, err := repo.List(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestDeviceRepositoryPGX_Create_Success(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			ptr := dest[0].(*string)
			*ptr = "dev-1"
			return nil
		}}
	}

	device := &entity.Device{
		UserID:     "user-1",
		DeviceType: "watch",
		DeviceName: "Apple Watch",
		Token:      "token-1",
	}
	result, err := repo.Create(ctx, device)
	require.NoError(t, err)
	assert.Equal(t, "dev-1", result.ID)
}

func TestDeviceRepositoryPGX_Create_QueryRowError(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			return assert.AnError
		}}
	}

	device := &entity.Device{UserID: "user-1", DeviceType: "watch", DeviceName: "Apple Watch"}
	result, err := repo.Create(ctx, device)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestDeviceRepositoryPGX_Delete_Success(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.execFunc = func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
		assert.Contains(t, query, "DELETE FROM devices")
		assert.Equal(t, []interface{}{"user-1", "dev-1"}, args)
		return pgconn.NewCommandTag("DELETE 1"), nil
	}

	err := repo.Delete(ctx, "user-1", "dev-1")
	require.NoError(t, err)
}

func TestDeviceRepositoryPGX_Delete_Error(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.execFunc = func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
		return pgconn.CommandTag{}, assert.AnError
	}

	err := repo.Delete(ctx, "user-1", "dev-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestDeviceRepositoryPGX_GetByID_Success(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()
	now := time.Now()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			if len(dest) >= 6 {
				if s, ok := dest[0].(*string); ok {
					*s = "dev-1"
				}
				if s, ok := dest[1].(*string); ok {
					*s = "user-1"
				}
				if s, ok := dest[2].(*string); ok {
					*s = "watch"
				}
				if s, ok := dest[3].(*string); ok {
					*s = "Apple Watch"
				}
				if b, ok := dest[4].(*bool); ok {
					*b = true
				}
				if t, ok := dest[5].(*time.Time); ok {
					*t = now
				}
			}
			return nil
		}}
	}

	result, err := repo.GetByID(ctx, "user-1", "dev-1")
	require.NoError(t, err)
	assert.Equal(t, "dev-1", result.ID)
	assert.Equal(t, "Apple Watch", result.DeviceName)
}

func TestDeviceRepositoryPGX_GetByID_NotFound(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			return pgx.ErrNoRows
		}}
	}

	result, err := repo.GetByID(ctx, "user-1", "dev-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.IsNotFound(err))
}

func TestDeviceRepositoryPGX_GetByID_Error(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			return assert.AnError
		}}
	}

	result, err := repo.GetByID(ctx, "user-1", "dev-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestDeviceRepositoryPGX_UpdateLastSync_Success(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.execFunc = func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
		assert.Contains(t, query, "UPDATE devices")
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}

	err := repo.UpdateLastSync(ctx, "dev-1")
	require.NoError(t, err)
}

func TestDeviceRepositoryPGX_UpdateLastSync_Error(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.execFunc = func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
		return pgconn.CommandTag{}, assert.AnError
	}

	err := repo.UpdateLastSync(ctx, "dev-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestDeviceRepositoryPGX_CountByUser_Success(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			ptr := dest[0].(*int)
			*ptr = 3
			return nil
		}}
	}

	count, err := repo.CountByUser(ctx, "user-1")
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestDeviceRepositoryPGX_CountByUser_Error(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			return assert.AnError
		}}
	}

	count, err := repo.CountByUser(ctx, "user-1")
	require.Error(t, err)
	assert.Equal(t, 0, count)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestDeviceRepositoryPGX_ExistsConnected_Success(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			ptr := dest[0].(*bool)
			*ptr = true
			return nil
		}}
	}

	exists, err := repo.ExistsConnected(ctx, "user-1", "watch")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestDeviceRepositoryPGX_ExistsConnected_Error(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			return assert.AnError
		}}
	}

	exists, err := repo.ExistsConnected(ctx, "user-1", "watch")
	require.Error(t, err)
	assert.False(t, exists)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestDeviceRepositoryPGX_DisconnectAll_Success(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.execFunc = func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
		assert.Contains(t, query, "UPDATE devices")
		return pgconn.NewCommandTag("UPDATE 2"), nil
	}

	err := repo.DisconnectAll(ctx, "user-1")
	require.NoError(t, err)
}

func TestDeviceRepositoryPGX_DisconnectAll_Error(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.execFunc = func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
		return pgconn.CommandTag{}, assert.AnError
	}

	err := repo.DisconnectAll(ctx, "user-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestDeviceRepositoryPGX_ListConnected_Success(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()
	now := time.Now()

	mock.queryFunc = func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
		assert.Contains(t, query, "is_connected = true")
		return newMockRows([][]interface{}{
			{"dev-1", "user-1", "watch", "Apple Watch", true, now},
		}, nil), nil
	}

	result, err := repo.ListConnected(ctx, "user-1")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.True(t, result[0].IsConnected)
}

func TestDeviceRepositoryPGX_ListConnected_QueryError(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.queryFunc = func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
		return nil, assert.AnError
	}

	result, err := repo.ListConnected(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestDeviceRepositoryPGX_BatchCreate_Success(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	var beginTxCalled bool
	mock.beginTxFunc = func(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
		beginTxCalled = true
		execCount := 0
		return &mockTx{
execFunc: func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			execCount++
			return pgconn.NewCommandTag("INSERT 0 1"), nil
		},
			commitErr: nil,
		}, nil
	}

	devices := []*entity.Device{
		{UserID: "user-1", DeviceType: "watch", DeviceName: "Apple Watch", Token: "tok-1"},
		{UserID: "user-1", DeviceType: "band", DeviceName: "Fitbit", Token: "tok-2"},
	}

	inserted, err := repo.BatchCreate(ctx, devices)
	require.NoError(t, err)
	assert.Equal(t, 2, inserted)
	assert.True(t, beginTxCalled)
}

func TestDeviceRepositoryPGX_BatchCreate_Empty(t *testing.T) {
	repo, _ := setupDeviceRepo(t)
	ctx := context.Background()

	inserted, err := repo.BatchCreate(ctx, []*entity.Device{})
	require.NoError(t, err)
	assert.Equal(t, 0, inserted)
}

func TestDeviceRepositoryPGX_BatchCreate_BeginTxError(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.beginTxFunc = func(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
		return nil, assert.AnError
	}

	inserted, err := repo.BatchCreate(ctx, []*entity.Device{{UserID: "user-1"}})
	require.Error(t, err)
	assert.Equal(t, 0, inserted)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestDeviceRepositoryPGX_BatchCreate_ExecError(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.beginTxFunc = func(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
		return &mockTx{
			execFunc: func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
				return pgconn.CommandTag{}, assert.AnError
			},
		}, nil
	}

	inserted, err := repo.BatchCreate(ctx, []*entity.Device{{UserID: "user-1"}})
	require.Error(t, err)
	assert.Equal(t, 0, inserted)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestDeviceRepositoryPGX_BatchCreate_CommitError(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.beginTxFunc = func(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
		return &mockTx{
execFunc: func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("INSERT 0 1"), nil
		},
			commitErr: assert.AnError,
		}, nil
	}

	inserted, err := repo.BatchCreate(ctx, []*entity.Device{{UserID: "user-1"}})
	require.Error(t, err)
	assert.Equal(t, 0, inserted)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestDeviceRepositoryPGX_DeleteInactive_Success(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.execFunc = func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
		assert.Contains(t, query, "DELETE FROM devices")
		return pgconn.NewCommandTag("DELETE 5"), nil
	}

	count, err := repo.DeleteInactive(ctx, time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

func TestDeviceRepositoryPGX_DeleteInactive_Error(t *testing.T) {
	repo, mock := setupDeviceRepo(t)
	ctx := context.Background()

	mock.execFunc = func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
		return pgconn.CommandTag{}, assert.AnError
	}

	count, err := repo.DeleteInactive(ctx, time.Now())
	require.Error(t, err)
	assert.Equal(t, int64(0), count)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}
