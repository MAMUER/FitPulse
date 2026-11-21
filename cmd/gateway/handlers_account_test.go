package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeleteAccount_Unauthorized — без userID в контексте handler отдаёт 401
// и не трогает БД.
func TestDeleteAccount_Unauthorized(t *testing.T) {
	g := newTestGateway()
	w := httptest.NewRecorder()

	g.deleteAccountHandler(w, catalogReq(context.Background(), "DELETE", "/api/v1/account", ""))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), msgUnauthorized)
}

// TestDeleteAccount_NilDB — если БД не подключена, отвечаем 503.
func TestDeleteAccount_NilDB(t *testing.T) {
	g := newTestGateway() // db == nil
	w := httptest.NewRecorder()

	g.deleteAccountHandler(w, catalogReq(catalogCtx(), "DELETE", "/api/v1/account", ""))

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// TestDeleteAccount_Success — все 7 явных DELETE + удаление users выполняются,
// ответ 200. Явные DELETE не имеют ON DELETE CASCADE, поэтому проверяем именно их.
func TestDeleteAccount_Success(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	explicit := []string{
		"DELETE FROM friends",
		"DELETE FROM friend_requests",
		"DELETE FROM nutrition_meals",
		"DELETE FROM calendar_events",
		"DELETE FROM menstrual_cycles",
		"DELETE FROM leaderboard",
		"DELETE FROM reports",
	}
	for _, query := range explicit {
		mock.ExpectExec(query).WithArgs("u1").WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectExec("DELETE FROM users").WithArgs("u1").WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	g.deleteAccountHandler(w, catalogReq(catalogCtx(), "DELETE", "/api/v1/account", ""))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "account deleted")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteAccount_IgnoresExplicitDeleteErrors — ошибки явных DELETE (например,
// таблицы нет) не фатальны: чистим что есть и продолжаем.
func TestDeleteAccount_IgnoresExplicitDeleteErrors(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	for _, query := range []string{
		"DELETE FROM friends",
		"DELETE FROM friend_requests",
		"DELETE FROM nutrition_meals",
		"DELETE FROM calendar_events",
		"DELETE FROM menstrual_cycles",
		"DELETE FROM leaderboard",
		"DELETE FROM reports",
	} {
		mock.ExpectExec(query).WithArgs("u1").WillReturnError(errors.New("relation does not exist"))
	}
	mock.ExpectExec("DELETE FROM users").WithArgs("u1").WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	g.deleteAccountHandler(w, catalogReq(catalogCtx(), "DELETE", "/api/v1/account", ""))

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteAccount_UserDeleteError — если падает удаление users, отвечаем 500.
func TestDeleteAccount_UserDeleteError(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	for _, query := range []string{
		"DELETE FROM friends",
		"DELETE FROM friend_requests",
		"DELETE FROM nutrition_meals",
		"DELETE FROM calendar_events",
		"DELETE FROM menstrual_cycles",
		"DELETE FROM leaderboard",
		"DELETE FROM reports",
	} {
		mock.ExpectExec(query).WithArgs("u1").WillReturnResult(sqlmock.NewResult(0, 0))
	}
	mock.ExpectExec("DELETE FROM users").WithArgs("u1").WillReturnError(errors.New("updateviolates"))

	w := httptest.NewRecorder()
	g.deleteAccountHandler(w, catalogReq(catalogCtx(), "DELETE", "/api/v1/account", ""))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to delete account")
}
