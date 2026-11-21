package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/auth/password"
)

func TestChangePassword_Unauthorized(t *testing.T) {
	g := newTestGateway()
	w := httptest.NewRecorder()

	g.changePasswordHandler(
		w,
		catalogReq(context.Background(), "POST", "/api/v1/auth/change-password", ""),
	)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestChangePassword_Validation(t *testing.T) {
	g := newTestGateway()

	cases := []struct {
		name string
		body string
	}{
		{"empty body", `{}`},
		{"short new password", `{"old_password":"oldpass123","new_password":"short"}`},
		{"missing old password", `{"new_password":"newpassword123"}`},
		{"broken json", `not-json`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			g.changePasswordHandler(
				w,
				catalogReq(catalogCtx(), "POST", "/api/v1/auth/change-password", tc.body),
			)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	hash, err := password.HashArgon2id("oldpass123")
	require.NoError(t, err)

	mock.ExpectQuery("SELECT password_hash FROM users").
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"password_hash"}).AddRow(hash))

	w := httptest.NewRecorder()
	body := `{"old_password":"wrongpass","new_password":"newpassword123"}`
	g.changePasswordHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/auth/change-password", body))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid current password")
}

func TestChangePassword_Success(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	hash, err := password.HashArgon2id("oldpass123")
	require.NoError(t, err)

	mock.ExpectQuery("SELECT password_hash FROM users").
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"password_hash"}).AddRow(hash))
	mock.ExpectExec("UPDATE users SET password_hash").
		WithArgs(sqlmock.AnyArg(), "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM refresh_tokens").
		WithArgs("u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	body := `{"old_password":"oldpass123","new_password":"newpassword123"}`
	g.changePasswordHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/auth/change-password", body))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChangePassword_UserNotFound(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT password_hash FROM users").
		WithArgs("u1").
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	body := `{"old_password":"oldpass123","new_password":"newpassword123"}`
	g.changePasswordHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/auth/change-password", body))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
