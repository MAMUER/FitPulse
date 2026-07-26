package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/biometric/domain"
)

func TestNewMedicalRepository(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewMedicalRepository(db)
	assert.NotNil(t, repo)
}

func TestGetActiveConstraints(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "user_id", "code", "label", "category", "severity", "custom_text", "impact_on_training", "validated_by", "validated_at", "active", "created_at", "updated_at"}).
		AddRow("mc-1", "user-123", "I10", "Hypertension", "cardiovascular", "moderate", "", `[{"metric":"heart_rate","action":"caution"}]`, "doctor@example.com", now, true, now, now)

	mock.ExpectQuery("SELECT.*FROM medical_constraints").
		WithArgs("user-123").
		WillReturnRows(rows)

	repo := NewMedicalRepository(db)
	constraints, err := repo.GetActiveConstraints(context.Background(), "user-123")

	require.NoError(t, err)
	require.Len(t, constraints, 1)
	assert.Equal(t, "mc-1", constraints[0].ID)
	assert.Equal(t, "I10", constraints[0].Code)
	assert.Equal(t, "Hypertension", constraints[0].Label)
	assert.Len(t, constraints[0].ImpactOnTraining, 1)
	assert.Equal(t, "heart_rate", constraints[0].ImpactOnTraining[0].Metric)
	assert.Equal(t, "caution", constraints[0].ImpactOnTraining[0].Action)
}

func TestGetActiveConstraintsNoResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows([]string{"id", "user_id", "code", "label", "category", "severity", "custom_text", "impact_on_training", "validated_by", "validated_at", "active", "created_at", "updated_at"})

	mock.ExpectQuery("SELECT.*FROM medical_constraints").
		WithArgs("user-456").
		WillReturnRows(rows)

	repo := NewMedicalRepository(db)
	constraints, err := repo.GetActiveConstraints(context.Background(), "user-456")

	require.NoError(t, err)
	assert.Empty(t, constraints)
}

func TestSaveConstraint(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectExec("INSERT INTO medical_constraints").
		WithArgs(
			"mc-123", "user-123", "I10", "Hypertension", "cardiovascular", "moderate", "",
			sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
			true, sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := NewMedicalRepository(db)
	constraint := domain.MedicalConstraint{
		ID:       "mc-123",
		UserID:   "user-123",
		Code:     "I10",
		Label:    "Hypertension",
		Category: "cardiovascular",
		Severity: "moderate",
		ImpactOnTraining: []domain.ImpactRule{
			{Metric: "heart_rate", Action: "caution"},
		},
		Active: true,
	}

	err = repo.SaveConstraint(context.Background(), constraint)
	require.NoError(t, err)
}

func TestSaveConstraintGeneratesID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectExec("INSERT INTO medical_constraints").
		WithArgs(
			sqlmock.AnyArg(), "user-123", "I10", "Hypertension", "cardiovascular", "moderate", "",
			sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
			true, sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := NewMedicalRepository(db)
	constraint := domain.MedicalConstraint{
		UserID:   "user-123",
		Code:     "I10",
		Label:    "Hypertension",
		Category: "cardiovascular",
		Severity: "moderate",
		ImpactOnTraining: []domain.ImpactRule{
			{Metric: "heart_rate", Action: "caution"},
		},
		Active: true,
	}

	err = repo.SaveConstraint(context.Background(), constraint)
	require.NoError(t, err)
}

func TestDeleteConstraint(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectExec("DELETE FROM medical_constraints WHERE id = \\$1").
		WithArgs("mc-123").
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := NewMedicalRepository(db)
	err = repo.DeleteConstraint(context.Background(), "mc-123")
	require.NoError(t, err)
}

func TestGetConstraintByCode(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "user_id", "code", "label", "category", "severity", "custom_text", "impact_on_training", "validated_by", "validated_at", "active", "created_at", "updated_at"}).
		AddRow("mc-1", "user-123", "I10", "Hypertension", "cardiovascular", "moderate", "", `[{"metric":"heart_rate","action":"caution"}]`, "doctor@example.com", now, true, now, now)

	mock.ExpectQuery("SELECT.*FROM medical_constraints").
		WithArgs("I10").
		WillReturnRows(rows)

	repo := NewMedicalRepository(db)
	constraints, err := repo.GetConstraintByCode(context.Background(), "I10")

	require.NoError(t, err)
	require.Len(t, constraints, 1)
	assert.Equal(t, "I10", constraints[0].Code)
	assert.Equal(t, "Hypertension", constraints[0].Label)
}

func TestGetConstraintByCodeNoResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows([]string{"id", "user_id", "code", "label", "category", "severity", "custom_text", "impact_on_training", "validated_by", "validated_at", "active", "created_at", "updated_at"})

	mock.ExpectQuery("SELECT.*FROM medical_constraints").
		WithArgs("Z99").
		WillReturnRows(rows)

	repo := NewMedicalRepository(db)
	constraints, err := repo.GetConstraintByCode(context.Background(), "Z99")

	require.NoError(t, err)
	assert.Empty(t, constraints)
}

func TestGetActiveConstraintsQueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT.*FROM medical_constraints").
		WithArgs("user-123").
		WillReturnError(errors.New("query failed"))

	repo := NewMedicalRepository(db)
	constraints, err := repo.GetActiveConstraints(context.Background(), "user-123")

	assert.Error(t, err)
	assert.Nil(t, constraints)
	assert.Contains(t, err.Error(), "query active constraints")
}

func TestGetActiveConstraintsRowsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows([]string{"id", "user_id", "code", "label", "category", "severity", "custom_text", "impact_on_training", "validated_by", "validated_at", "active", "created_at", "updated_at"}).
		AddRow("mc-1", "user-123", "I10", "Hypertension", "cardiovascular", "moderate", "", `[{"metric":"heart_rate","action":"caution"}]`, "doctor@example.com", time.Now(), true, time.Now(), time.Now())

	mock.ExpectQuery("SELECT.*FROM medical_constraints").
		WithArgs("user-123").
		WillReturnRows(rows)

	repo := NewMedicalRepository(db)

	if closeErr := db.Close(); closeErr != nil {
		t.Logf("failed to close db: %v", closeErr)
	}

	constraints, err := repo.GetActiveConstraints(context.Background(), "user-123")
	assert.Error(t, err)
	assert.Nil(t, constraints)
}

func TestSaveConstraintDBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectExec("INSERT INTO medical_constraints").
		WithArgs(
			sqlmock.AnyArg(), "user-123", "I10", "Hypertension", "cardiovascular", "moderate", "",
			sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
			true, sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnError(errors.New("database error"))

	repo := NewMedicalRepository(db)
	constraint := domain.MedicalConstraint{
		UserID:   "user-123",
		Code:     "I10",
		Label:    "Hypertension",
		Category: "cardiovascular",
		Severity: "moderate",
		ImpactOnTraining: []domain.ImpactRule{
			{Metric: "heart_rate", Action: "caution"},
		},
		Active: true,
	}

	err = repo.SaveConstraint(context.Background(), constraint)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save constraint")
}

func TestDeleteConstraintDBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectExec("DELETE FROM medical_constraints WHERE id = \\$1").
		WithArgs("mc-123").
		WillReturnError(errors.New("delete failed"))

	repo := NewMedicalRepository(db)
	err = repo.DeleteConstraint(context.Background(), "mc-123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delete constraint")
}

func TestGetConstraintByCodeQueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT.*FROM medical_constraints").
		WithArgs("I10").
		WillReturnError(errors.New("query failed"))

	repo := NewMedicalRepository(db)
	constraints, err := repo.GetConstraintByCode(context.Background(), "I10")

	assert.Error(t, err)
	assert.Nil(t, constraints)
	assert.Contains(t, err.Error(), "query constraints by code")
}

func TestGetConstraintByCodeRowsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows([]string{"id", "user_id", "code", "label", "category", "severity", "custom_text", "impact_on_training", "validated_by", "validated_at", "active", "created_at", "updated_at"}).
		AddRow("mc-1", "user-123", "I10", "Hypertension", "cardiovascular", "moderate", "", `[{"metric":"heart_rate","action":"caution"}]`, "doctor@example.com", time.Now(), true, time.Now(), time.Now())

	mock.ExpectQuery("SELECT.*FROM medical_constraints").
		WithArgs("I10").
		WillReturnRows(rows)

	repo := NewMedicalRepository(db)

	if closeErr := db.Close(); closeErr != nil {
		t.Logf("failed to close db: %v", closeErr)
	}

	constraints, err := repo.GetConstraintByCode(context.Background(), "I10")
	assert.Error(t, err)
	assert.Nil(t, constraints)
}

func TestScanMedicalConstraintWithValidatedFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "user_id", "code", "label", "category", "severity", "custom_text", "impact_on_training", "validated_by", "validated_at", "active", "created_at", "updated_at"}).
		AddRow("mc-1", "user-123", "I10", "Hypertension", "cardiovascular", "moderate", "", `[{"metric":"heart_rate","action":"caution"}]`, "doctor@example.com", now, true, now, now)

	mock.ExpectQuery("SELECT.*FROM medical_constraints").
		WithArgs("user-123").
		WillReturnRows(rows)

	repo := NewMedicalRepository(db)
	constraints, err := repo.GetActiveConstraints(context.Background(), "user-123")

	require.NoError(t, err)
	require.Len(t, constraints, 1)
	assert.Equal(t, "mc-1", constraints[0].ID)
	assert.Equal(t, "I10", constraints[0].Code)
	assert.Equal(t, "Hypertension", constraints[0].Label)
	assert.NotNil(t, constraints[0].ValidatedBy)
	assert.Equal(t, "doctor@example.com", *constraints[0].ValidatedBy)
	assert.NotNil(t, constraints[0].ValidatedAt)
}

func TestScanMedicalConstraintWithoutImpactJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "user_id", "code", "label", "category", "severity", "custom_text", "impact_on_training", "validated_by", "validated_at", "active", "created_at", "updated_at"}).
		AddRow("mc-1", "user-123", "I10", "Hypertension", "cardiovascular", "moderate", "", nil, nil, nil, true, now, now)

	mock.ExpectQuery("SELECT.*FROM medical_constraints").
		WithArgs("user-123").
		WillReturnRows(rows)

	repo := NewMedicalRepository(db)
	constraints, err := repo.GetActiveConstraints(context.Background(), "user-123")

	require.NoError(t, err)
	require.Len(t, constraints, 1)
	assert.Nil(t, constraints[0].ValidatedBy)
	assert.Nil(t, constraints[0].ValidatedAt)
	assert.Empty(t, constraints[0].ImpactOnTraining)
}

func TestScanMedicalConstraintWithInvalidImpactJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "user_id", "code", "label", "category", "severity", "custom_text", "impact_on_training", "validated_by", "validated_at", "active", "created_at", "updated_at"}).
		AddRow("mc-1", "user-123", "I10", "Hypertension", "cardiovascular", "moderate", "", `invalid-json`, nil, nil, true, now, now)

	mock.ExpectQuery("SELECT.*FROM medical_constraints").
		WithArgs("user-123").
		WillReturnRows(rows)

	repo := NewMedicalRepository(db)
	constraints, err := repo.GetActiveConstraints(context.Background(), "user-123")

	assert.Error(t, err)
	assert.Nil(t, constraints)
	assert.Contains(t, err.Error(), "unmarshal impact rules")
}
