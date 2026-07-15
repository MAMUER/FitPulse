package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	biometricpb "github.com/MAMUER/project/api/gen/biometric"
	"github.com/MAMUER/project/internal/config"
	"github.com/MAMUER/project/internal/db"
	grpctls "github.com/MAMUER/project/internal/grpc"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/metrics"
	"github.com/MAMUER/project/internal/middleware"
	"github.com/MAMUER/project/internal/sanitize"
	"github.com/MAMUER/project/internal/telemetry"
	"github.com/MAMUER/project/internal/validator"
)

// ========== Valid device types ==========

// isValidDeviceType checks if the device type is supported
func isValidDeviceType(dt string) bool {
	switch dt {
	case "fitbit", "garmin", "withings":
		return true
	}
	return false
}

// metricSyncRules returns sync interval rules for a metric type
func metricSyncRules(metricType string) (minMs, maxMs int, name string, ok bool) {
	rules := map[string]struct {
		min, max int
		name     string
	}{
		"heart_rate": {5000, 15000, "heart_rate"},
		"spo2":       {60000, 300000, "spo2"},
		"steps":      {30000, 30000, "steps"},
		"sleep":      {86400000, 86400000, "sleep"},
	}
	r, ok := rules[metricType]
	return r.min, r.max, r.name, ok
}

// ========== Data structures ==========

// Device represents a registered wearable device
type Device struct {
	ID         string    `json:"device_id"`
	UserID     string    `json:"user_id"`
	DeviceType string    `json:"device_type"`
	Token      string    `json:"device_token"`
	CreatedAt  time.Time `json:"created_at"`
}

// DeviceRegisterRequest is the request body for device registration
type DeviceRegisterRequest struct {
	DeviceType string `json:"device_type"`
	UserID     string `json:"user_id"`
}

// IngestRecord represents a single biometric reading from a device
type IngestRecord struct {
	MetricType string    `json:"metric_type"`
	Value      float64   `json:"value"`
	Timestamp  time.Time `json:"timestamp"`
	Quality    string    `json:"quality"`
}

// IngestRequest is the request body for batched data ingestion
type IngestRequest struct {
	DeviceType     string         `json:"device_type"`
	DeviceToken    string         `json:"device_token"`
	SyncIntervalMs int            `json:"sync_interval_ms"`
	Records        []IngestRecord `json:"records"`
}

// IngestStats tracks deduplication and forwarding statistics
type IngestStats struct {
	TotalReceived int `json:"total_received"`
	Duplicates    int `json:"duplicates"`
	Forwarded     int `json:"forwarded"`
	Failed        int `json:"failed"`
}

// ========== Server ==========

type deviceConnector struct {
	db              *sql.DB
	biometricClient biometricpb.BiometricServiceClient
	log             *logger.Logger
}

// ========== Health Check ==========

func (s *deviceConnector) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check database connectivity
	dbOK := true
	if err := s.db.PingContext(r.Context()); err != nil {
		s.log.Warn("Database health check failed", zap.Error(err))
		dbOK = false
	}

	statusCode := http.StatusOK
	overallStatus := "ok"
	if !dbOK {
		statusCode = http.StatusServiceUnavailable
		overallStatus = "degraded"
	}

	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    overallStatus,
		"service":   "device-connector",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"database":  dbOK,
	}); err != nil {
		s.log.Error("Failed to encode health response", zap.Error(err))
	}
}

// ========== Device Registration ==========

func (s *deviceConnector) registerDeviceHandler(w http.ResponseWriter, r *http.Request) {
	var req DeviceRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.log.Warn("Некорректное тело запроса регистрации", zap.Error(err))
		http.Error(w, "Некорректное тело запроса", http.StatusBadRequest)
		return
	}

	// Validate device type
	if req.DeviceType == "" {
		http.Error(w, "device_type обязателен", http.StatusBadRequest)
		return
	}
	if !isValidDeviceType(req.DeviceType) {
		http.Error(w, "Неподдерживаемый тип устройства: "+req.DeviceType, http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		http.Error(w, "user_id обязателен", http.StatusBadRequest)
		return
	}

	// Check if user already has a device registered
	var count int
	checkErr := s.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM devices WHERE user_id = $1`, req.UserID).Scan(&count)
	if checkErr != nil {
		s.log.Error("Failed to check existing devices", zap.Error(checkErr))
		http.Error(w, "Ошибка регистрации устройства", http.StatusInternalServerError)
		return
	}
	if count > 0 {
		http.Error(w, "У вас уже есть подключённое устройство", http.StatusConflict)
		return
	}

	// Generate device ID and auth token
	deviceID := uuid.New().String()
	token := uuid.New().String()

	_, err := s.db.ExecContext(r.Context(), `
		INSERT INTO devices (id, user_id, device_type, token, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, deviceID, req.UserID, req.DeviceType, token, time.Now().UTC())
	if err != nil {
		s.log.Error("Failed to register device", zap.Error(err))
		http.Error(w, "Ошибка регистрации устройства", http.StatusInternalServerError)
		return
	}

	s.log.Info("Device registered",
		zap.String("device_id", sanitize.LogString(deviceID)),
		zap.String("device_type", sanitize.LogString(req.DeviceType)),
		zap.String("user_id", sanitize.LogString(req.UserID)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"device_id":    deviceID,
		"device_type":  req.DeviceType,
		"user_id":      req.UserID,
		"device_token": token,
	}); err != nil {
		s.log.Error("Failed to encode register response", zap.Error(err))
	}
}

// ========== Data Ingestion ==========

func (s *deviceConnector) ingestHandler(w http.ResponseWriter, r *http.Request) {
	deviceID, req, apiErr := s.ingestInputs(r)
	if apiErr != nil {
		http.Error(w, apiErr.Message, apiErr.Code)
		return
	}

	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		s.log.Error("Failed to begin transaction", zap.Error(err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback() }()

	device, err := s.authenticateDevice(r.Context(), deviceID, req.DeviceToken)
	if err != nil {
		s.log.Warn("Device authentication failed",
			zap.String("device_id", sanitize.LogString(deviceID)),
			zap.String("error", sanitize.LogString(err.Error())),
		)
		http.Error(w, "Неверные учётные данные устройства", http.StatusUnauthorized)
		return
	}

	stats, pbRecords, err := s.processIngestRecords(r.Context(), tx, deviceID, device, req)
	if err != nil {
		s.log.Error("Failed to process ingest records", zap.Error(err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		s.log.Error("Failed to commit transaction", zap.Error(err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	if len(pbRecords) > 0 {
		s.forwardRecords(r.Context(), pbRecords, &stats)
	}

	s.log.Info("Ingest completed",
		zap.String("device_id", sanitize.LogString(deviceID)),
		zap.String("device_type", sanitize.LogString(device.DeviceType)),
		zap.Int("total", stats.TotalReceived),
		zap.Int("duplicates", stats.Duplicates),
		zap.Int("forwarded", stats.Forwarded),
		zap.Int("failed", stats.Failed),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		s.log.Error("Failed to encode ingest response", zap.Error(err))
	}
}

type ingestInputError struct {
	Message string
	Code    int
}

func (e ingestInputError) Error() string {
	return e.Message
}

func (s *deviceConnector) ingestInputs(r *http.Request) (string, IngestRequest, *ingestInputError) {
	deviceID := chi.URLParam(r, "device_id")
	if deviceID == "" {
		return "", IngestRequest{}, &ingestInputError{"device_id обязателен", http.StatusBadRequest}
	}

	var req IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return "", IngestRequest{}, &ingestInputError{"Некорректное тело запроса", http.StatusBadRequest}
	}

	if len(req.Records) == 0 {
		return "", IngestRequest{}, &ingestInputError{"Записи не могут быть пустыми", http.StatusBadRequest}
	}

	return deviceID, req, nil
}

func (s *deviceConnector) processIngestRecords(ctx context.Context, tx *sql.Tx, deviceID string, device *Device, req IngestRequest) (IngestStats, []*biometricpb.BiometricRecord, error) {
	stats := IngestStats{TotalReceived: len(req.Records)}
	pbRecords := make([]*biometricpb.BiometricRecord, 0, len(req.Records))

	for _, rec := range req.Records {
		if !s.validateIngestRecord(&rec, &stats) {
			continue
		}

		exists, dupErr := s.isDuplicate(ctx, tx, deviceID, rec)
		if dupErr != nil {
			s.log.Error("Failed to check duplicate", zap.Error(dupErr))
			stats.Failed++
			return stats, nil, dupErr
		}
		if exists {
			stats.Duplicates++
			s.log.Debug("Duplicate record skipped",
				zap.String("device_id", sanitize.LogString(deviceID)),
				zap.String("metric_type", sanitize.LogString(rec.MetricType)),
				zap.Time("timestamp", rec.Timestamp),
			)
			continue
		}

		_, insErr := tx.ExecContext(ctx,
			`INSERT INTO device_ingest_log (id, device_id, metric_type, timestamp, quality) VALUES ($1, $2, $3, $4, $5)`,
			uuid.New().String(), deviceID, rec.MetricType, rec.Timestamp, rec.Quality,
		)
		if insErr != nil {
			s.log.Error("Failed to log ingestion", zap.Error(insErr))
			stats.Failed++
			return stats, nil, insErr
		}

		pbRecords = append(pbRecords, &biometricpb.BiometricRecord{
			UserId:     device.UserID,
			MetricType: rec.MetricType,
			Value:      rec.Value,
			Timestamp:  timestamppb.New(rec.Timestamp),
			DeviceType: device.DeviceType,
		})
	}

	return stats, pbRecords, nil
}

func (s *deviceConnector) validateIngestRecord(rec *IngestRecord, stats *IngestStats) bool {
	if rec.MetricType == "" {
		stats.Failed++
		s.log.Warn("Skipping record with empty metric_type")
		return false
	}
	if rec.Value < 0 {
		stats.Failed++
		s.log.Warn("Skipping record with negative value",
			zap.String("metric_type", sanitize.LogString(rec.MetricType)),
		)
		return false
	}
	if _, _, _, ok := metricSyncRules(rec.MetricType); ok {
		if rec.MetricType == "heart_rate" && (rec.Value < 30 || rec.Value > 220) {
			stats.Failed++
			s.log.Warn("Heart rate out of range", zap.Float64("value", rec.Value))
			return false
		}
		if rec.MetricType == "spo2" && (rec.Value < 70 || rec.Value > 100) {
			stats.Failed++
			s.log.Warn("SpO2 out of valid range", zap.Float64("value", rec.Value))
			return false
		}
	}
	return true
}

func (s *deviceConnector) isDuplicate(ctx context.Context, tx *sql.Tx, deviceID string, rec IngestRecord) (bool, error) {
	var exists bool
	err := tx.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM device_ingest_log WHERE device_id = $1 AND timestamp = $2 AND metric_type = $3)`,
		deviceID, rec.Timestamp, rec.MetricType,
	).Scan(&exists)
	return exists, err
}

func (s *deviceConnector) forwardRecords(ctx context.Context, pbRecords []*biometricpb.BiometricRecord, stats *IngestStats) {
	for _, pbRec := range pbRecords {
		if err := validator.ValidateBiometricRecord(&biometricpb.AddRecordRequest{
			UserId:     pbRec.UserId,
			MetricType: pbRec.MetricType,
			Value:      pbRec.Value,
			Timestamp:  pbRec.Timestamp,
			DeviceType: pbRec.DeviceType,
		}); err != nil {
			s.log.Warn("Record failed validation before forwarding",
				zap.String("metric_type", sanitize.LogString(pbRec.MetricType)),
				zap.String("error", sanitize.LogString(err.Error())),
			)
			stats.Failed++
			continue
		}

		_, err := s.biometricClient.AddRecord(ctx, &biometricpb.AddRecordRequest{
			UserId:     pbRec.UserId,
			MetricType: pbRec.MetricType,
			Value:      pbRec.Value,
			Timestamp:  pbRec.Timestamp,
			DeviceType: pbRec.DeviceType,
		})
		if err != nil {
			st, ok := status.FromError(err)
			errMsg := err.Error()
			if ok {
				errMsg = st.Message()
			}
			s.log.Error("Failed to forward record to biometric-service",
				zap.String("metric_type", sanitize.LogString(pbRec.MetricType)),
				zap.String("error", sanitize.LogString(errMsg)),
			)
			stats.Failed++
			continue
		}
		stats.Forwarded++
	}
}

// ========== Helper Functions ==========

func (s *deviceConnector) authenticateDevice(ctx context.Context, deviceID, token string) (*Device, error) {
	var device Device
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, device_type, token, created_at
		FROM devices
		WHERE id = $1 AND token = $2
	`, deviceID, token).Scan(&device.ID, &device.UserID, &device.DeviceType, &device.Token, &device.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("invalid device credentials")
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	return &device, nil
}

// ========== Database Initialization ==========

// initDatabase creates required tables if they don't exist
func initDatabase(database *sql.DB, log *logger.Logger) error {
	// Devices table — stores registered devices and their auth tokens
	_, err := database.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS devices (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			device_type TEXT NOT NULL,
			token TEXT NOT NULL UNIQUE,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create devices table: %w", err)
	}
	log.Info("Devices table ready")

	// Ingest log table — tracks ingested records for deduplication
	_, err = database.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS device_ingest_log (
			id UUID PRIMARY KEY,
			device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
			metric_type TEXT NOT NULL,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			quality TEXT DEFAULT 'good',
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create device_ingest_log table: %w", err)
	}
	log.Info("Device ingest log table ready")

	// Index for deduplication queries
	_, err = database.ExecContext(context.Background(), `
		CREATE INDEX IF NOT EXISTS idx_ingest_dedup
		ON device_ingest_log (device_id, timestamp, metric_type)
	`)
	if err != nil {
		return fmt.Errorf("failed to create dedup index: %w", err)
	}
	log.Info("Deduplication index ready")

	return nil
}

// ========== Main ==========

func main() {
	log := logger.New("device-connector")
	defer func() {
		if syncErr := log.Sync(); syncErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", syncErr)
		}
	}()

	shutdownTraces := telemetry.InitTracer()
	defer func() {
		if err := shutdownTraces(context.Background()); err != nil {
			log.Warn("Failed to shutdown traces", zap.Error(err))
		}
	}()

	port := config.GetEnv("DEVICE_CONNECTOR_PORT", "8082")

	dbCfg := db.Config{
		Host:     config.GetEnv("DB_HOST"),
		Port:     config.GetEnv("DB_PORT"),
		User:     config.GetEnv("POSTGRES_USER"),
		Password: config.GetEnv("POSTGRES_PASSWORD"),
		DBName:   config.GetEnv("POSTGRES_DB"),
		SSLMode:  config.GetEnv("DB_SSLMODE"),
	}

	biometricServiceAddr := config.GetEnv("BIOMETRIC_SERVICE_ADDR", "localhost:50052")

	// Connect to PostgreSQL
	database, err := db.NewConnection(dbCfg)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer func() {
		if closeErr := database.Close(); closeErr != nil {
			log.Error("Failed to close database connection", zap.Error(closeErr))
		}
	}()

	// Initialize tables
	if initErr := initDatabase(database, log); initErr != nil {
		log.Fatal("Failed to initialize database", zap.Error(initErr))
	}

	// Connect to biometric-service via gRPC
	var dialOpts []grpc.DialOption
	dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(metrics.UnaryClientInterceptor("device-connector")))
	dialOpts = append(dialOpts, telemetry.ClientHandlerOption())
	dialOpts = append(dialOpts, grpc.WithDefaultCallOptions(grpc.WaitForReady(true), grpc.MaxCallRecvMsgSize(10<<20)))
	if tlsCreds, err2 := grpctls.GetClientTLSCredentials(); err2 == nil && tlsCreds != nil {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(tlsCreds))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	biometricConn, err := grpc.NewClient(biometricServiceAddr, dialOpts...)
	if err != nil {
		log.Fatal("Failed to connect to biometric service", zap.Error(err))
	}
	defer func() {
		if closeErr := biometricConn.Close(); closeErr != nil {
			log.Error("Failed to close biometric service connection", zap.Error(closeErr))
		}
	}()

	// Create server
	s := &deviceConnector{
		db:              database,
		biometricClient: biometricpb.NewBiometricServiceClient(biometricConn),
		log:             log,
	}

	// Setup router
	r := chi.NewRouter()

	// Apply middleware
	r.Use(middleware.CorrelationIDHTTP)
	r.Use(middleware.RequestID)
	r.Use(middleware.LoggingMiddleware(log.Logger, nil, nil, nil))

	// Health check (public)
	r.Get("/health", s.healthHandler)

	// Device management routes
	r.Post("/api/v1/devices/register", s.registerDeviceHandler)
	r.Post("/api/v1/devices/{device_id}/ingest", s.ingestHandler)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Info("Device connector starting",
		zap.String("port", port),
		zap.String("biometric_service", biometricServiceAddr),
	)

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal("Failed to start server", zap.Error(err))
	}
}
