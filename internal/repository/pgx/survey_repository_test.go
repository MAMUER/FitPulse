package pgx

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"time"
)

type mockDBForSurvey struct {
	queryRowFunc func(context.Context, string, ...interface{}) pgx.Row
	execFunc     func(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	beginFunc    func(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

func (m *mockDBForSurvey) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

func (m *mockDBForSurvey) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(ctx, query, args...)
	}
	return &mockRowForSurvey{}
}

func (m *mockDBForSurvey) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, query, args...)
	}
	return pgconn.CommandTag{}, nil
}

func (m *mockDBForSurvey) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	if m.beginFunc != nil {
		return m.beginFunc(ctx, txOptions)
	}
	return &mockTxForSurvey{}, nil
}

type mockRowForSurvey struct {
	scanErr error
}

func (m *mockRowForSurvey) Scan(dest ...interface{}) error {
	if m.scanErr != nil {
		return m.scanErr
	}
	return nil
}

type mockTxForSurvey struct{}

func (m *mockTxForSurvey) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (m *mockTxForSurvey) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

func (m *mockTxForSurvey) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return &mockRowForSurvey{}
}

func (m *mockTxForSurvey) Begin(ctx context.Context) (pgx.Tx, error) {
	return m, nil
}

func (m *mockTxForSurvey) Commit(ctx context.Context) error {
	return nil
}

func (m *mockTxForSurvey) Rollback(ctx context.Context) error {
	return nil
}

func (m *mockTxForSurvey) Conn() *pgx.Conn {
	return nil
}

func (m *mockTxForSurvey) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (m *mockTxForSurvey) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}

func (m *mockTxForSurvey) LargeObjects() pgx.LargeObjects {
	var lo pgx.LargeObjects
	return lo
}

func (m *mockTxForSurvey) PgConn() *pgconn.PgConn {
	return nil
}

func (m *mockTxForSurvey) Close() error {
	return nil
}

func (m *mockTxForSurvey) Ping(ctx context.Context) error {
	return nil
}

func (m *mockTxForSurvey) TypeMap() *pgtype.Map {
	return nil
}

func (m *mockTxForSurvey) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

func TestSurveyRepositoryPGX_SaveSurvey_Error(t *testing.T) {
	ctx := context.Background()
	mockDB := &mockDBForSurvey{
		execFunc: func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, assert.AnError
		},
	}
	repo := NewSurveyRepositoryPGX(mockDB)

	err := repo.SaveSurvey(ctx, "user-1", map[string]interface{}{"key": "value"}, true, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save survey")
}

func TestSurveyRepositoryPGX_LoadSurvey_QueryError(t *testing.T) {
	ctx := context.Background()
	mockDB := &mockDBForSurvey{
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			return &mockRowForSurvey{scanErr: assert.AnError}
		},
	}
	repo := NewSurveyRepositoryPGX(mockDB)

	survey, completed, completedAt, err := repo.LoadSurvey(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, survey)
	assert.False(t, completed)
	assert.Nil(t, completedAt)
}

type mockRowForSurveyData struct {
	jsonData []byte
	completed bool
	completedAt *time.Time
	scanErr error
}

func (m *mockRowForSurveyData) Scan(dest ...interface{}) error {
	if m.scanErr != nil {
		return m.scanErr
	}
	if len(dest) >= 3 {
		*dest[0].(*[]byte) = m.jsonData
		*dest[1].(*bool) = m.completed
		if m.completedAt != nil {
			*dest[2].(**time.Time) = m.completedAt
		}
	}
	return nil
}

func TestSurveyRepositoryPGX_LoadSurvey_NoRows(t *testing.T) {
	ctx := context.Background()
	mockDB := &mockDBForSurvey{
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			return &mockRowForSurvey{scanErr: pgx.ErrNoRows}
		},
	}
	repo := NewSurveyRepositoryPGX(mockDB)

	survey, completed, completedAt, err := repo.LoadSurvey(ctx, "user-1")
	require.NoError(t, err)
	assert.NotNil(t, survey)
	assert.False(t, completed)
	assert.Nil(t, completedAt)
}

func TestSurveyRepositoryPGX_LoadSurvey_EmptyJSON(t *testing.T) {
	ctx := context.Background()
	mockDB := &mockDBForSurvey{
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			return &mockRowForSurveyData{jsonData: []byte{}}
		},
	}
	repo := NewSurveyRepositoryPGX(mockDB)

	survey, completed, completedAt, err := repo.LoadSurvey(ctx, "user-1")
	require.NoError(t, err)
	assert.NotNil(t, survey)
	assert.False(t, completed)
	assert.Nil(t, completedAt)
}

func TestSurveyRepositoryPGX_LoadSurvey_UnmarshalError(t *testing.T) {
	ctx := context.Background()
	mockDB := &mockDBForSurvey{
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			return &mockRowForSurveyData{jsonData: []byte("not-json")}
		},
	}
	repo := NewSurveyRepositoryPGX(mockDB)

	survey, completed, completedAt, err := repo.LoadSurvey(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, survey)
	assert.False(t, completed)
	assert.Nil(t, completedAt)
}

func TestSurveyRepositoryPGX_LoadSurvey_Success(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	mockDB := &mockDBForSurvey{
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
			return &mockRowForSurveyData{jsonData: []byte(`{"q1":"a"}`), completed: true, completedAt: &now}
		},
	}
	repo := NewSurveyRepositoryPGX(mockDB)

	survey, completed, completedAt, err := repo.LoadSurvey(ctx, "user-1")
	require.NoError(t, err)
	assert.NotNil(t, survey)
	assert.True(t, completed)
	assert.NotNil(t, completedAt)
}

func TestSurveyRepositoryPGX_SaveSurvey_MarshalError(t *testing.T) {
	ctx := context.Background()
	mockDB := &mockDBForSurvey{}
	repo := NewSurveyRepositoryPGX(mockDB)

	// channels cannot be marshaled to JSON
	err := repo.SaveSurvey(ctx, "user-1", map[string]interface{}{"bad": make(chan int)}, false, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal survey")
}
