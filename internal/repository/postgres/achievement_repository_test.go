package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/apperrors"
)

func setupAchievementRepo(t *testing.T) (*achievementRepositoryEx, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return NewAchievementRepositoryEx(db).(*achievementRepositoryEx), mock
}

func TestAchievementRepositoryEx_ListWithEarnedStatus_Success(t *testing.T) {
	repo, mock := setupAchievementRepo(t)
	ctx := context.Background()
	userID := "user-1"
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "name", "description", "icon_url", "earned_at", "created_at"}).
		AddRow("ach-1", "First Steps", "Complete your first workout", "icon1", now, now).
		AddRow("ach-2", "Early Bird", "Workout before 7am", "icon2", nil, now)

	mock.ExpectQuery("SELECT a.id").
		WithArgs(userID).
		WillReturnRows(rows)

	result, err := repo.ListWithEarnedStatus(ctx, userID)
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "ach-1", result[0].ID)
	assert.Equal(t, "First Steps", result[0].Name)
	assert.NotNil(t, result[0].EarnedAt)
	assert.Equal(t, "ach-2", result[1].ID)
	assert.Nil(t, result[1].EarnedAt)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAchievementRepositoryEx_ListWithEarnedStatus_QueryError(t *testing.T) {
	repo, mock := setupAchievementRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT a.id").
		WillReturnError(assert.AnError)

	result, err := repo.ListWithEarnedStatus(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAchievementRepositoryEx_ListWithEarnedStatus_ScanError(t *testing.T) {
	repo, mock := setupAchievementRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "name", "description", "icon_url", "earned_at", "created_at"}).
		AddRow("ach-1", nil, nil, nil, nil, nil)

	mock.ExpectQuery("SELECT a.id").
		WillReturnRows(rows)

	result, err := repo.ListWithEarnedStatus(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAchievementRepositoryEx_ListWithEarnedStatus_RowsError(t *testing.T) {
	repo, mock := setupAchievementRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "name", "description", "icon_url", "earned_at", "created_at"}).
		AddRow("ach-1", "Test", "Desc", "icon", nil, time.Now())
	rows.CloseError(assert.AnError)

	mock.ExpectQuery("SELECT a.id").
		WillReturnRows(rows)

	result, err := repo.ListWithEarnedStatus(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAchievementRepositoryEx_Earn_Success(t *testing.T) {
	repo, mock := setupAchievementRepo(t)
	ctx := context.Background()

	mock.ExpectExec("INSERT INTO user_achievements").
		WithArgs("user-1", "ach-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Earn(ctx, "user-1", "ach-1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAchievementRepositoryEx_Earn_Error(t *testing.T) {
	repo, mock := setupAchievementRepo(t)
	ctx := context.Background()

	mock.ExpectExec("INSERT INTO user_achievements").
		WithArgs("user-1", "ach-1").
		WillReturnError(assert.AnError)

	err := repo.Earn(ctx, "user-1", "ach-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}
