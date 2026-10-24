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

func setupInviteCodeRepo(t *testing.T) (*inviteCodeRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return NewInviteCodeRepository(db).(*inviteCodeRepository), mock
}

func TestInviteCodeRepository_List_Success(t *testing.T) {
	repo, mock := setupInviteCodeRepo(t)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"code", "role", "specialty", "max_uses", "used_count", "is_active", "created_by", "created_at", "total_count"}).
		AddRow("INVITE1", "user", "doctor", 5, 2, true, "admin", now, 1)

	mock.ExpectQuery("SELECT code").WillReturnRows(rows)

	result, total, err := repo.List(ctx, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, result, 1)
	assert.Equal(t, "INVITE1", result[0].Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_List_QueryError(t *testing.T) {
	repo, mock := setupInviteCodeRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT code").WillReturnError(assert.AnError)

	_, _, err := repo.List(ctx, 0, 10)
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_Create_Success(t *testing.T) {
	repo, mock := setupInviteCodeRepo(t)
	ctx := context.Background()

	mock.ExpectExec("INSERT INTO invite_codes").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, &port.InviteCode{Code: "INVITE1", Role: "user", MaxUses: 5, CreatedBy: "admin"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_Create_Error(t *testing.T) {
	repo, mock := setupInviteCodeRepo(t)
	ctx := context.Background()

	mock.ExpectExec("INSERT INTO invite_codes").WillReturnError(assert.AnError)

	err := repo.Create(ctx, &port.InviteCode{Code: "INVITE1", Role: "user", MaxUses: 5, CreatedBy: "admin"})
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_Revoke_Success(t *testing.T) {
	repo, mock := setupInviteCodeRepo(t)
	ctx := context.Background()

	mock.ExpectExec("UPDATE invite_codes").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Revoke(ctx, "INVITE1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_Revoke_NotFound(t *testing.T) {
	repo, mock := setupInviteCodeRepo(t)
	ctx := context.Background()

	mock.ExpectExec("UPDATE invite_codes").WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Revoke(ctx, "INVITE1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "NOT_FOUND")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_Revoke_Error(t *testing.T) {
	repo, mock := setupInviteCodeRepo(t)
	ctx := context.Background()

	mock.ExpectExec("UPDATE invite_codes").WillReturnError(assert.AnError)

	err := repo.Revoke(ctx, "INVITE1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_Validate_Success(t *testing.T) {
	repo, mock := setupInviteCodeRepo(t)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"code", "role", "specialty", "max_uses", "used_count", "is_active", "created_by", "created_at"}).
		AddRow("INVITE1", "user", "doctor", 5, 2, true, "admin", now)

	mock.ExpectQuery("SELECT code").WillReturnRows(rows)

	result, err := repo.Validate(ctx, "INVITE1")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "INVITE1", result.Code)
	assert.Equal(t, "user", result.Role)
	assert.Equal(t, "doctor", *result.Specialty)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_Validate_NotFound(t *testing.T) {
	repo, mock := setupInviteCodeRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT code").WillReturnError(sql.ErrNoRows)

	result, err := repo.Validate(ctx, "INVITE1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "NOT_FOUND")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_Validate_QueryError(t *testing.T) {
	repo, mock := setupInviteCodeRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT code").WillReturnError(assert.AnError)

	result, err := repo.Validate(ctx, "INVITE1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_UseInviteCode_Success(t *testing.T) {
	repo, mock := setupInviteCodeRepo(t)
	ctx := context.Background()

	mock.ExpectExec("SELECT \\* FROM use_invite_code\\(").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UseInviteCode(ctx, "INVITE1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_UseInviteCode_NotFound(t *testing.T) {
	repo, mock := setupInviteCodeRepo(t)
	ctx := context.Background()

	mock.ExpectExec("SELECT \\* FROM use_invite_code\\(").WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.UseInviteCode(ctx, "INVITE1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "NOT_FOUND")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_UseInviteCode_Error(t *testing.T) {
	repo, mock := setupInviteCodeRepo(t)
	ctx := context.Background()

	mock.ExpectExec("SELECT \\* FROM use_invite_code\\(").WillReturnError(assert.AnError)

	err := repo.UseInviteCode(ctx, "INVITE1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_ValidateInviteCodeUse_Success(t *testing.T) {
	repo, mock := setupInviteCodeRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"is_valid", "role", "specialty", "err_msg"}).
		AddRow(true, "user", "doctor", "")

	mock.ExpectQuery("SELECT \\* FROM use_invite_code").WillReturnRows(rows)

	isValid, role, specialty, errMsg, err := repo.ValidateInviteCodeUse(ctx, "INVITE1")
	require.NoError(t, err)
	assert.True(t, isValid)
	assert.Equal(t, "user", role)
	assert.Equal(t, "doctor", specialty)
	assert.Equal(t, "", errMsg)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_ValidateInviteCodeUse_Error(t *testing.T) {
	repo, mock := setupInviteCodeRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT \\* FROM use_invite_code").WillReturnError(assert.AnError)

	isValid, role, specialty, errMsg, err := repo.ValidateInviteCodeUse(ctx, "INVITE1")
	require.Error(t, err)
	assert.False(t, isValid)
	assert.Equal(t, "", role)
	assert.Equal(t, "", specialty)
	assert.Equal(t, "", errMsg)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_LogInviteCodeUse_Success(t *testing.T) {
	repo, mock := setupInviteCodeRepo(t)
	ctx := context.Background()

	mock.ExpectExec("SELECT log_invite_code_use").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.LogInviteCodeUse(ctx, "INVITE1", "user-1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteCodeRepository_LogInviteCodeUse_Error(t *testing.T) {
	repo, mock := setupInviteCodeRepo(t)
	ctx := context.Background()

	mock.ExpectExec("SELECT log_invite_code_use").WillReturnError(assert.AnError)

	err := repo.LogInviteCodeUse(ctx, "INVITE1", "user-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}
