package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/MAMUER/project/internal/chat"
	"github.com/MAMUER/project/internal/middleware"
	userpb "github.com/MAMUER/project/api/gen/user"
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

	answer := chat.FindAnswer(req.Message)
	if g.externalLLM != nil {
		ctx := r.Context()
		profile, err := g.userClient.GetProfile(ctx, &userpb.GetProfileRequest{UserId: userID})
		if err == nil && profile != nil && profile.AiAssistantEnabled {
			llmAnswer, llmErr := g.externalLLM.QueryWithFallback(ctx, userID, req.Message)
			if llmErr == nil && llmAnswer != "" {
				answer = llmAnswer
			}
		}
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"message":   req.Message,
		"answer":    answer,
		"user_id":   userID,
		"timestamp": time.Now().Unix(),
	})
}
