package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	userpb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/middleware"
)

func (g *gateway) adminListUsersHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
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
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
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

func (g *gateway) adminListInvitesHandler(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePagination(r)
	pageSize32 := safeIntToInt32(pageSize)
	page32 := safeIntToInt32(page)

	resp, err := g.userClient.AdminListInvites(r.Context(), &userpb.AdminListInvitesRequest{
		Page:     page32,
		PageSize: pageSize32,
	})
	if err != nil {
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

func (g *gateway) adminCreateInviteHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role      string `json:"role"`
		Specialty string `json:"specialty"`
		MaxUses   int    `json:"max_uses"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.AdminCreateInvite(r.Context(), &userpb.AdminCreateInviteRequest{
		Role:      req.Role,
		Specialty: req.Specialty,
		MaxUses:   safeIntToInt32(req.MaxUses),
	})
	if err != nil {
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
		g.log.Error("Failed to encode response", zap.Error(err))
	}
}

func (g *gateway) adminRevokeInviteHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Код инвайта не указан", http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.AdminRevokeInvite(r.Context(), &userpb.AdminRevokeInviteRequest{
		Code: code,
	})
	if err != nil {
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  resp.GetSuccess(),
		"message": resp.GetMessage(),
	}); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
	}
}
