package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/port"
)

func setupBodyCompositionRepo(t *testing.T) (*userBodyCompositionRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return NewUserBodyCompositionRepository(db).(*userBodyCompositionRepository), mock
}

func TestUserBodyCompositionRepository_List_FromAndTo(t *testing.T) {
	repo, mock := setupBodyCompositionRepo(t)
	ctx := context.Background()
	from := time.Now().Add(-30 * 24 * time.Hour)
	to := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "recorded_at", "weight_kg", "height_cm", "bmi", "body_fat_percentage", "muscle_mass_percentage", "bone_mass_percentage", "water_percentage", "visceral_fat_rating", "metabolic_age", "source", "created_at"}).
		AddRow("bc-1", "user-1", from, 70.0, 175.0, 22.9, 15.0, 45.0, 3.5, 60.0, 5.0, 30.0, "manual", from)

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1", from, to, 10).
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1", &from, &to, 10)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "bc-1", result[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserBodyCompositionRepository_List_FromOnly(t *testing.T) {
	repo, mock := setupBodyCompositionRepo(t)
	ctx := context.Background()
	from := time.Now().Add(-30 * 24 * time.Hour)

	rows := sqlmock.NewRows([]string{"id", "user_id", "recorded_at", "weight_kg", "height_cm", "bmi", "body_fat_percentage", "muscle_mass_percentage", "bone_mass_percentage", "water_percentage", "visceral_fat_rating", "metabolic_age", "source", "created_at"}).
		AddRow("bc-1", "user-1", from, 70.0, 175.0, 22.9, 15.0, 45.0, 3.5, 60.0, 5.0, 30.0, "manual", from)

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1", from, 10).
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1", &from, nil, 10)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserBodyCompositionRepository_List_ToOnly(t *testing.T) {
	repo, mock := setupBodyCompositionRepo(t)
	ctx := context.Background()
	to := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "recorded_at", "weight_kg", "height_cm", "bmi", "body_fat_percentage", "muscle_mass_percentage", "bone_mass_percentage", "water_percentage", "visceral_fat_rating", "metabolic_age", "source", "created_at"}).
		AddRow("bc-1", "user-1", to, 70.0, 175.0, 22.9, 15.0, 45.0, 3.5, 60.0, 5.0, 30.0, "manual", to)

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1", to, 10).
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1", nil, &to, 10)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserBodyCompositionRepository_List_NoFilters(t *testing.T) {
	repo, mock := setupBodyCompositionRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "recorded_at", "weight_kg", "height_cm", "bmi", "body_fat_percentage", "muscle_mass_percentage", "bone_mass_percentage", "water_percentage", "visceral_fat_rating", "metabolic_age", "source", "created_at"}).
		AddRow("bc-1", "user-1", time.Now(), 70.0, 175.0, 22.9, 15.0, 45.0, 3.5, 60.0, 5.0, 30.0, "manual", time.Now())

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1", 10).
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1", nil, nil, 10)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserBodyCompositionRepository_List_QueryError(t *testing.T) {
	repo, mock := setupBodyCompositionRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id").WillReturnError(assert.AnError)

	result, err := repo.List(ctx, "user-1", nil, nil, 10)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserBodyCompositionRepository_List_ScanError(t *testing.T) {
	repo, mock := setupBodyCompositionRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "recorded_at", "weight_kg", "height_cm", "bmi", "body_fat_percentage", "muscle_mass_percentage", "bone_mass_percentage", "water_percentage", "visceral_fat_rating", "metabolic_age", "source", "created_at"}).
		AddRow("bc-1", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	mock.ExpectQuery("SELECT id").WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1", nil, nil, 10)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserBodyCompositionRepository_List_RowsError(t *testing.T) {
	repo, mock := setupBodyCompositionRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "recorded_at", "weight_kg", "height_cm", "bmi", "body_fat_percentage", "muscle_mass_percentage", "bone_mass_percentage", "water_percentage", "visceral_fat_rating", "metabolic_age", "source", "created_at"}).
		AddRow("bc-1", "user-1", time.Now(), 70.0, 175.0, 22.9, 15.0, 45.0, 3.5, 60.0, 5.0, 30.0, "manual", time.Now())
	rows.CloseError(assert.AnError)

	mock.ExpectQuery("SELECT id").WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1", nil, nil, 10)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserBodyCompositionRepository_Create_Success(t *testing.T) {
	repo, mock := setupBodyCompositionRepo(t)
	ctx := context.Background()
	bc := &port.UserBodyComposition{
		UserID:        "user-1",
		WeightKG:      70.0,
		HeightCM:      175.0,
		BMI:           22.9,
		BodyFatPercentage: ptrFloat64(15.0),
		MuscleMassPercentage: ptrFloat64(45.0),
		BoneMassPercentage: ptrFloat64(3.5),
		WaterPercentage: ptrFloat64(60.0),
		VisceralFatRating: ptrFloat64(5.0),
		MetabolicAge:  ptrFloat64(30.0),
		Source:        "manual",
	}

	rows := sqlmock.NewRows([]string{"id", "recorded_at"}).
		AddRow("bc-1", time.Now())

	mock.ExpectQuery("INSERT INTO user_body_composition").
		WithArgs("user-1", nil, 70.0, 175.0, 22.9, 15.0, 45.0, 3.5, 60.0, 5.0, 30.0, "manual").
		WillReturnRows(rows)

	result, err := repo.Create(ctx, bc)
	require.NoError(t, err)
	assert.Equal(t, "bc-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserBodyCompositionRepository_Create_WithRecordedAt(t *testing.T) {
	repo, mock := setupBodyCompositionRepo(t)
	ctx := context.Background()
	recordedAt := time.Now().Add(-24 * time.Hour)
	bc := &port.UserBodyComposition{
		UserID:        "user-1",
		RecordedAt:    recordedAt,
		WeightKG:      70.0,
		HeightCM:      175.0,
		BMI:           22.9,
		Source:        "manual",
	}

	rows := sqlmock.NewRows([]string{"id", "recorded_at"}).
		AddRow("bc-1", recordedAt)

	mock.ExpectQuery("INSERT INTO user_body_composition").
		WithArgs("user-1", recordedAt, 70.0, 175.0, 22.9, nil, nil, nil, nil, nil, nil, "manual").
		WillReturnRows(rows)

	result, err := repo.Create(ctx, bc)
	require.NoError(t, err)
	assert.Equal(t, "bc-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserBodyCompositionRepository_Create_QueryError(t *testing.T) {
	repo, mock := setupBodyCompositionRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("INSERT INTO user_body_composition").
		WillReturnError(assert.AnError)

	result, err := repo.Create(ctx, &port.UserBodyComposition{UserID: "user-1"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func ptrFloat64(v float64) *float64 {
	return &v
}
