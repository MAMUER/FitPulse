package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/port"
)

func setupProfileRepo(t *testing.T) (*profileRepository, sqlmock.Sqlmock) {

	t.Helper()

	db, mock, err := sqlmock.New()

	require.NoError(t, err)

	return &profileRepository{db: db}, mock

}

func TestProfileRepository_GetProfile_Success(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-1", "test@example.com", "hash", "Test User", "user", true, now, now)

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.GetProfile(ctx, "user-1")

	require.NoError(t, err)

	assert.Equal(t, "user-1", result.ID)

	assert.Equal(t, "Test User", result.FullName)

	assert.True(t, result.EmailVerified)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_GetProfile_NotFound(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1").
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetProfile(ctx, "user-1")

	require.Error(t, err)

	assert.Nil(t, result)

	assert.True(t, apperrors.IsNotFound(err))

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_GetProfile_QueryError(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT id").WillReturnError(assert.AnError)

	result, err := repo.GetProfile(ctx, "user-1")

	require.Error(t, err)

	assert.Nil(t, result)

	assert.True(t, apperrors.Code(err) == "INTERNAL")

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_UpdateProfile_Success(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users").
		WithArgs("Test User", sqlmock.AnyArg(), "user-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdateProfile(ctx, "user-1", "Test User", nil, nil, "balanced", 7.5)

	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_UpdateProfile_Error(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users").WillReturnError(assert.AnError)

	err := repo.UpdateProfile(ctx, "user-1", "Test User", nil, nil, "balanced", 7.5)

	require.Error(t, err)

	assert.True(t, apperrors.Code(err) == "INTERNAL")

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_UserExists_True(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("user-1").
		WillReturnRows(rows)

	exists, err := repo.UserExists(ctx, "user-1")

	require.NoError(t, err)

	assert.True(t, exists)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_UserExists_False(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("user-1").
		WillReturnRows(rows)

	exists, err := repo.UserExists(ctx, "user-1")

	require.NoError(t, err)

	assert.False(t, exists)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_UserExists_Error(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT EXISTS").WillReturnError(assert.AnError)

	exists, err := repo.UserExists(ctx, "user-1")

	require.Error(t, err)

	assert.False(t, exists)

	assert.True(t, apperrors.Code(err) == "INTERNAL")

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_CreateProfile_Success(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	mock.ExpectExec("INSERT INTO user_profiles").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateProfile(ctx, "user-1")

	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_CreateProfile_Error(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	mock.ExpectExec("INSERT INTO user_profiles").WillReturnError(assert.AnError)

	err := repo.CreateProfile(ctx, "user-1")

	require.Error(t, err)

	assert.True(t, apperrors.Code(err) == "INTERNAL")

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_UpsertProfile_Success(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	data := &port.ProfileData{

		Age: 30,

		Gender: "male",

		HeightCm: 180,

		WeightKg: 75.0,

		FitnessLevel: "intermediate",

		Nutrition: "balanced",

		SleepHours: 7.5,
	}

	mock.ExpectExec("INSERT INTO user_profiles").
		WithArgs("user-1", int32(30), "male", int32(180), 75.0, "intermediate", "balanced", float32(7.5)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpsertProfile(ctx, "user-1", data)

	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_UpsertProfile_Error(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	mock.ExpectExec("INSERT INTO user_profiles").WillReturnError(assert.AnError)

	err := repo.UpsertProfile(ctx, "user-1", &port.ProfileData{})

	require.Error(t, err)

	assert.True(t, apperrors.Code(err) == "INTERNAL")

	require.NoError(t, mock.ExpectationsWereMet())

}

func setupProfileRepoDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {

	t.Helper()

	db, mock, err := sqlmock.New()

	require.NoError(t, err)

	return db, mock

}

func TestProfileRepository_NewProfileRepository(t *testing.T) {

	db, mock := setupProfileRepoDB(t)

	repo := NewProfileRepository(db)

	require.NotNil(t, repo)

	profileRepo, ok := repo.(*profileRepository)

	require.True(t, ok)

	assert.Same(t, db, profileRepo.db)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_SaveSurvey_Success(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	survey := map[string]interface{}{"fitness_level": "advanced"}

	surveyJSON, err := json.Marshal(survey)

	require.NoError(t, err)

	mock.ExpectExec("UPDATE user_profiles").
		WithArgs(surveyJSON, true, nil, "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.SaveSurvey(ctx, "user-1", survey, true, nil)

	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_SaveSurvey_WithCompletedAt(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	completedAt := time.Now()

	survey := map[string]interface{}{}

	surveyJSON, err := json.Marshal(survey)

	require.NoError(t, err)

	mock.ExpectExec("UPDATE user_profiles").
		WithArgs(surveyJSON, false, completedAt, "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.SaveSurvey(ctx, "user-1", survey, false, &completedAt)

	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_SaveSurvey_MarshalError(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	survey := map[string]interface{}{"invalid": make(chan int)}

	err := repo.SaveSurvey(ctx, "user-1", survey, true, nil)

	require.Error(t, err)

	assert.True(t, apperrors.Code(err) == "INTERNAL")

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_SaveSurvey_Error(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE user_profiles").WillReturnError(assert.AnError)

	err := repo.SaveSurvey(ctx, "user-1", map[string]interface{}{}, true, nil)

	require.Error(t, err)

	assert.True(t, apperrors.Code(err) == "INTERNAL")

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_LoadSurvey_WithData(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	completedAt := time.Now().UTC().Truncate(time.Second)

	rows := sqlmock.NewRows([]string{"survey_data", "survey_completed", "survey_completed_at"}).
		AddRow([]byte(`{"fitness_level":"advanced"}`), true, completedAt)

	mock.ExpectQuery("SELECT survey_data").
		WithArgs("user-1").
		WillReturnRows(rows)

	survey, completed, loadedAt, err := repo.LoadSurvey(ctx, "user-1")

	require.NoError(t, err)

	assert.Equal(t, "advanced", survey["fitness_level"])

	assert.True(t, completed)

	require.NotNil(t, loadedAt)

	assert.Equal(t, completedAt, *loadedAt)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_LoadSurvey_EmptySurveyData(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"survey_data", "survey_completed", "survey_completed_at"}).
		AddRow([]byte{}, false, nil)

	mock.ExpectQuery("SELECT survey_data").
		WithArgs("user-1").
		WillReturnRows(rows)

	survey, completed, loadedAt, err := repo.LoadSurvey(ctx, "user-1")

	require.NoError(t, err)

	assert.Empty(t, survey)

	assert.False(t, completed)

	assert.Nil(t, loadedAt)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_LoadSurvey_NilSurveyData(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"survey_data", "survey_completed", "survey_completed_at"}).
		AddRow(nil, false, nil)

	mock.ExpectQuery("SELECT survey_data").
		WithArgs("user-1").
		WillReturnRows(rows)

	survey, completed, loadedAt, err := repo.LoadSurvey(ctx, "user-1")

	require.NoError(t, err)

	assert.Empty(t, survey)

	assert.False(t, completed)

	assert.Nil(t, loadedAt)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_LoadSurvey_NoRows(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT survey_data").
		WithArgs("user-1").
		WillReturnError(sql.ErrNoRows)

	survey, completed, loadedAt, err := repo.LoadSurvey(ctx, "user-1")

	require.NoError(t, err)

	assert.Empty(t, survey)

	assert.False(t, completed)

	assert.Nil(t, loadedAt)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_LoadSurvey_QueryError(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT survey_data").WillReturnError(assert.AnError)

	survey, completed, loadedAt, err := repo.LoadSurvey(ctx, "user-1")

	require.Error(t, err)

	assert.Nil(t, survey)

	assert.False(t, completed)

	assert.Nil(t, loadedAt)

	assert.True(t, apperrors.Code(err) == "INTERNAL")

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_LoadSurvey_ScanError(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"survey_data", "survey_completed", "survey_completed_at"}).
		AddRow([]byte(`{}`), "not-a-bool", nil)

	mock.ExpectQuery("SELECT survey_data").
		WithArgs("user-1").
		WillReturnRows(rows)

	survey, completed, loadedAt, err := repo.LoadSurvey(ctx, "user-1")

	require.Error(t, err)

	assert.Nil(t, survey)

	assert.False(t, completed)

	assert.Nil(t, loadedAt)

	assert.True(t, apperrors.Code(err) == "INTERNAL")

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestProfileRepository_LoadSurvey_UnmarshalError(t *testing.T) {

	repo, mock := setupProfileRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"survey_data", "survey_completed", "survey_completed_at"}).
		AddRow([]byte(`{"fitness_level":`), true, nil)

	mock.ExpectQuery("SELECT survey_data").
		WithArgs("user-1").
		WillReturnRows(rows)

	survey, completed, loadedAt, err := repo.LoadSurvey(ctx, "user-1")

	require.Error(t, err)

	assert.Nil(t, survey)

	assert.False(t, completed)

	assert.Nil(t, loadedAt)

	assert.True(t, apperrors.Code(err) == "INTERNAL")

	require.NoError(t, mock.ExpectationsWereMet())

}
