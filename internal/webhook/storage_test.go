package webhook

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockDBForSources struct {
	sources []SourceInfo
}

func (m *mockDBForSources) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	return &mockTx{}, nil
}

func (m *mockDBForSources) ExecContext(ctx context.Context, query string, args ...interface{}) (Rower, error) {
	return &mockRower{}, nil
}

func (m *mockDBForSources) GetSources(ctx context.Context, userID string) ([]SourceInfo, error) {
	return m.sources, nil
}

func (m *mockDBForSources) DeleteBySource(ctx context.Context, userID, source string) (int64, error) {
	return 1, nil
}

func TestStorageGetSources(t *testing.T) {
	t.Run("returns sources", func(t *testing.T) {
		now := time.Now()
		db := &mockDBForSources{
			sources: []SourceInfo{
				{Source: "open_wearables", SourceName: "Open Wearables", ConnectedAt: now},
			},
		}
		store := NewStorage(db, zap.NewNop())
		sources, err := store.GetSources(context.Background(), "user-1")
		assert.NoError(t, err)
		assert.Len(t, sources, 1)
		assert.Equal(t, "open_wearables", sources[0].Source)
	})
}

func TestStorageDeleteBySource(t *testing.T) {
	t.Run("deletes records", func(t *testing.T) {
		db := &mockDBForSources{}
		store := NewStorage(db, zap.NewNop())
		count, err := store.DeleteBySource(context.Background(), "user-1", "open_wearables")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})
}

type mockRower struct{}

func (m *mockRower) RowsAffected() (int64, error) {
	return 1, nil
}

type fakeWebhookDBForStorageTest struct {
	beginTxErr    error
	tx            Tx
	execErr       error
	sources       []SourceInfo
	getSourcesErr error
	deleteCount   int64
	deleteErr     error
	beginCalled   bool
	execQueries   []string
}

func (f *fakeWebhookDBForStorageTest) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	f.beginCalled = true
	if f.beginTxErr != nil {
		return nil, f.beginTxErr
	}
	if f.tx != nil {
		return f.tx, nil
	}
	return &fakeWebhookTxForStorageTest{}, nil
}

func (f *fakeWebhookDBForStorageTest) ExecContext(ctx context.Context, query string, args ...interface{}) (Rower, error) {
	f.execQueries = append(f.execQueries, query)
	if f.execErr != nil {
		return nil, f.execErr
	}
	return &fakeWebhookResultForStorageTest{}, nil
}

func (f *fakeWebhookDBForStorageTest) GetSources(ctx context.Context, userID string) ([]SourceInfo, error) {
	if f.getSourcesErr != nil {
		return nil, f.getSourcesErr
	}
	return f.sources, nil
}

func (f *fakeWebhookDBForStorageTest) DeleteBySource(ctx context.Context, userID, source string) (int64, error) {
	if f.deleteErr != nil {
		return 0, f.deleteErr
	}
	return f.deleteCount, nil
}

type fakeWebhookResultForStorageTest struct{}

func (f *fakeWebhookResultForStorageTest) RowsAffected() (int64, error) {
	return 3, nil
}

type fakeWebhookTxForStorageTest struct {
	execErr     error
	commitErr   error
	rollbackErr error
	rowErr      error
	execCalls   int
	execArgs    [][]interface{}
}

func (f *fakeWebhookTxForStorageTest) ExecContext(ctx context.Context, query string, args ...interface{}) (Rower, error) {
	f.execCalls++
	f.execArgs = append(f.execArgs, args)
	if f.execErr != nil {
		return nil, f.execErr
	}
	return &fakeWebhookResultForStorageTest{}, nil
}

func (f *fakeWebhookTxForStorageTest) QueryRowContext(ctx context.Context, query string, args ...interface{}) Scanner {
	if f.rowErr != nil {
		return fakeWebhookRowForStorageTest{err: f.rowErr}
	}
	return sqlRowNoRows{}
}

func (f *fakeWebhookTxForStorageTest) Commit() error {
	return f.commitErr
}

func (f *fakeWebhookTxForStorageTest) Rollback() error {
	return f.rollbackErr
}

type fakeWebhookRowForStorageTest struct {
	err error
}

func (r fakeWebhookRowForStorageTest) Scan(dest ...interface{}) error {
	return r.err
}

func newWebhookPayloadForStorageTest() *OpenWearablesWebhookPayload {
	return &OpenWearablesWebhookPayload{
		UserID:    "user-1",
		Source:    SourceTypeOpenWearables,
		Timestamp: time.Now().UTC(),
		Metrics: []OpenWearablesMetric{
			{Type: MetricTypeHeartRate, Value: 72.0},
		},
	}
}

func TestStorageSaveMetricsEmptyMetrics(t *testing.T) {
	t.Run("empty payload is a no-op", func(t *testing.T) {
		db := &fakeWebhookDBForStorageTest{}
		store := NewStorage(db, zap.NewNop())

		require.NoError(t, store.SaveMetrics(context.Background(), &OpenWearablesWebhookPayload{}))
		assert.False(t, db.beginCalled)
		assert.Empty(t, db.execQueries)
	})
}

func TestStorageSaveMetricsSuccessPassesMetricArguments(t *testing.T) {
	tx := &fakeWebhookTxForStorageTest{}
	db := &fakeWebhookDBForStorageTest{tx: tx}
	store := NewStorage(db, zap.NewNop())

	payload := newWebhookPayloadForStorageTest()
	metricTimestamp := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	payload.Metrics = append(payload.Metrics, OpenWearablesMetric{
		Type:      MetricTypeSteps,
		Value:     1200,
		Timestamp: metricTimestamp,
	})

	require.NoError(t, store.SaveMetrics(context.Background(), payload))
	assert.True(t, db.beginCalled)
	assert.Equal(t, 2, tx.execCalls)
	require.Len(t, tx.execArgs, 2)

	assert.Equal(t, payload.UserID, tx.execArgs[0][0])
	assert.Equal(t, MetricTypeHeartRate, tx.execArgs[0][1])
	assert.InDelta(t, 72.0, tx.execArgs[0][2], 0.0001)
	assert.Equal(t, payload.Timestamp, tx.execArgs[0][3])
	assert.Equal(t, string(SourceTypeOpenWearables), tx.execArgs[0][4])
	assert.Equal(t, string(SourceTypeOpenWearables), tx.execArgs[0][5])

	assert.Equal(t, metricTimestamp, tx.execArgs[1][3])
}

func TestStorageSaveMetricsBeginTxError(t *testing.T) {
	db := &fakeWebhookDBForStorageTest{beginTxErr: errors.New("begin failed")}
	store := NewStorage(db, zap.NewNop())

	err := store.SaveMetrics(context.Background(), newWebhookPayloadForStorageTest())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "begin transaction")
	assert.Contains(t, err.Error(), "begin failed")
}

func TestStorageSaveMetricsExecError(t *testing.T) {
	tx := &fakeWebhookTxForStorageTest{execErr: errors.New("insert failed")}
	db := &fakeWebhookDBForStorageTest{tx: tx}
	store := NewStorage(db, zap.NewNop())

	err := store.SaveMetrics(context.Background(), newWebhookPayloadForStorageTest())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "save metric heart_rate for user user-1")
	assert.Contains(t, err.Error(), "insert metric")
}

func TestStorageSaveMetricsCommitError(t *testing.T) {
	tx := &fakeWebhookTxForStorageTest{commitErr: errors.New("commit failed")}
	db := &fakeWebhookDBForStorageTest{tx: tx}
	store := NewStorage(db, zap.NewNop())

	err := store.SaveMetrics(context.Background(), newWebhookPayloadForStorageTest())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "commit transaction")
}

func TestStorageSaveMetricsRollbackFailureIsTolerated(t *testing.T) {
	t.Run("rollback error after commit does not fail the call", func(t *testing.T) {
		tx := &fakeWebhookTxForStorageTest{rollbackErr: errors.New("rollback failed")}
		db := &fakeWebhookDBForStorageTest{tx: tx}
		store := NewStorage(db, zap.NewNop())

		require.NoError(t, store.SaveMetrics(context.Background(), newWebhookPayloadForStorageTest()))
	})

	t.Run("tx done error after commit does not fail the call", func(t *testing.T) {
		tx := &fakeWebhookTxForStorageTest{rollbackErr: sql.ErrTxDone}
		db := &fakeWebhookDBForStorageTest{tx: tx}
		store := NewStorage(db, zap.NewNop())

		require.NoError(t, store.SaveMetrics(context.Background(), newWebhookPayloadForStorageTest()))
	})

	t.Run("rollback error after failed metric is tolerated", func(t *testing.T) {
		tx := &fakeWebhookTxForStorageTest{
			execErr:     errors.New("insert failed"),
			rollbackErr: errors.New("rollback failed"),
		}
		db := &fakeWebhookDBForStorageTest{tx: tx}
		store := NewStorage(db, zap.NewNop())

		require.Error(t, store.SaveMetrics(context.Background(), newWebhookPayloadForStorageTest()))
	})
}

func TestStorageGetSourcesError(t *testing.T) {
	db := &fakeWebhookDBForStorageTest{getSourcesErr: errors.New("sources unavailable")}
	store := NewStorage(db, zap.NewNop())

	sources, err := store.GetSources(context.Background(), "user-1")
	require.Error(t, err)
	assert.Nil(t, sources)
}

func TestStorageDeleteBySourceError(t *testing.T) {
	db := &fakeWebhookDBForStorageTest{deleteErr: errors.New("delete unavailable")}
	store := NewStorage(db, zap.NewNop())

	count, err := store.DeleteBySource(context.Background(), "user-1", string(SourceTypeOpenWearables))
	require.Error(t, err)
	assert.Zero(t, count)
}

func TestStorageCheckAndSaveNonceBeginTxError(t *testing.T) {
	db := &fakeWebhookDBForStorageTest{beginTxErr: errors.New("begin failed")}
	store := NewStorage(db, zap.NewNop())

	err := store.CheckAndSaveNonce(context.Background(), "user-1", "nonce-1", time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "begin transaction")
}

func TestStorageCheckAndSaveNonceUnexpectedScanError(t *testing.T) {
	tx := &fakeWebhookTxForStorageTest{rowErr: errors.New("scan exploded")}
	db := &fakeWebhookDBForStorageTest{tx: tx}
	store := NewStorage(db, zap.NewNop())

	err := store.CheckAndSaveNonce(context.Background(), "user-1", "nonce-1", time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check nonce")
	assert.Zero(t, tx.execCalls)
}

func TestStorageCheckAndSaveNonceInsertError(t *testing.T) {
	tx := &fakeWebhookTxForStorageTest{execErr: errors.New("insert failed")}
	db := &fakeWebhookDBForStorageTest{tx: tx}
	store := NewStorage(db, zap.NewNop())

	err := store.CheckAndSaveNonce(context.Background(), "user-1", "nonce-1", time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insert nonce")
	require.Len(t, tx.execArgs, 1)
	assert.Equal(t, "user-1", tx.execArgs[0][0])
	assert.Equal(t, "nonce-1", tx.execArgs[0][1])
}

func TestStorageCheckAndSaveNonceCommitError(t *testing.T) {
	tx := &fakeWebhookTxForStorageTest{commitErr: errors.New("commit failed")}
	db := &fakeWebhookDBForStorageTest{tx: tx}
	store := NewStorage(db, zap.NewNop())

	err := store.CheckAndSaveNonce(context.Background(), "user-1", "nonce-1", time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "commit nonce transaction")
}

func TestStorageCheckAndSaveNonceAlreadyUsed(t *testing.T) {
	usedAt := time.Now().UTC().Truncate(time.Second)
	tx := &fakeWebhookTxWithNonceForStorageTest{createdAt: usedAt}
	db := &fakeWebhookDBForStorageTest{tx: tx}
	store := NewStorage(db, zap.NewNop())

	err := store.CheckAndSaveNonce(context.Background(), "user-1", "nonce-1", usedAt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonce already used at")
	assert.Zero(t, tx.execCalls)
}

func TestStorageCheckAndSaveNonceRollbackFailureIsTolerated(t *testing.T) {
	tx := &fakeWebhookTxForStorageTest{
		execErr:     errors.New("insert failed"),
		rollbackErr: errors.New("rollback failed"),
	}
	db := &fakeWebhookDBForStorageTest{tx: tx}
	store := NewStorage(db, zap.NewNop())

	require.Error(t, store.CheckAndSaveNonce(context.Background(), "user-1", "nonce-1", time.Now()))
}

type fakeWebhookTxWithNonceForStorageTest struct {
	createdAt time.Time
	execCalls int
}

func (f *fakeWebhookTxWithNonceForStorageTest) ExecContext(ctx context.Context, query string, args ...interface{}) (Rower, error) {
	f.execCalls++
	return &fakeWebhookResultForStorageTest{}, nil
}

func (f *fakeWebhookTxWithNonceForStorageTest) QueryRowContext(ctx context.Context, query string, args ...interface{}) Scanner {
	return sqlRowWithTime{time: f.createdAt}
}

func (f *fakeWebhookTxWithNonceForStorageTest) Commit() error {
	return nil
}

func (f *fakeWebhookTxWithNonceForStorageTest) Rollback() error {
	return nil
}

func TestStoragePurgeExpiredNonces(t *testing.T) {
	t.Run("purges expired nonces", func(t *testing.T) {
		db := &fakeWebhookDBForStorageTest{}
		store := NewStorage(db, zap.NewNop())

		require.NoError(t, store.PurgeExpiredNonces(context.Background()))
		require.Len(t, db.execQueries, 1)
		assert.Equal(t, deleteExpiredWebhookNoncesQuery, db.execQueries[0])
	})

	t.Run("propagates database errors", func(t *testing.T) {
		db := &fakeWebhookDBForStorageTest{execErr: errors.New("purge failed")}
		store := NewStorage(db, zap.NewNop())

		err := store.PurgeExpiredNonces(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "purge expired nonces")
	})
}

func newSQLMockForWebhookStorageTest(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func TestSQLDBAdapterBeginTx(t *testing.T) {
	t.Run("starts a transaction", func(t *testing.T) {
		db, mock := newSQLMockForWebhookStorageTest(t)
		mock.ExpectBegin()

		adapter := NewSQLDBAdapter(db)
		tx, err := adapter.BeginTx(context.Background(), nil)
		require.NoError(t, err)
		require.NotNil(t, tx)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates begin errors", func(t *testing.T) {
		db, mock := newSQLMockForWebhookStorageTest(t)
		mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

		adapter := NewSQLDBAdapter(db)
		tx, err := adapter.BeginTx(context.Background(), nil)
		require.Error(t, err)
		assert.Nil(t, tx)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSQLDBAdapterExecContext(t *testing.T) {
	t.Run("returns a result adapter", func(t *testing.T) {
		db, mock := newSQLMockForWebhookStorageTest(t)
		mock.ExpectExec(deleteExpiredWebhookNoncesQuery).WillReturnResult(sqlmock.NewResult(0, 5))

		adapter := NewSQLDBAdapter(db)
		rower, err := adapter.ExecContext(context.Background(), deleteExpiredWebhookNoncesQuery)
		require.NoError(t, err)
		require.NotNil(t, rower)
		affected, err := rower.RowsAffected()
		require.NoError(t, err)
		assert.Equal(t, int64(5), affected)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates exec errors", func(t *testing.T) {
		db, mock := newSQLMockForWebhookStorageTest(t)
		mock.ExpectExec(deleteExpiredWebhookNoncesQuery).WillReturnError(errors.New("exec failed"))

		adapter := NewSQLDBAdapter(db)
		rower, err := adapter.ExecContext(context.Background(), deleteExpiredWebhookNoncesQuery)
		require.Error(t, err)
		assert.Nil(t, rower)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSQLDBAdapterGetSources(t *testing.T) {
	t.Run("returns connected sources", func(t *testing.T) {
		db, mock := newSQLMockForWebhookStorageTest(t)
		now := time.Now().UTC().Truncate(time.Second)
		mock.ExpectQuery(getSourcesQuery).WithArgs("user-1").WillReturnRows(
			sqlmock.NewRows([]string{"source", "connected_at"}).
				AddRow("garmin", now).
				AddRow("apple_health", now),
		)

		adapter := NewSQLDBAdapter(db)
		sources, err := adapter.GetSources(context.Background(), "user-1")
		require.NoError(t, err)
		require.Len(t, sources, 2)
		assert.Equal(t, "garmin", sources[0].Source)
		assert.Equal(t, "garmin", sources[0].SourceName)
		assert.True(t, sources[0].ConnectedAt.Equal(now))
		assert.Equal(t, "apple_health", sources[1].Source)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty list when no rows", func(t *testing.T) {
		db, mock := newSQLMockForWebhookStorageTest(t)
		mock.ExpectQuery(getSourcesQuery).WithArgs("user-1").WillReturnRows(
			sqlmock.NewRows([]string{"source", "connected_at"}),
		)

		adapter := NewSQLDBAdapter(db)
		sources, err := adapter.GetSources(context.Background(), "user-1")
		require.NoError(t, err)
		assert.Empty(t, sources)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates query errors", func(t *testing.T) {
		db, mock := newSQLMockForWebhookStorageTest(t)
		mock.ExpectQuery(getSourcesQuery).WithArgs("user-1").WillReturnError(errors.New("query failed"))

		adapter := NewSQLDBAdapter(db)
		sources, err := adapter.GetSources(context.Background(), "user-1")
		require.Error(t, err)
		assert.Nil(t, sources)
		assert.Contains(t, err.Error(), "query sources")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates scan errors", func(t *testing.T) {
		db, mock := newSQLMockForWebhookStorageTest(t)
		mock.ExpectQuery(getSourcesQuery).WithArgs("user-1").WillReturnRows(
			sqlmock.NewRows([]string{"source", "connected_at"}).AddRow("garmin", "not-a-timestamp"),
		)

		adapter := NewSQLDBAdapter(db)
		sources, err := adapter.GetSources(context.Background(), "user-1")
		require.Error(t, err)
		assert.Nil(t, sources)
		assert.Contains(t, err.Error(), "scan source")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates iteration errors", func(t *testing.T) {
		db, mock := newSQLMockForWebhookStorageTest(t)
		mock.ExpectQuery(getSourcesQuery).WithArgs("user-1").WillReturnRows(
			sqlmock.NewRows([]string{"source", "connected_at"}).
				AddRow("garmin", time.Now()).
				RowError(0, errors.New("iteration failed")),
		)

		adapter := NewSQLDBAdapter(db)
		sources, err := adapter.GetSources(context.Background(), "user-1")
		require.Error(t, err)
		assert.Nil(t, sources)
		assert.Contains(t, err.Error(), "iterate sources")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("surfaces close errors from iteration", func(t *testing.T) {
		db, mock := newSQLMockForWebhookStorageTest(t)
		mock.ExpectQuery(getSourcesQuery).WithArgs("user-1").WillReturnRows(
			sqlmock.NewRows([]string{"source", "connected_at"}).
				AddRow("garmin", time.Now()).
				CloseError(errors.New("close failed")),
		)

		adapter := NewSQLDBAdapter(db)
		sources, err := adapter.GetSources(context.Background(), "user-1")
		require.Error(t, err)
		assert.Nil(t, sources)
		assert.Contains(t, err.Error(), "close failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSQLDBAdapterDeleteBySource(t *testing.T) {
	t.Run("returns deleted row count", func(t *testing.T) {
		db, mock := newSQLMockForWebhookStorageTest(t)
		mock.ExpectExec(deleteBySourceQuery).
			WithArgs("user-1", "garmin").
			WillReturnResult(sqlmock.NewResult(0, 7))

		adapter := NewSQLDBAdapter(db)
		count, err := adapter.DeleteBySource(context.Background(), "user-1", "garmin")
		require.NoError(t, err)
		assert.Equal(t, int64(7), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates exec errors", func(t *testing.T) {
		db, mock := newSQLMockForWebhookStorageTest(t)
		mock.ExpectExec(deleteBySourceQuery).WithArgs("user-1", "garmin").
			WillReturnError(errors.New("delete failed"))

		adapter := NewSQLDBAdapter(db)
		count, err := adapter.DeleteBySource(context.Background(), "user-1", "garmin")
		require.Error(t, err)
		assert.Zero(t, count)
		assert.Contains(t, err.Error(), "delete by source")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates rows affected errors", func(t *testing.T) {
		db, mock := newSQLMockForWebhookStorageTest(t)
		mock.ExpectExec(deleteBySourceQuery).WithArgs("user-1", "garmin").
			WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected failed")))

		adapter := NewSQLDBAdapter(db)
		count, err := adapter.DeleteBySource(context.Background(), "user-1", "garmin")
		require.Error(t, err)
		assert.Zero(t, count)
		assert.Contains(t, err.Error(), "rows affected")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSQLTxAdapterExecAndQueryRow(t *testing.T) {
	db, mock := newSQLMockForWebhookStorageTest(t)
	createdAt := time.Now().UTC().Truncate(time.Second)
	mock.ExpectBegin()
	mock.ExpectExec(insertBiometricMetricQuery).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(checkWebhookNonceQuery).WithArgs("user-1", "nonce-1").
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(createdAt))

	adapter := NewSQLDBAdapter(db)
	tx, err := adapter.BeginTx(context.Background(), nil)
	require.NoError(t, err)

	rower, err := tx.ExecContext(context.Background(), insertBiometricMetricQuery,
		"user-1", MetricTypeHeartRate, 72.0, createdAt, "open_wearables", "open_wearables")
	require.NoError(t, err)
	require.NotNil(t, rower)

	scanner := tx.QueryRowContext(context.Background(), checkWebhookNonceQuery, "user-1", "nonce-1")
	require.NotNil(t, scanner)

	var scanned time.Time
	require.NoError(t, scanner.Scan(&scanned))
	assert.True(t, scanned.Equal(createdAt))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLTxAdapterCommitAndRollback(t *testing.T) {
	t.Run("commit then rollback reports tx done", func(t *testing.T) {
		db, mock := newSQLMockForWebhookStorageTest(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		adapter := NewSQLDBAdapter(db)
		tx, err := adapter.BeginTx(context.Background(), nil)
		require.NoError(t, err)

		require.NoError(t, tx.Commit())
		assert.ErrorIs(t, tx.Rollback(), sql.ErrTxDone)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rollback succeeds on open transaction", func(t *testing.T) {
		db, mock := newSQLMockForWebhookStorageTest(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		adapter := NewSQLDBAdapter(db)
		tx, err := adapter.BeginTx(context.Background(), nil)
		require.NoError(t, err)

		require.NoError(t, tx.Rollback())
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSQLResultAdapterRowsAffected(t *testing.T) {
	t.Run("returns rows affected", func(t *testing.T) {
		adapter := NewSQLResultAdapter(sqlmock.NewResult(1, 4))
		count, err := adapter.RowsAffected()
		require.NoError(t, err)
		assert.Equal(t, int64(4), count)
	})

	t.Run("propagates errors", func(t *testing.T) {
		adapter := NewSQLResultAdapter(sqlmock.NewErrorResult(errors.New("boom")))
		_, err := adapter.RowsAffected()
		require.Error(t, err)
	})
}
