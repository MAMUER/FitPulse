package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/port"
)

func setupInviteRepo(t *testing.T) (*inviteCodeRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return NewInviteCodeRepository(db).(*inviteCodeRepository), mock
}

func TestInviteCodeRepository_List_Success(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()
	now := time.Now()
	specialty := "cardiology"

	rows := sqlmock.NewRows([]string{"code", "role", "specialty", "max_uses", "used_count", "is_active", "created_by", "created_at", "total_count"}).
		AddRow("INVITE-1", "doctor", specialty, 10, 2, true, "admin-1", now, 1)

	mock.ExpectQuery("SELECT code").
		WithArgs(10, 0).
		WillReturnRows(rows)

	result, total, err := repo.List(ctx, 0, 10)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "INVITE-1", result[0].Code)
	assert.Equal(t, 1, total)
	assert.NotNil(t, result[0].Specialty)
	assert.Equal(t, "cardiology", *result[0].Specialty)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_List_Empty(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"code", "role", "specialty", "max_uses", "used_count", "is_active", "created_by", "created_at", "total_count"})

	mock.ExpectQuery("SELECT code").
		WithArgs(10, 0).
		WillReturnRows(rows)

	result, total, err := repo.List(ctx, 0, 10)
	require.NoError(t, err)
	assert.Empty(t, result)
	assert.Equal(t, 0, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_List_QueryError(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT code").WillReturnError(assert.AnError)

	result, total, err := repo.List(ctx, 0, 10)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_List_ScanError(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"code", "role", "specialty", "max_uses", "used_count", "is_active", "created_by", "created_at", "total_count"}).
		AddRow(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	mock.ExpectQuery("SELECT code").
		WithArgs(10, 0).
		WillReturnRows(rows)

	result, total, err := repo.List(ctx, 0, 10)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_Create_Success(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()
	invite := &port.InviteCode{
		Code:      "INVITE-1",
		Role:      "doctor",
		MaxUses:   10,
		CreatedBy: "admin-1",
	}

	mock.ExpectExec("INSERT INTO invite_codes").
		WithArgs("INVITE-1", "doctor", nil, 10, "admin-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, invite)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_Create_WithSpecialty(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()
	specialty := "cardiology"
	invite := &port.InviteCode{
		Code:      "INVITE-1",
		Role:      "doctor",
		Specialty: &specialty,
		MaxUses:   10,
		CreatedBy: "admin-1",
	}

	mock.ExpectExec("INSERT INTO invite_codes").
		WithArgs("INVITE-1", "doctor", "cardiology", 10, "admin-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, invite)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_Create_Error(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()

	mock.ExpectExec("INSERT INTO invite_codes").WillReturnError(assert.AnError)

	err := repo.Create(ctx, &port.InviteCode{Code: "INVITE-1", Role: "doctor", MaxUses: 10, CreatedBy: "admin-1"})
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_Revoke_Success(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()

	mock.ExpectExec("UPDATE invite_codes").
		WithArgs("INVITE-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Revoke(ctx, "INVITE-1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_Revoke_NotFound(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()

	mock.ExpectExec("UPDATE invite_codes").
		WithArgs("INVITE-1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Revoke(ctx, "INVITE-1")
	require.Error(t, err)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_Revoke_Error(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()

	mock.ExpectExec("UPDATE invite_codes").
		WithArgs("INVITE-1").
		WillReturnError(assert.AnError)

	err := repo.Revoke(ctx, "INVITE-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_Validate_Success(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"code", "role", "specialty", "max_uses", "used_count", "is_active", "created_by", "created_at"}).
		AddRow("INVITE-1", "doctor", nil, 10, 2, true, "admin-1", now)

	mock.ExpectQuery("SELECT code").
		WithArgs("INVITE-1").
		WillReturnRows(rows)

	result, err := repo.Validate(ctx, "INVITE-1")
	require.NoError(t, err)
	assert.Equal(t, "INVITE-1", result.Code)
	assert.Equal(t, "doctor", result.Role)
	assert.True(t, result.IsActive)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_Validate_NotFound(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT code").
		WithArgs("INVITE-1").
		WillReturnError(sql.ErrNoRows)

	result, err := repo.Validate(ctx, "INVITE-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_Validate_QueryError(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT code").WillReturnError(assert.AnError)

	result, err := repo.Validate(ctx, "INVITE-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_UseInviteCode_Success(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()

	mock.ExpectExec("SELECT").
		WithArgs("INVITE-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UseInviteCode(ctx, "INVITE-1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_UseInviteCode_NotFound(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()

	mock.ExpectExec("SELECT").
		WithArgs("INVITE-1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.UseInviteCode(ctx, "INVITE-1")
	require.Error(t, err)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_UseInviteCode_Error(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()

	mock.ExpectExec("SELECT").WillReturnError(assert.AnError)

	err := repo.UseInviteCode(ctx, "INVITE-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_ValidateInviteCodeUse_Success(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"is_valid", "role", "specialty", "err_msg"}).
		AddRow(true, "doctor", "cardiology", "")

	mock.ExpectQuery("SELECT").
		WithArgs("INVITE-1").
		WillReturnRows(rows)

	valid, role, specialty, errMsg, err := repo.ValidateInviteCodeUse(ctx, "INVITE-1")
	require.NoError(t, err)
	assert.True(t, valid)
	assert.Equal(t, "doctor", role)
	assert.Equal(t, "cardiology", specialty)
	assert.Empty(t, errMsg)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_ValidateInviteCodeUse_Error(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT").WillReturnError(assert.AnError)

	valid, role, specialty, errMsg, err := repo.ValidateInviteCodeUse(ctx, "INVITE-1")
	require.Error(t, err)
	assert.False(t, valid)
	assert.Empty(t, role)
	assert.Empty(t, specialty)
	assert.Empty(t, errMsg)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_LogInviteCodeUse_Success(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()

	mock.ExpectExec("SELECT").
		WithArgs("INVITE-1", "user-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.LogInviteCodeUse(ctx, "INVITE-1", "user-1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_LogInviteCodeUse_Error(t *testing.T) {
	repo, mock := setupInviteRepo(t)
	ctx := context.Background()

	mock.ExpectExec("SELECT").WillReturnError(assert.AnError)

	err := repo.LogInviteCodeUse(ctx, "INVITE-1", "user-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}
