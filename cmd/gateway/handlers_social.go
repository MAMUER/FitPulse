package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/MAMUER/project/internal/middleware"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

const (
	// msgDBUnavailable — user-facing message when the database is not configured.
	msgDBUnavailable = "database unavailable"
	// msgDefaultName — fallback display name for a user without a set name.
	msgDefaultName = "Участник"
)

// ---------------------------------------------------------------------------
// Leaderboard
// ---------------------------------------------------------------------------

// @Summary      Leaderboard
// @Description  Returns the top users by points.
// @Tags         leaderboard
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/leaderboard [get]
func (g *gateway) getLeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	if g.db == nil {
		writeJSON(w, map[string]interface{}{"status": "ok", "leaderboard": []interface{}{}})
		return
	}
	rows, err := g.db.Query(
		`SELECT user_id, display_name, points, xp, workouts
		 FROM leaderboard
		 ORDER BY points DESC, xp DESC
		 LIMIT 50`,
	)
	if err != nil {
		g.log.Error("Failed to load leaderboard", zap.Error(err))
		http.Error(w, "failed to load leaderboard", http.StatusInternalServerError)
		return
	}
	defer func() { _ = rows.Close() }()

	entries := []map[string]interface{}{}
	for rows.Next() {
		var (
			id, name             string
			points, xp, workouts int
		)
		if rows.Scan(&id, &name, &points, &xp, &workouts) != nil {
			continue
		}
		entries = append(entries, map[string]interface{}{
			"user_id":      id,
			"display_name": name,
			"points":       points,
			"xp":           xp,
			"workouts":     workouts,
			"me":           id == userID,
		})
	}
	if err := rows.Err(); err != nil {
		g.log.Error("Failed to read leaderboard", zap.Error(err))
		http.Error(w, "failed to read leaderboard", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{"status": "ok", "leaderboard": entries})
}

// @Summary      Sync my leaderboard stats
// @Description  Upserts the caller's display name and score.
// @Tags         leaderboard
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/leaderboard/me [put]
func (g *gateway) syncLeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	if g.db == nil {
		http.Error(w, msgDBUnavailable, http.StatusServiceUnavailable)
		return
	}

	var req struct {
		DisplayName string `json:"display_name"`
		Points      int    `json:"points"`
		XP          int    `json:"xp"`
		Workouts    int    `json:"workouts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, msgInvalidRequest, http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(req.DisplayName)
	if name == "" {
		name = msgDefaultName
	}
	if req.Points < 0 {
		req.Points = 0
	}
	if req.XP < 0 {
		req.XP = 0
	}
	if req.Workouts < 0 {
		req.Workouts = 0
	}

	_, err := g.db.Exec(
		`INSERT INTO leaderboard (user_id, display_name, points, xp, workouts, updated_at)
		 VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
		 ON CONFLICT (user_id) DO UPDATE SET
		   display_name = EXCLUDED.display_name,
		   points       = EXCLUDED.points,
		   xp           = EXCLUDED.xp,
		   workouts     = EXCLUDED.workouts,
		   updated_at   = CURRENT_TIMESTAMP`,
		userID, name, req.Points, req.XP, req.Workouts,
	)
	if err != nil {
		g.log.Error("Failed to sync leaderboard", zap.Error(err))
		http.Error(w, "failed to sync leaderboard", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{"status": "ok"})
}

// @Summary      Search users by display name
// @Tags         users
// @Produce      json
// @Param        q   query     string  true  "Name substring"
// @Success      200 {object}  map[string]interface{}
// @Router       /api/v1/users/search [get]
func (g *gateway) searchUsersHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	if g.db == nil {
		writeJSON(w, map[string]interface{}{"status": "ok", "users": []interface{}{}})
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 2 {
		writeJSON(w, map[string]interface{}{"status": "ok", "users": []interface{}{}})
		return
	}

	rows, err := g.db.Query(
		`SELECT user_id, display_name, points
		 FROM leaderboard
		 WHERE display_name ILIKE '%' || $1 || '%' AND user_id <> $2
		 ORDER BY points DESC
		 LIMIT 20`,
		q, userID,
	)
	if err != nil {
		g.log.Error("Failed to search users", zap.Error(err))
		http.Error(w, "failed to search users", http.StatusInternalServerError)
		return
	}
	defer func() { _ = rows.Close() }()

	users := []map[string]interface{}{}
	for rows.Next() {
		var (
			id, name string
			points   int
		)
		if rows.Scan(&id, &name, &points) != nil {
			continue
		}
		users = append(users, map[string]interface{}{
			"user_id":      id,
			"display_name": name,
			"points":       points,
		})
	}
	if err := rows.Err(); err != nil {
		g.log.Error("Failed to read user search", zap.Error(err))
		http.Error(w, "failed to read user search", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{"status": "ok", "users": users})
}

// ---------------------------------------------------------------------------
// Friend requests
// ---------------------------------------------------------------------------

// @Summary      Send a friend request
// @Tags         friends
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/friends/request [post]
func (g *gateway) sendFriendRequestHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	if g.db == nil {
		http.Error(w, msgDBUnavailable, http.StatusServiceUnavailable)
		return
	}

	var req struct {
		ToUserID string `json:"to_user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, msgInvalidRequest, http.StatusBadRequest)
		return
	}
	to := strings.TrimSpace(req.ToUserID)
	if to == "" || to == userID {
		http.Error(w, msgInvalidRequest, http.StatusBadRequest)
		return
	}

	_, err := g.db.Exec(
		`INSERT INTO friend_requests (from_user_id, to_user_id, status)
		 VALUES ($1, $2, 'pending')
		 ON CONFLICT (from_user_id, to_user_id) DO NOTHING`,
		userID, to,
	)
	if err != nil {
		g.log.Error("Failed to send friend request", zap.Error(err))
		http.Error(w, "failed to send friend request", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{"status": "ok"})
}

// @Summary      List incoming friend requests
// @Tags         friends
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/friends/requests [get]
func (g *gateway) listFriendRequestsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	if g.db == nil {
		writeJSON(w, map[string]interface{}{"status": "ok", "requests": []interface{}{}})
		return
	}

	rows, err := g.db.Query(
		`SELECT fr.id, fr.from_user_id, COALESCE(l.display_name, '') AS display_name
		 FROM friend_requests fr
		 LEFT JOIN leaderboard l ON l.user_id = fr.from_user_id
		 WHERE fr.to_user_id = $1 AND fr.status = 'pending'
		 ORDER BY fr.created_at DESC`,
		userID,
	)
	if err != nil {
		g.log.Error("Failed to list friend requests", zap.Error(err))
		http.Error(w, "failed to list friend requests", http.StatusInternalServerError)
		return
	}
	defer func() { _ = rows.Close() }()

	requests := []map[string]interface{}{}
	for rows.Next() {
		var (
			id          int64
			fromID      string
			displayName string
		)
		if rows.Scan(&id, &fromID, &displayName) != nil {
			continue
		}
		name := displayName
		if name == "" {
			name = msgDefaultName
		}
		requests = append(requests, map[string]interface{}{
			"id":           id,
			"from_user_id": fromID,
			"display_name": name,
		})
	}
	if err := rows.Err(); err != nil {
		g.log.Error("Failed to read friend requests", zap.Error(err))
		http.Error(w, "failed to read friend requests", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{"status": "ok", "requests": requests})
}

// @Summary      Accept or decline a friend request
// @Tags         friends
// @Produce      json
// @Param        id      path  int     true  "Request id"
// @Param        action  path  string  true  "accept | decline"
// @Success      200     {object}  map[string]interface{}
// @Router       /api/v1/friends/requests/{id}/{action} [post]
func (g *gateway) respondFriendRequestHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	if g.db == nil {
		http.Error(w, msgDBUnavailable, http.StatusServiceUnavailable)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, msgInvalidRequest, http.StatusBadRequest)
		return
	}
	action := chi.URLParam(r, "action")
	if action != "accept" && action != "decline" {
		http.Error(w, msgInvalidRequest, http.StatusBadRequest)
		return
	}

	var fromID, currentStatus string
	err = g.db.QueryRow(
		`SELECT from_user_id, status FROM friend_requests WHERE id = $1 AND to_user_id = $2`,
		id, userID,
	).Scan(&fromID, &currentStatus)
	if err == sql.ErrNoRows {
		http.Error(w, "заявка не найдена", http.StatusNotFound)
		return
	}
	if err != nil {
		g.log.Error("Failed to load friend request", zap.Error(err))
		http.Error(w, "failed to load friend request", http.StatusInternalServerError)
		return
	}
	if currentStatus != "pending" {
		writeJSON(w, map[string]interface{}{"status": "ok"})
		return
	}

	newStatus := "declined"
	if action == "accept" {
		newStatus = "accepted"
	}
	if _, err := g.db.Exec(
		`UPDATE friend_requests SET status = $1 WHERE id = $2`, newStatus, id,
	); err != nil {
		g.log.Error("Failed to update friend request", zap.Error(err))
		http.Error(w, "failed to update friend request", http.StatusInternalServerError)
		return
	}

	if action == "accept" {
		g.linkFriends(fromID, userID)
		g.linkFriends(userID, fromID)
	}
	writeJSON(w, map[string]interface{}{"status": "ok"})
}

// linkFriends adds friendID to userID's friends list if not already present.
func (g *gateway) linkFriends(userID, friendID string) {
	if userID == "" || friendID == "" || userID == friendID {
		return
	}
	var name string
	// Prefer the friend's display name from the leaderboard.
	if err := g.db.QueryRow(
		`SELECT display_name FROM leaderboard WHERE user_id = $1`, friendID,
	).Scan(&name); err != nil || strings.TrimSpace(name) == "" {
		name = msgDefaultName
	}
	var exists bool
	if err := g.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM friends WHERE user_id = $1 AND friend_user_id = $2)`,
		userID, friendID,
	).Scan(&exists); err != nil || exists {
		return
	}
	if _, err := g.db.Exec(
		`INSERT INTO friends (user_id, friend_name, friend_user_id, status)
		 VALUES ($1, $2, $3, 'offline')`,
		userID, name, sql.NullString{String: friendID, Valid: true},
	); err != nil {
		g.log.Error("Failed to link friends", zap.Error(err))
	}
}
