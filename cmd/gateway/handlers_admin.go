package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	userpb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/middleware"
)

// @Summary      List users
// @Description  Lists all users (admin only)
// @Tags         Admin
// @Produce      json
// @Param        page      query  int  false  "Page number (default 1)"
// @Param        page_size query  int  false  "Items per page (default 20)"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/admin/users [get]

func (g *gateway) adminListUsersHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		g.log.Error("Unauthorized access to admin list users")
		http.Error(w, "Не найдено", http.StatusNotFound)
		return
	}

	page, pageSize := parsePagination(r)
	pageSize32 := safeIntToInt32(pageSize)
	page32 := safeIntToInt32(page)

	resp, err := g.userClient.ListUsers(r.Context(), &userpb.ListUsersRequest{
		RequesterUserId: userID,
		Page:            page32,
		PageSize:        pageSize32,
	})
	if err != nil {
		g.log.Error("Failed to list users", zap.Error(err), zap.String("user_id", userID))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	users := make([]map[string]interface{}, len(resp.Users))
	for i, u := range resp.Users {
		users[i] = map[string]interface{}{
			"id":         u.GetUserId(),
			"email":      u.GetEmail(),
			"full_name":  u.GetFullName(),
			"role":       u.GetRole(),
			"created_at": u.GetCreatedAt(),
			"updated_at": u.GetUpdatedAt(),
		}
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"users":  users,
		"total":  resp.GetTotal(),
	}); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
		http.Error(w, "encodeResponseError", http.StatusInternalServerError)
		return
	}
}

func parsePagination(r *http.Request) (int, int) {
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}
	pageSize := 20
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if val, err := strconv.Atoi(ps); err == nil && val > 0 {
			pageSize = val
		}
	}
	return page, pageSize
}

// @Summary      List invite codes
// @Description  Lists all invite codes (admin only)
// @Tags         Admin
// @Produce      json
// @Param        page      query  int  false  "Page number (default 1)"
// @Param        page_size query  int  false  "Items per page (default 20)"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/admin/invites [get]

func (g *gateway) adminListInvitesHandler(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePagination(r)
	pageSize32 := safeIntToInt32(pageSize)
	page32 := safeIntToInt32(page)

	resp, err := g.userClient.AdminListInvites(r.Context(), &userpb.AdminListInvitesRequest{
		Page:     page32,
		PageSize: pageSize32,
	})
	if err != nil {
		g.log.Error("Failed to list invites", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	invites := make([]map[string]interface{}, len(resp.Invites))
	for i, inv := range resp.Invites {
		invites[i] = map[string]interface{}{
			"code":       inv.GetCode(),
			"role":       inv.GetRole(),
			"specialty":  inv.GetSpecialty(),
			"max_uses":   inv.GetMaxUses(),
			"used_count": inv.GetUsedCount(),
			"is_active":  inv.GetIsActive(),
			"created_at": inv.GetCreatedAt(),
			"invite_url": inv.GetInviteUrl(),
		}
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"invites": invites,
		"page":    page,
		"total":   resp.GetTotal(),
	}); err != nil {
		g.log.Error("Failed to encode invites response", zap.Error(err))
	}
}

// @Summary      Create invite code
// @Description  Creates a new invite code (admin only)
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        request  body  object  required  "Invite creation data"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/v1/admin/invites [post]

func (g *gateway) adminCreateInviteHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role      string `json:"role"`
		Specialty string `json:"specialty"`
		MaxUses   int    `json:"max_uses"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.log.Error("Failed to decode create invite request", zap.Error(err))
		http.Error(w, errBadRequest, http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.AdminCreateInvite(r.Context(), &userpb.AdminCreateInviteRequest{
		Role:      req.Role,
		Specialty: req.Specialty,
		MaxUses:   safeIntToInt32(req.MaxUses),
	})
	if err != nil {
		g.log.Error("Failed to create invite", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"code":       resp.GetCode(),
		"role":       resp.GetRole(),
		"max_uses":   resp.GetMaxUses(),
		"specialty":  resp.GetSpecialty(),
		"invite_url": resp.GetInviteUrl(),
	}); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
	}
}

// @Summary      Revoke invite code
// @Description  Revokes an existing invite code by its value (admin only)
// @Tags         Admin
// @Param        code  path  string  true  "Invite code"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/v1/admin/invites/{code}/revoke [post]

func (g *gateway) adminRevokeInviteHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		g.log.Error("Missing invite code in revoke request")
		http.Error(w, "Код инвайта не указан", http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.AdminRevokeInvite(r.Context(), &userpb.AdminRevokeInviteRequest{
		Code: code,
	})
	if err != nil {
		g.log.Error("Failed to revoke invite", zap.Error(err), zap.String("code", code))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  resp.GetSuccess(),
		"message": resp.GetMessage(),
	}); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
	}
}
