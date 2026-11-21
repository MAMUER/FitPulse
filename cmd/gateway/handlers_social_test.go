package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// socialReqWithParams builds a request with chi URL params (id/action) attached,
// as the router would provide for /friends/requests/{id}/{action}.
func socialReqWithParams(ctx context.Context, target, id, action string) *http.Request {
	r := catalogReq(ctx, "POST", target, "")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	rctx.URLParams.Add("action", action)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// ========== Leaderboard ==========

func TestSocial_Leaderboard_NilDB(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.getLeaderboardHandler(w, catalogReq(catalogCtx(), "GET", "/api/v1/leaderboard", ""))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "leaderboard")
}

func TestSocial_Leaderboard_Success(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT user_id, display_name, points").
		WillReturnRows(
			sqlmock.NewRows([]string{"user_id", "display_name", "points", "xp", "workouts"}).
				AddRow("u1", "Анна", 100, 10, 5).
				AddRow("u2", "Борис", 50, 5, 2),
		)

	w := httptest.NewRecorder()
	g.getLeaderboardHandler(w, catalogReq(catalogCtx(), "GET", "/api/v1/leaderboard", ""))

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Анна")
	assert.Contains(t, body, `"me":true`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSocial_Leaderboard_QueryError(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT user_id, display_name, points").WillReturnError(errors.New("boom"))

	w := httptest.NewRecorder()
	g.getLeaderboardHandler(w, catalogReq(catalogCtx(), "GET", "/api/v1/leaderboard", ""))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ========== Sync leaderboard ==========

func TestSocial_SyncLeaderboard_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.syncLeaderboardHandler(w, catalogReq(context.Background(), "PUT", "/api/v1/leaderboard/me", `{"points":1}`))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSocial_SyncLeaderboard_NilDB(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.syncLeaderboardHandler(w, catalogReq(catalogCtx(), "PUT", "/api/v1/leaderboard/me", `{"points":1}`))

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestSocial_SyncLeaderboard_InvalidJSON(t *testing.T) {
	db, _ := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	w := httptest.NewRecorder()
	g.syncLeaderboardHandler(w, catalogReq(catalogCtx(), "PUT", "/api/v1/leaderboard/me", "{"))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSocial_SyncLeaderboard_Success(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectExec("INSERT INTO leaderboard").WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	g.syncLeaderboardHandler(w, catalogReq(catalogCtx(), "PUT", "/api/v1/leaderboard/me",
		`{"display_name":"Анна","points":100,"xp":10,"workouts":5}`))

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSocial_SyncLeaderboard_ClampsNegatives(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	// Empty name falls back to the default; negative counters clamp to 0.
	mock.ExpectExec("INSERT INTO leaderboard").WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	g.syncLeaderboardHandler(w, catalogReq(catalogCtx(), "PUT", "/api/v1/leaderboard/me",
		`{"display_name":"  ","points":-5,"xp":-1,"workouts":-2}`))

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSocial_SyncLeaderboard_ExecError(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectExec("INSERT INTO leaderboard").WillReturnError(errors.New("boom"))

	w := httptest.NewRecorder()
	g.syncLeaderboardHandler(w, catalogReq(catalogCtx(), "PUT", "/api/v1/leaderboard/me", `{"points":1}`))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ========== Search users ==========

func TestSocial_SearchUsers_NilDB(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.searchUsersHandler(w, catalogReq(catalogCtx(), "GET", "/api/v1/users/search?q=ан", ""))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "users")
}

func TestSocial_SearchUsers_ShortQuery(t *testing.T) {
	db, _ := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	w := httptest.NewRecorder()
	g.searchUsersHandler(w, catalogReq(catalogCtx(), "GET", "/api/v1/users/search?q=a", ""))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"users":[]`)
}

func TestSocial_SearchUsers_Success(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT user_id, display_name, points").
		WillReturnRows(
			sqlmock.NewRows([]string{"user_id", "display_name", "points"}).
				AddRow("u2", "Анна", 42),
		)

	w := httptest.NewRecorder()
	g.searchUsersHandler(w, catalogReq(catalogCtx(), "GET", "/api/v1/users/search?q=анна", ""))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Анна")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSocial_SearchUsers_QueryError(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT user_id, display_name, points").WillReturnError(errors.New("boom"))

	w := httptest.NewRecorder()
	g.searchUsersHandler(w, catalogReq(catalogCtx(), "GET", "/api/v1/users/search?q=анна", ""))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ========== Send friend request ==========

func TestSocial_SendFriendRequest_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.sendFriendRequestHandler(w, catalogReq(context.Background(), "POST", "/api/v1/friends/request", `{"to_user_id":"u2"}`))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSocial_SendFriendRequest_NilDB(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.sendFriendRequestHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/friends/request", `{"to_user_id":"u2"}`))

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestSocial_SendFriendRequest_InvalidJSON(t *testing.T) {
	db, _ := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	w := httptest.NewRecorder()
	g.sendFriendRequestHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/friends/request", "{"))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSocial_SendFriendRequest_EmptyOrSelf(t *testing.T) {
	db, _ := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	// Empty target.
	w := httptest.NewRecorder()
	g.sendFriendRequestHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/friends/request", `{"to_user_id":""}`))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Self target.
	w2 := httptest.NewRecorder()
	g.sendFriendRequestHandler(w2, catalogReq(catalogCtx(), "POST", "/api/v1/friends/request", `{"to_user_id":"u1"}`))
	assert.Equal(t, http.StatusBadRequest, w2.Code)
}

func TestSocial_SendFriendRequest_Success(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectExec("INSERT INTO friend_requests").WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	g.sendFriendRequestHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/friends/request", `{"to_user_id":"u2"}`))

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSocial_SendFriendRequest_ExecError(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectExec("INSERT INTO friend_requests").WillReturnError(errors.New("boom"))

	w := httptest.NewRecorder()
	g.sendFriendRequestHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/friends/request", `{"to_user_id":"u2"}`))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ========== List friend requests ==========

func TestSocial_ListFriendRequests_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.listFriendRequestsHandler(w, catalogReq(context.Background(), "GET", "/api/v1/friends/requests", ""))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSocial_ListFriendRequests_NilDB(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.listFriendRequestsHandler(w, catalogReq(catalogCtx(), "GET", "/api/v1/friends/requests", ""))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "requests")
}

func TestSocial_ListFriendRequests_Success(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT fr.id, fr.from_user_id").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "from_user_id", "display_name"}).
				AddRow(int64(1), "u2", "Анна").
				AddRow(int64(2), "u3", ""), // empty name -> default
		)

	w := httptest.NewRecorder()
	g.listFriendRequestsHandler(w, catalogReq(catalogCtx(), "GET", "/api/v1/friends/requests", ""))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Анна")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSocial_ListFriendRequests_QueryError(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT fr.id, fr.from_user_id").WillReturnError(errors.New("boom"))

	w := httptest.NewRecorder()
	g.listFriendRequestsHandler(w, catalogReq(catalogCtx(), "GET", "/api/v1/friends/requests", ""))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ========== Respond friend request ==========

func TestSocial_Respond_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.respondFriendRequestHandler(w, socialReqWithParams(context.Background(), "/api/v1/friends/requests/1/accept", "1", "accept"))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSocial_Respond_NilDB(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.respondFriendRequestHandler(w, socialReqWithParams(catalogCtx(), "/api/v1/friends/requests/1/accept", "1", "accept"))

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestSocial_Respond_InvalidID(t *testing.T) {
	db, _ := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	w := httptest.NewRecorder()
	g.respondFriendRequestHandler(w, socialReqWithParams(catalogCtx(), "/api/v1/friends/requests/x/accept", "x", "accept"))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSocial_Respond_InvalidAction(t *testing.T) {
	db, _ := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	w := httptest.NewRecorder()
	g.respondFriendRequestHandler(w, socialReqWithParams(catalogCtx(), "/api/v1/friends/requests/1/whatever", "1", "whatever"))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSocial_Respond_NotFound(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT from_user_id, status FROM friend_requests").
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	g.respondFriendRequestHandler(w, socialReqWithParams(catalogCtx(), "/api/v1/friends/requests/1/accept", "1", "accept"))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSocial_Respond_LoadError(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT from_user_id, status FROM friend_requests").WillReturnError(errors.New("boom"))

	w := httptest.NewRecorder()
	g.respondFriendRequestHandler(w, socialReqWithParams(catalogCtx(), "/api/v1/friends/requests/1/accept", "1", "accept"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSocial_Respond_NotPending(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT from_user_id, status FROM friend_requests").
		WillReturnRows(sqlmock.NewRows([]string{"from_user_id", "status"}).AddRow("u2", "accepted"))

	w := httptest.NewRecorder()
	g.respondFriendRequestHandler(w, socialReqWithParams(catalogCtx(), "/api/v1/friends/requests/1/accept", "1", "accept"))

	assert.Equal(t, http.StatusOK, w.Code)
	// No UPDATE expected because the request is not pending.
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSocial_Respond_Decline(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT from_user_id, status FROM friend_requests").
		WillReturnRows(sqlmock.NewRows([]string{"from_user_id", "status"}).AddRow("u2", "pending"))
	mock.ExpectExec("UPDATE friend_requests SET status").WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	g.respondFriendRequestHandler(w, socialReqWithParams(catalogCtx(), "/api/v1/friends/requests/1/decline", "1", "decline"))

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSocial_Respond_UpdateError(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT from_user_id, status FROM friend_requests").
		WillReturnRows(sqlmock.NewRows([]string{"from_user_id", "status"}).AddRow("u2", "pending"))
	mock.ExpectExec("UPDATE friend_requests SET status").WillReturnError(errors.New("boom"))

	w := httptest.NewRecorder()
	g.respondFriendRequestHandler(w, socialReqWithParams(catalogCtx(), "/api/v1/friends/requests/1/decline", "1", "decline"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSocial_Respond_Accept(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT from_user_id, status FROM friend_requests").
		WillReturnRows(sqlmock.NewRows([]string{"from_user_id", "status"}).AddRow("u2", "pending"))
	mock.ExpectExec("UPDATE friend_requests SET status").WillReturnResult(sqlmock.NewResult(0, 1))
	// linkFriends(fromID,userID) and linkFriends(userID,fromID): each reads the
	// leaderboard name then checks existence. Force the exists check to error so
	// linking stops without inserts (insert path is covered by TestLinkFriends).
	mock.ExpectQuery("SELECT display_name FROM leaderboard").WillReturnError(errors.New("no name"))
	mock.ExpectQuery("SELECT EXISTS").WillReturnError(errors.New("exists check failed"))
	mock.ExpectQuery("SELECT display_name FROM leaderboard").WillReturnError(errors.New("no name"))
	mock.ExpectQuery("SELECT EXISTS").WillReturnError(errors.New("exists check failed"))

	w := httptest.NewRecorder()
	g.respondFriendRequestHandler(w, socialReqWithParams(catalogCtx(), "/api/v1/friends/requests/1/accept", "1", "accept"))

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// ========== linkFriends ==========

func TestLinkFriends_Guards(t *testing.T) {
	db, _ := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	// All guard branches return without touching the DB.
	g.linkFriends("", "u2")
	g.linkFriends("u1", "")
	g.linkFriends("u1", "u1")
}

func TestLinkFriends_InsertsWhenMissing(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT display_name FROM leaderboard").
		WillReturnRows(sqlmock.NewRows([]string{"display_name"}).AddRow("Анна"))
	mock.ExpectQuery("SELECT EXISTS").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec("INSERT INTO friends").WillReturnResult(sqlmock.NewResult(0, 1))

	g.linkFriends("u1", "u2")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLinkFriends_NameFallbackAndExists(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	// display_name error -> fallback name, then exists=true -> return (no insert).
	mock.ExpectQuery("SELECT display_name FROM leaderboard").WillReturnError(errors.New("no name"))
	mock.ExpectQuery("SELECT EXISTS").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	g.linkFriends("u1", "u2")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLinkFriends_InsertError(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT display_name FROM leaderboard").
		WillReturnRows(sqlmock.NewRows([]string{"display_name"}).AddRow("Анна"))
	mock.ExpectQuery("SELECT EXISTS").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec("INSERT INTO friends").WillReturnError(errors.New("boom"))

	// Insert failure is logged, not returned.
	g.linkFriends("u1", "u2")

	require.NoError(t, mock.ExpectationsWereMet())
}
