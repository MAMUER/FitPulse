package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/middleware"
)

// newCatalogMockDB creates a sqlmock-backed *sql.DB for the catalog/friends/report
// handler tests and closes it on cleanup.
func newCatalogMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func catalogCtx() context.Context {
	return context.WithValue(context.Background(), middleware.UserIDKey, "u1")
}

func catalogReq(ctx context.Context, method, target, body string) *http.Request {
	if body == "" {
		return httptest.NewRequestWithContext(ctx, method, target, nil)
	}
	return httptest.NewRequestWithContext(ctx, method, target, bytes.NewBufferString(body))
}

// ========== handlers_dishes.go ==========

func TestDishes_GetDishesHandler_NilDB(t *testing.T) {
	g := newTestGateway() // db == nil

	w := httptest.NewRecorder()
	g.getDishesHandler(w, catalogReq(catalogCtx(), "GET", "/api/v1/dishes", ""))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "dishes")
}

func TestDishes_GetDishesHandler_Success(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT id, name, meal_slot").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "meal_slot", "kcal", "protein_g", "fat_g", "carbs_g", "image_url"}).
				AddRow(int64(1), "Овсянка", "breakfast", 300, 10, 5, 60, ""),
		)

	w := httptest.NewRecorder()
	g.getDishesHandler(w, catalogReq(catalogCtx(), "GET", "/api/v1/dishes?slot=breakfast&q=овс", ""))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Овсянка")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDishes_GetDishesHandler_QueryError(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT id, name, meal_slot").WillReturnError(errors.New("boom"))

	w := httptest.NewRecorder()
	g.getDishesHandler(w, catalogReq(catalogCtx(), "GET", "/api/v1/dishes", ""))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDishes_AddDishHandler_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.addDishHandler(w, catalogReq(context.Background(), "POST", "/api/v1/dishes", `{"name":"x"}`))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDishes_AddDishHandler_NilDB(t *testing.T) {
	g := newTestGateway() // db == nil

	w := httptest.NewRecorder()
	g.addDishHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/dishes", `{"name":"x"}`))

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestDishes_AddDishHandler_InvalidJSON(t *testing.T) {
	db, _ := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	w := httptest.NewRecorder()
	g.addDishHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/dishes", "{"))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDishes_AddDishHandler_EmptyName(t *testing.T) {
	db, _ := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	w := httptest.NewRecorder()
	g.addDishHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/dishes", `{"name":"   "}`))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDishes_AddDishHandler_ImageTooLarge(t *testing.T) {
	db, _ := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	big := strings.Repeat("a", 400_001)
	body := `{"name":"Блюдо","image_url":"` + big + `"}`

	w := httptest.NewRecorder()
	g.addDishHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/dishes", body))

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestDishes_AddDishHandler_CountError(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT COUNT").WillReturnError(errors.New("boom"))

	w := httptest.NewRecorder()
	g.addDishHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/dishes", `{"name":"Блюдо"}`))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDishes_AddDishHandler_LimitReached(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(maxDishesPerUser))

	w := httptest.NewRecorder()
	g.addDishHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/dishes", `{"name":"Блюдо"}`))

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestDishes_AddDishHandler_Success(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("INSERT INTO dishes").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(7)))

	w := httptest.NewRecorder()
	g.addDishHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/dishes",
		`{"name":"Блюдо","meal_slot":"lunch","kcal":500,"protein_g":20,"fat_g":10,"carbs_g":60}`))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":7`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDishes_AddDishHandler_InsertError(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("INSERT INTO dishes").WillReturnError(errors.New("boom"))

	w := httptest.NewRecorder()
	g.addDishHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/dishes", `{"name":"Блюдо"}`))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ========== handlers_friends.go ==========

func TestFriends_GetFriendsHandler_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.getFriendsHandler(w, catalogReq(context.Background(), "GET", "/api/v1/friends", ""))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestFriends_GetFriendsHandler_NilDB(t *testing.T) {
	g := newTestGateway() // db == nil

	w := httptest.NewRecorder()
	g.getFriendsHandler(w, catalogReq(catalogCtx(), "GET", "/api/v1/friends", ""))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "friends")
}

func TestFriends_GetFriendsHandler_Success(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT id, friend_name, friend_user_id").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "friend_name", "friend_user_id", "status"}).
				AddRow(int64(1), "Анна", "user-2", "online").
				AddRow(int64(2), "Борис", nil, "offline"),
		)

	w := httptest.NewRecorder()
	g.getFriendsHandler(w, catalogReq(catalogCtx(), "GET", "/api/v1/friends", ""))

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Анна")
	assert.Contains(t, body, `"online":true`)
	assert.Contains(t, body, "Борис")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFriends_GetFriendsHandler_QueryError(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT id, friend_name, friend_user_id").WillReturnError(errors.New("boom"))

	w := httptest.NewRecorder()
	g.getFriendsHandler(w, catalogReq(catalogCtx(), "GET", "/api/v1/friends", ""))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestFriends_AddFriendHandler_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.addFriendHandler(w, catalogReq(context.Background(), "POST", "/api/v1/friends", `{"friend_name":"x"}`))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestFriends_AddFriendHandler_NilDB(t *testing.T) {
	g := newTestGateway() // db == nil

	w := httptest.NewRecorder()
	g.addFriendHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/friends", `{"friend_name":"x"}`))

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestFriends_AddFriendHandler_InvalidJSON(t *testing.T) {
	db, _ := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	w := httptest.NewRecorder()
	g.addFriendHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/friends", "{"))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFriends_AddFriendHandler_EmptyName(t *testing.T) {
	db, _ := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	w := httptest.NewRecorder()
	g.addFriendHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/friends", `{"friend_name":"  "}`))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFriends_AddFriendHandler_Success(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("INSERT INTO friends").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(9)))

	w := httptest.NewRecorder()
	g.addFriendHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/friends",
		`{"friend_name":"Анна","friend_user_id":"user-2","status":"online"}`))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":9`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFriends_AddFriendHandler_InsertError(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("INSERT INTO friends").WillReturnError(errors.New("boom"))

	w := httptest.NewRecorder()
	g.addFriendHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/friends", `{"friend_name":"Анна"}`))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestFriends_RemoveFriendHandler_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.removeFriendHandler(w, catalogReq(context.Background(), "DELETE", "/api/v1/friends/1", ""))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestFriends_RemoveFriendHandler_NilDB(t *testing.T) {
	g := newTestGateway() // db == nil

	w := httptest.NewRecorder()
	g.removeFriendHandler(w, catalogReq(catalogCtx(), "DELETE", "/api/v1/friends/1", ""))

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestFriends_RemoveFriendHandler_InvalidID(t *testing.T) {
	db, _ := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	w := httptest.NewRecorder()
	g.removeFriendHandler(w, catalogReq(catalogCtx(), "DELETE", "/api/v1/friends/abc", ""))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFriends_RemoveFriendHandler_NonPositiveID(t *testing.T) {
	db, _ := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	w := httptest.NewRecorder()
	g.removeFriendHandler(w, catalogReq(catalogCtx(), "DELETE", "/api/v1/friends/0", ""))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFriends_RemoveFriendHandler_Success(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectExec("DELETE FROM friends").WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	g.removeFriendHandler(w, catalogReq(catalogCtx(), "DELETE", "/api/v1/friends/5", ""))

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFriends_RemoveFriendHandler_ExecError(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectExec("DELETE FROM friends").WillReturnError(errors.New("boom"))

	w := httptest.NewRecorder()
	g.removeFriendHandler(w, catalogReq(catalogCtx(), "DELETE", "/api/v1/friends/5", ""))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ========== handlers_report.go ==========

func TestReport_Handler_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.reportHandler(w, catalogReq(context.Background(), "POST", "/api/v1/report", `{"reason":"x"}`))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestReport_Handler_InvalidJSON(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.reportHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/report", "{"))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReport_Handler_NilDB(t *testing.T) {
	// notifyOncall must not attempt a network call without Telegram env.
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")
	g := newTestGateway() // db == nil

	w := httptest.NewRecorder()
	g.reportHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/report",
		`{"reported_user_id":"u2","reason":"spam"}`))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestReport_Handler_Success(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectExec("INSERT INTO reports").WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	g.reportHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/report",
		`{"reported_user_id":"u2","reason":"spam"}`))

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReport_InsertError(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	// Insert error is logged but not fatal: the handler still returns ok.
	mock.ExpectExec("INSERT INTO reports").WillReturnError(errors.New("boom"))

	w := httptest.NewRecorder()
	g.reportHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/report",
		`{"reason":"spam"}`))

	assert.Equal(t, http.StatusOK, w.Code)
}

// ========== getEnvOrFile ==========

func TestGetEnvOrFile_DirectEnv(t *testing.T) {
	t.Setenv("FP_TEST_DIRECT", "value-from-env")

	assert.Equal(t, "value-from-env", getEnvOrFile("FP_TEST_DIRECT"))
}

func TestGetEnvOrFile_FromFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "secret")
	require.NoError(t, os.WriteFile(file, []byte("  secret-value\n"), 0o600))

	t.Setenv("FP_TEST_FILEVAR", "")
	t.Setenv("FP_TEST_FILEVAR_FILE", file)

	assert.Equal(t, "secret-value", getEnvOrFile("FP_TEST_FILEVAR"))
}

func TestGetEnvOrFile_Missing(t *testing.T) {
	t.Setenv("FP_TEST_MISSING", "")
	t.Setenv("FP_TEST_MISSING_FILE", "")

	assert.Equal(t, "", getEnvOrFile("FP_TEST_MISSING"))
}

func TestGetEnvOrFile_FileUnreadable(t *testing.T) {
	t.Setenv("FP_TEST_BADFILE", "")
	t.Setenv("FP_TEST_BADFILE_FILE", filepath.Join(t.TempDir(), "does-not-exist"))

	assert.Equal(t, "", getEnvOrFile("FP_TEST_BADFILE"))
}
