package main

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/chat"
	"github.com/MAMUER/project/internal/middleware"
)

// @Summary      Chat
// @Description  Rule-based FAQ/чат
// @Tags         Chat
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /api/v1/chat [post]
func (g *gateway) chatHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, errBadRequest, http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		http.Error(w, "message обязателен", http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}

	answer, menu := chat.Respond(req.Message)

	resp := map[string]interface{}{
		"status":    "ok",
		"message":   req.Message,
		"answer":    answer,
		"user_id":   userID,
		"timestamp": time.Now().Unix(),
	}
	// When the user asked for a menu, attach the generated day plan so the
	// frontend renders a structured "План питания" block (see web useAI.js).
	if menu != nil {
		resp["diet"] = menu
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		g.log.Error("Failed to encode chat response", zap.Error(err))
	}
}
