package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/sanitize"
)

// mlFeedbackRequest represents the request body for ml feedback
type mlFeedbackRequest struct {
	PredictionID string `json:"prediction_id"`
	Feedback     string `json:"feedback"`
	UserID       string `json:"user_id,omitempty"`
	// ClassIndex — индекс класса (0..6), который пользователь подтвердил.
	// Если не задан, берётся classifier_state / metadata.predicted_class.
	ClassIndex *int `json:"class_index,omitempty"`
	// ClassifierState — имя класса («endurance_basic», «recovery», ...).
	ClassifierState string `json:"classifier_state,omitempty"`
	// Features — физиологические признаки, на которых была сделана оценка.
	Features map[string]float64 `json:"features,omitempty"`
	// TargetValue — целевая метка для дельты: 1.0 «класс подтверждён»,
	// 0.0 «класс неверен». По умолчанию 1.0.
	TargetValue *float64               `json:"target_value,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// mlFeedbackResponse represents the response body for ml feedback
type mlFeedbackResponse struct {
	Status       string    `json:"status"`
	FeedbackID   string    `json:"feedback_id"`
	DeltaApplied bool      `json:"delta_applied"`
	DeltaReason  string    `json:"delta_reason,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// classifierStateToIndex — имена классов в порядке обучения модели.
// Должны совпадать с trainingClasses в cmd/classifier.
var classifierStateToIndex = map[string]int{
	"recovery":            0,
	"endurance_basic":     1,
	"endurance_threshold": 2,
	"power_hiit":          3,
	"overtraining":        4,
	"illness":             5,
	"unknown":             6,
}

type deltaRequest struct {
	UserID      string             `json:"user_id"`
	ClassIndex  int                `json:"class_index"`
	TargetValue float64            `json:"target_value"`
	Features    map[string]float64 `json:"features,omitempty"`
}

type deltaResponse struct {
	Status     string    `json:"status"`
	Applied    bool      `json:"applied"`
	TargetUsed float64   `json:"target_used"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// handleMLFeedback handles POST /api/v1/ml/feedback
func (g *gateway) handleMLFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req mlFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.log.Error("Failed to decode ml feedback request", zap.Error(err))
		http.Error(w, errBadRequest, http.StatusBadRequest)
		return
	}

	if req.PredictionID == "" {
		http.Error(w, errBadRequest, http.StatusBadRequest)
		return
	}

	if req.Feedback == "" {
		http.Error(w, errBadRequest, http.StatusBadRequest)
		return
	}

	timestamp := time.Now().UTC()
	feedbackID := generateID()

	g.log.Info("ML feedback received",
		zap.String("prediction_id", sanitize.LogString(req.PredictionID)),
		zap.String("feedback_id", sanitize.LogString(feedbackID)),
		zap.String("user_id", sanitize.LogString(req.UserID)),
	)

	resp := mlFeedbackResponse{
		Status:     "success",
		FeedbackID: feedbackID,
		Timestamp:  timestamp,
	}

	// Дельта обновляется только когда известен пользователь и класс.
	if req.UserID == "" {
		resp.DeltaReason = "user_id is required for personalization"
		g.writeJSON(w, http.StatusOK, resp)
		return
	}

	classIndex, ok := g.resolveFeedbackClassIndex(&req)
	if !ok {
		resp.DeltaReason = "class_index or classifier_state is required"
		g.writeJSON(w, http.StatusOK, resp)
		return
	}

	target := 1.0
	if req.TargetValue != nil {
		target = *req.TargetValue
	}

	if err := g.applyClassifierDelta(r.Context(), req.UserID, classIndex, target, req.Features); err != nil {
		g.log.Warn("Не удалось применить дельту классификатора",
			zap.String("user_id", sanitize.LogString(req.UserID)),
			zap.Error(err))
		resp.DeltaReason = err.Error()
		g.writeJSON(w, http.StatusOK, resp)
		return
	}

	resp.DeltaApplied = true
	g.writeJSON(w, http.StatusOK, resp)
}

// resolveFeedbackClassIndex определяет индекс класса из запроса или метаданных.
func (g *gateway) resolveFeedbackClassIndex(req *mlFeedbackRequest) (int, bool) {
	if req.ClassIndex != nil && *req.ClassIndex >= 0 && *req.ClassIndex < 7 {
		return *req.ClassIndex, true
	}
	if idx, ok := classifierStateToIndex[normalizeState(req.ClassifierState)]; ok {
		return idx, true
	}
	return classIndexFromMetadata(req.Metadata)
}

func normalizeState(state string) string {
	return strings.ToLower(strings.TrimSpace(state))
}

// classIndexFromMetadata достаёт класс из metadata.predicted_class / class_index.
func classIndexFromMetadata(metadata map[string]interface{}) (int, bool) {
	if metadata == nil {
		return 0, false
	}
	if name, ok := metadata["predicted_class"].(string); ok {
		if idx, found := classifierStateToIndex[normalizeState(name)]; found {
			return idx, true
		}
	}
	if idx, ok := toInt(metadata["class_index"]); ok && idx >= 0 && idx < 7 {
		return idx, true
	}
	return 0, false
}

func toInt(v interface{}) (int, bool) {
	switch typed := v.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case float32:
		return int(typed), true
	default:
		return 0, false
	}
}

// applyClassifierDelta пересылает дельту в classifier /classify/delta.
// Это и есть персональное дообучение: Δ_u накапливается из подтверждений
// пользователя и влияет на его следующие оценки.
func (g *gateway) applyClassifierDelta(
	ctx context.Context,
	userID string,
	classIndex int,
	target float64,
	features map[string]float64,
) error {
	body, err := json.Marshal(deltaRequest{
		UserID:      userID,
		ClassIndex:  classIndex,
		TargetValue: target,
		Features:    features,
	})
	if err != nil {
		return fmt.Errorf("marshal delta request: %w", err)
	}

	url := strings.TrimRight(g.classifierURL, "/") + "/classify/delta"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create delta request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("classifier delta request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("classifier returned status %d", resp.StatusCode)
	}

	var dr deltaResponse
	if err := json.NewDecoder(resp.Body).Decode(&dr); err != nil {
		return fmt.Errorf("decode delta response: %w", err)
	}
	if !dr.Applied {
		return errors.New("delta not applied")
	}
	return nil
}

// generateID generates a unique feedback ID.
func generateID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand недоступен — используем время с наносекундами.
		return "fb-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	return "fb-" + hex.EncodeToString(buf)
}
