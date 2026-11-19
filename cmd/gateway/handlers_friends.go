package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/MAMUER/project/internal/middleware"
	"go.uber.org/zap"
)

// friendsTableAvailable сообщает, сконфигурирована ли БД для друзей.
func (g *gateway) friendsTableAvailable() bool {
	return g.db != nil
}

// msgInvalidRequest — user-facing message for malformed friend payloads.
const msgInvalidRequest = "неверный запрос"

// listFriendsFromDB возвращает друзей пользователя из таблицы friends.
func (g *gateway) listFriendsFromDB(
	userID string,
) ([]map[string]interface{}, error) {
	rows, err := g.db.Query(
		`SELECT id, friend_name, friend_user_id, status
		 FROM friends WHERE user_id = $1 ORDER BY added_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	friends := []map[string]interface{}{}
	for rows.Next() {
		var (
			id           int64
			name         string
			friendUserID sql.NullString
			status       string
		)
		if rows.Scan(&id, &name, &friendUserID, &status) != nil {
			continue
		}
		online := status == "online" || status == "gym" || status == "pool"
		friend := map[string]interface{}{
			"id":     id,
			"name":   name,
			"status": status,
			"online": online,
			"added":  true,
		}
		if friendUserID.Valid {
			friend["friend_user_id"] = friendUserID.String
		}
		friends = append(friends, friend)
	}
	return friends, rows.Err()
}

// @Summary      List friends
// @Description  Returns the user's friends list.
// @Tags         friends
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/friends [get]
func (g *gateway) getFriendsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		g.log.Error(errUnauthorized, zap.String("handler", "getFriends"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}

	if !g.friendsTableAvailable() {
		writeJSON(w, map[string]interface{}{"status": "ok", "friends": []interface{}{}})
		return
	}

	friends, err := g.listFriendsFromDB(userID)
	if err != nil {
		g.log.Error("Failed to list friends", zap.Error(err))
		http.Error(w, "failed to list friends", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{"status": "ok", "friends": friends})
}

// @Summary      Add friend
// @Description  Adds a friend to the user's list.
// @Tags         friends
// @Accept       json
// @Produce      json
// @Param        body  body      object  true  "Friend payload"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/v1/friends [post]
func (g *gateway) addFriendHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	if !g.friendsTableAvailable() {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		FriendName   string `json:"friend_name"`
		FriendUserID string `json:"friend_user_id"`
		Status       string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, msgInvalidRequest, http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(req.FriendName)
	if name == "" {
		http.Error(w, msgInvalidRequest, http.StatusBadRequest)
		return
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "offline"
	}

	var friendUserID sql.NullString
	if v := strings.TrimSpace(req.FriendUserID); v != "" {
		friendUserID = sql.NullString{String: v, Valid: true}
	}

	var id int64
	err := g.db.QueryRow(
		`INSERT INTO friends (user_id, friend_name, friend_user_id, status)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		userID, name, friendUserID, status,
	).Scan(&id)
	if err != nil {
		g.log.Error("Failed to add friend", zap.Error(err))
		http.Error(w, "failed to add friend", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{"status": "ok", "id": id})
}

// @Summary      Remove friend
// @Description  Removes a friend by id.
// @Tags         friends
// @Produce      json
// @Param        id   path      int  true  "Friend id"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/friends/{id} [delete]
func (g *gateway) removeFriendHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	if !g.friendsTableAvailable() {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/friends/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, msgInvalidRequest, http.StatusBadRequest)
		return
	}

	if _, err := g.db.Exec(
		`DELETE FROM friends WHERE id = $1 AND user_id = $2`, id, userID,
	); err != nil {
		g.log.Error("Failed to remove friend", zap.Error(err))
		http.Error(w, "failed to remove friend", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{"status": "ok"})
}
