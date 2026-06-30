package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	biometricpb "github.com/MAMUER/project/api/gen/biometric"
	"github.com/MAMUER/project/internal/middleware"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ========== Biometric Handlers ==========

func (g *gateway) addBiometricRecordHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}

	var req struct {
		MetricType string    `json:"metric_type"`
		Value      float64   `json:"value"`
		Timestamp  time.Time `json:"timestamp"`
		DeviceType string    `json:"device_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректное тело запроса", http.StatusBadRequest)
		return
	}
	// Валидация
	if req.MetricType == "" || req.Value < 0 {
		http.Error(w, "Некорректные данные метрики", http.StatusBadRequest)
		return
	}

	client, err := g.getBiometricClient()
	if err != nil {
		http.Error(w, "Сервис биометрии временно недоступен", http.StatusServiceUnavailable)
		return
	}

	_, err = client.AddRecord(r.Context(), &biometricpb.AddRecordRequest{
		UserId:     userID,
		MetricType: req.MetricType,
		Value:      req.Value,
		Timestamp:  timestamppb.New(req.Timestamp),
		DeviceType: req.DeviceType,
	})
	if err != nil {
		g.log.Error("Failed to add biometric record", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "created"}); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

func (g *gateway) getBiometricRecordsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}

	metricType := r.URL.Query().Get("metric_type")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	limitStr := r.URL.Query().Get("limit")

	var from, to time.Time
	if fromStr != "" {
		from, _ = time.Parse(time.RFC3339, fromStr)
	}
	if toStr != "" {
		to, _ = time.Parse(time.RFC3339, toStr)
	}
	limitInt := int32(100)
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 10000 {
			limitInt = safeIntToInt32(l)
		}
	}

	client, err := g.getBiometricClient()
	if err != nil {
		http.Error(w, "Сервис биометрии временно недоступен", http.StatusServiceUnavailable)
		return
	}

	resp, err := client.GetRecords(r.Context(), &biometricpb.GetRecordsRequest{
		UserId:     userID,
		MetricType: metricType,
		From:       timestamppb.New(from),
		To:         timestamppb.New(to),
		Limit:      limitInt,
	})
	if err != nil {
		g.log.Error("Failed to get biometric records", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	// Convert protobuf records to JSON-serializable format
	records := make([]map[string]interface{}, len(resp.Records))
	for i, rec := range resp.Records {
		records[i] = map[string]interface{}{
			"id":          rec.Id,
			"user_id":     rec.UserId,
			"metric_type": rec.MetricType,
			"value":       rec.Value,
			"timestamp":   rec.Timestamp.AsTime().Format(time.RFC3339),
			"device_type": rec.DeviceType,
			"created_at":  rec.CreatedAt.AsTime().Format(time.RFC3339),
		}
	}

	response := map[string]interface{}{
		"status":  "ok",
		"records": records,
	}

	// Требование #11: HMAC-SHA256 подпись критического ответа
	if err := middleware.SignAndSendJSON(w, response, g.jwtPublicKeyPEM, g.log.Logger); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}
