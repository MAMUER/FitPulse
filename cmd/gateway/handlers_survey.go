package main

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	userpb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/middleware"
)

// @Summary      Сохранить анкету
// @Description  Сохраняет данные анкеты пользователя
// @Tags         Survey
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/v1/survey [post]
func (g *gateway) saveSurveyHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Survey           map[string]interface{} `json:"survey"`
		SurveyCompleted  bool                   `json:"survey_completed"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, errBadRequest, http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}

	ctx := r.Context()

	completedAt := time.Now()
	if !req.SurveyCompleted {
		completedAt = time.Time{}
	}

	_, err := g.userClient.SaveSurvey(ctx, &userpb.SaveSurveyRequest{
		UserId:           userID,
		Survey:           mustStruct(req.Survey),
		SurveyCompleted:   req.SurveyCompleted,
		SurveyCompletedAt: mustTimestamp(completedAt),
	})
	if err != nil {
		g.log.Error("Failed to save survey", zap.Error(err), zap.String("user_id", userID))
		http.Error(w, "failed to save survey", http.StatusInternalServerError)
		return
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

// @Summary      Загрузить анкету
// @Description  Загружает данные анкеты пользователя
// @Tags         Survey
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/v1/survey [get]
func (g *gateway) loadSurveyHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}

	ctx := r.Context()
	resp, err := g.userClient.LoadSurvey(ctx, &userpb.LoadSurveyRequest{
		UserId: userID,
	})
	if err != nil {
		g.log.Error("Failed to load survey", zap.Error(err), zap.String("user_id", userID))
		http.Error(w, "failed to load survey", http.StatusInternalServerError)
		return
	}

	survey := map[string]interface{}{}
	if resp.Survey != nil {
		survey = resp.Survey.AsMap()
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           resp.Status,
		"survey":           survey,
		"survey_completed": resp.SurveyCompleted,
		"survey_completed_at": resp.SurveyCompletedAt.AsTime().Format(time.RFC3339),
	})
}

func mustStruct(m map[string]interface{}) *structpb.Struct {
	s, err := structpb.NewStruct(m)
	if err != nil {
		return &structpb.Struct{}
	}
	return s
}

func mustTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}
