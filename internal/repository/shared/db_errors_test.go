package shared

import (
	"database/sql"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dbErrorsFailingResult struct{}

func (dbErrorsFailingResult) LastInsertId() (int64, error) { return 0, nil }
func (dbErrorsFailingResult) RowsAffected() (int64, error) {
	return 0, errors.New("rows affected failed")
}

type dbErrorsResult struct{ n int64 }

func (r dbErrorsResult) LastInsertId() (int64, error) { return 0, nil }
func (r dbErrorsResult) RowsAffected() (int64, error) { return r.n, nil }

func TestLogTxRollback(t *testing.T) {
	t.Run("logs real rollback errors", func(t *testing.T) {
		assert.NotPanics(t, func() {
			LogTxRollback(errors.New("rollback failed"))
		})
	})

	t.Run("ignores committed transactions", func(t *testing.T) {
		assert.NotPanics(t, func() {
			LogTxRollback(sql.ErrTxDone)
		})
	})

	t.Run("ignores wrapped ErrTxDone", func(t *testing.T) {
		assert.NotPanics(t, func() {
			LogTxRollback(errors.Join(sql.ErrTxDone, errors.New("extra")))
		})
	})
}

func TestRowsAffected(t *testing.T) {
	assert.Equal(t, int64(3), RowsAffected(dbErrorsResult{n: 3}))
	assert.Equal(t, int64(0), RowsAffected(dbErrorsFailingResult{}))
}

func TestCloseRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))

	rows, queryErr := db.Query("SELECT 1")
	require.NoError(t, queryErr)

	CloseRows(rows)
	CloseRows(rows)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEmptySurvey(t *testing.T) {
	survey, completed, completedAt, err := EmptySurvey()
	assert.NoError(t, err)
	assert.NotNil(t, survey)
	assert.False(t, completed)
	assert.Nil(t, completedAt)
	assert.Empty(t, survey)
}
