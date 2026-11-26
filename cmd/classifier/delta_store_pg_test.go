package main

import (
	"database/sql"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockStore(t *testing.T) (*PostgresDeltaStore, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewPostgresDeltaStore(db), mock
}

func TestPostgresDeltaStore_GetDelta(t *testing.T) {
	store, mock := newMockStore(t)

	mock.ExpectQuery("SELECT delta FROM classifier_deltas WHERE user_id").
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"delta"}).AddRow([]byte(`[1,2,3,4]`)))

	delta, ok := store.GetDelta("u1")
	assert.True(t, ok)
	assert.Equal(t, []float64{1, 2, 3, 4}, delta)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresDeltaStore_GetDeltaNotFound(t *testing.T) {
	store, mock := newMockStore(t)

	mock.ExpectQuery("SELECT delta FROM classifier_deltas WHERE user_id").
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	delta, ok := store.GetDelta("missing")
	assert.False(t, ok)
	assert.Nil(t, delta)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresDeltaStore_GetDeltaInvalidJSON(t *testing.T) {
	store, mock := newMockStore(t)

	mock.ExpectQuery("SELECT delta FROM classifier_deltas WHERE user_id").
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"delta"}).AddRow([]byte(`not-json`)))

	delta, ok := store.GetDelta("u1")
	assert.False(t, ok)
	assert.Nil(t, delta)
}

func TestPostgresDeltaStore_SetAndReset(t *testing.T) {
	store, mock := newMockStore(t)

	mock.ExpectExec("INSERT INTO classifier_deltas").
		WithArgs("u1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	store.SetDelta("u1", []float64{0.5, 0.25})
	assert.Equal(t, 2, store.DeltaSize())

	mock.ExpectExec("DELETE FROM classifier_deltas").
		WithArgs("u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	store.ResetDelta("u1")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresDeltaStore_GetAllDeltas(t *testing.T) {
	store, mock := newMockStore(t)

	mock.ExpectQuery("SELECT user_id, delta FROM classifier_deltas").
		WillReturnRows(
			sqlmock.NewRows([]string{"user_id", "delta"}).
				AddRow("u1", []byte(`[1,2]`)).
				AddRow("u2", []byte(`bad`)),
		)

	out := store.GetAllDeltas()
	assert.Equal(t, []float64{1, 2}, out["u1"])
	assert.NotContains(t, out, "u2")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresDeltaStore_GetAllDeltasRowError(t *testing.T) {
	store, mock := newMockStore(t)

	mock.ExpectQuery("SELECT user_id, delta FROM classifier_deltas").
		WillReturnError(errors.New("boom"))

	out := store.GetAllDeltas()
	assert.Empty(t, out)
}

func TestPostgresDeltaStore_NilSafe(t *testing.T) {
	var store *PostgresDeltaStore
	delta, ok := store.GetDelta("u1")
	assert.False(t, ok)
	assert.Nil(t, delta)
	store.SetDelta("u1", []float64{1})
	store.ResetDelta("u1")
	assert.Equal(t, 0, store.DeltaSize())
	assert.Empty(t, store.GetAllDeltas())
}

func TestPostgresDeltaStore_EmptyUserID(t *testing.T) {
	store, mock := newMockStore(t)
	store.SetDelta("", []float64{1})
	store.ResetDelta("")
	_, ok := store.GetDelta("")
	assert.False(t, ok)
	assert.NoError(t, mock.ExpectationsWereMet())
}
