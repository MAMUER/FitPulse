package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	biometricpb "github.com/MAMUER/project/api/gen/biometric"
	"github.com/MAMUER/project/internal/middleware"
)

// ========== Biometric Handlers ==========

// @Summary      Proxy biometric integration providers
// @Description  Proxies request to biometric service with authenticated user ID injected
// @Tags         Biometric
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/integrations/providers [get]

func (g *gateway) proxyToBiometricWithUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		g.log.Error("Unauthorized access", zap.String("handler", "proxyToBiometricWithUser"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}

	q := r.URL.Query()
	q.Set("user_id", userID)
	r.URL.RawQuery = q.Encode()

	g.biometricWebhookProxy.ServeHTTP(w, r)
}

// @Summary      Add biometric record
// @Description  Adds a new biometric metric record for the authenticated user
// @Tags         Biometric
// @Accept       json
// @Produce      json
// @Param        request  body  object  required  "Biometric metric data"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/biometrics [post]

func (g *gateway) addBiometricRecordHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		g.log.Error("Unauthorized access", zap.String("handler", "addBiometricRecord"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}

	var req struct {
		MetricType string    `json:"metric_type"`
		Value      float64   `json:"value"`
		Timestamp  time.Time `json:"timestamp"`
		DeviceType string    `json:"device_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.log.Error("Failed to decode add biometric record request", zap.Error(err))
		http.Error(w, "Некорректное тело запроса", http.StatusBadRequest)
		return
	}
	// Валидация
	if req.MetricType == "" || req.Value < 0 {
		g.log.Error("Invalid biometric metric data")
		http.Error(w, "Некорректные данные метрики", http.StatusBadRequest)
		return
	}

	client, err := g.getBiometricClient()
	if err != nil {
		g.log.Error("Failed to get biometric client", zap.Error(err))
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "created"}); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
		http.Error(w, "encodeResponseError", http.StatusInternalServerError)
		return
	}
}

// @Summary      Get biometric records
// @Description  Retrieves biometric records for the authenticated user with optional filtering
// @Tags         Biometric
// @Produce      json
// @Param        metric_type  query  string  false  "Filter by metric type"
// @Param        from         query  string  false  "Start timestamp filter (RFC3339)"
// @Param        to           query  string  false  "End timestamp filter (RFC3339)"
// @Param        limit        query  int     false  "Maximum records (default 100, max 10000)"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/biometrics [get]

func (g *gateway) getBiometricRecordsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		g.log.Error("Unauthorized access", zap.String("handler", "getBiometricRecords"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
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
		g.log.Error("Failed to get biometric client", zap.Error(err))
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

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
		http.Error(w, "encodeResponseError", http.StatusInternalServerError)
		return
	}
}
