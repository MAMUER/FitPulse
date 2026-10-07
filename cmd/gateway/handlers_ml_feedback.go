package main

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// mlFeedbackRequest represents the request body for ml feedback
type mlFeedbackRequest struct {
	PredictionID string                 `json:"prediction_id"`
	Feedback     string                 `json:"feedback"`
	UserID       string                 `json:"user_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// mlFeedbackResponse represents the response body for ml feedback
type mlFeedbackResponse struct {
	Status      string    `json:"status"`
	FeedbackID  string    `json:"feedback_id"`
	Timestamp   time.Time `json:"timestamp"`
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

	feedbackID := generateID()
	timestamp := time.Now().UTC()

	g.log.Info("ML feedback received",
		zap.String("prediction_id", req.PredictionID),
		zap.String("feedback_id", feedbackID),
		zap.String("user_id", req.UserID),
	)

	resp := mlFeedbackResponse{
		Status:      "success",
		FeedbackID:  feedbackID,
		Timestamp:   timestamp,
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// generateID generates a simple unique ID for feedback
func generateID() string {
	return "fb-" + time.Now().Format("20060102150405")
}
