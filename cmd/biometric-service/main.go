package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/MAMUER/project/api/gen/biometric"
	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/config"
	"github.com/MAMUER/project/internal/db"
	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/domain/service"
	grpctls "github.com/MAMUER/project/internal/grpc"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/metrics"
	"github.com/MAMUER/project/internal/middleware"
	"github.com/MAMUER/project/internal/queue"
	"github.com/MAMUER/project/internal/repository/pgx"
	_ "github.com/MAMUER/project/internal/repository/postgres" // registers PostgreSQL driver
	"github.com/MAMUER/project/internal/telemetry"
	"github.com/MAMUER/project/internal/validator"
	"github.com/MAMUER/project/internal/webhook"
)

const (
	errRecordNotFound = "record not found"
	serviceName       = "biometric-service"
)

type biometricServer struct {
	pb.UnimplementedBiometricServiceServer
	db           *sql.DB
	biometricSvc service.BiometricService
	log          *logger.Logger
	rabbitQueue  queue.Publisher
}

func safeIntToInt32(v int) int32 {
	if v > 2147483647 {
		return 2147483647
	}
	if v < -2147483648 {
		return -2147483648
	}
	return int32(v)
}

func (s *biometricServer) AddRecord(ctx context.Context, req *pb.AddRecordRequest) (*pb.AddRecordResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	start := time.Now()
	s.log.Info("BIOMETRIC_DATA_RECEIVED",
		zap.String("action", "BIOMETRIC_DATA_RECEIVED"),
		zap.String("user_id", req.UserId),
		zap.String("metric_type", req.MetricType),
		zap.Float64("value", req.Value),
	)

	if err := validator.ValidateBiometricRequest(req); err != nil {
		s.log.Warn("Invalid biometric request", zap.Error(err))
		return nil, err
	}

	if s.biometricSvc != nil {
		record := &entity.BiometricRecord{
			ID:         uuid.New().String(),
			UserID:     req.UserId,
			MetricType: req.MetricType,
			Value:      req.Value,
			Timestamp:  req.Timestamp.AsTime(),
			DeviceType: req.DeviceType,
			Source:     "unknown",
		}

		stored, err := s.biometricSvc.AddRecord(ctx, record)
		if err != nil {
			s.log.Error("Failed to add record", zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to insert record")
		}

		lag := time.Since(start).Seconds()
		metrics.BiometricSyncLagSeconds.WithLabelValues(req.DeviceType, "default").Set(lag)

		event := map[string]interface{}{
			"user_id":     req.UserId,
			"metric_type": req.MetricType,
			"value":       req.Value,
			"timestamp":   record.Timestamp,
		}

		if s.rabbitQueue != nil {
			if err := s.rabbitQueue.Publish(ctx, event); err != nil {
				s.log.Error("Failed to publish to queue", zap.Error(err))
				return nil, status.Error(codes.Internal, "failed to queue event")
			}
		}

		return &pb.AddRecordResponse{Id: stored.ID}, nil
	}

	var userExists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", req.UserId).Scan(&userExists)
	if err != nil {
		s.log.Error("Failed to check user existence", zap.Error(err), zap.String("user_id", req.UserId))
		return nil, status.Error(codes.Internal, "failed to verify user")
	}
	if !userExists {
		s.log.Warn("User not found", zap.String("user_id", req.UserId))
		return nil, status.Error(codes.NotFound, "user not found")
	}

	newID := uuid.New().String()
	timestamp := req.Timestamp.AsTime()

	var storedID string
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO biometric_data (id, user_id, metric_type, value, timestamp, device_type, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, metric_type, timestamp, source) DO UPDATE SET id = biometric_data.id
		RETURNING id
	`, newID, req.UserId, req.MetricType, req.Value, timestamp, req.DeviceType, "unknown").Scan(&storedID)
	if err != nil {
		s.log.Error("Failed to insert or fetch record id", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to insert record")
	}

	lag := time.Since(start).Seconds()
	metrics.BiometricSyncLagSeconds.WithLabelValues(req.DeviceType, "default").Set(lag)

	event := map[string]interface{}{
		"user_id":     req.UserId,
		"metric_type": req.MetricType,
		"value":       req.Value,
		"timestamp":   timestamp,
	}

	if s.rabbitQueue != nil {
		if err := s.rabbitQueue.Publish(ctx, event); err != nil {
			s.log.Error("Failed to publish to queue", zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to queue event")
		}
	}

	return &pb.AddRecordResponse{Id: storedID}, nil
}

func (s *biometricServer) BatchAddRecords(ctx context.Context, req *pb.BatchAddRecordsRequest) (*pb.BatchAddRecordsResponse, error) {
	if err := validateBatchRequest(req); err != nil {
		return nil, err
	}
	if err := s.checkUserExists(ctx, req.UserId); err != nil {
		return nil, err
	}

	if s.biometricSvc != nil {
		records := make([]*entity.BiometricRecord, 0, len(req.Records))
		for _, rec := range req.Records {
			records = append(records, &entity.BiometricRecord{
				ID:         uuid.New().String(),
				UserID:     req.UserId,
				MetricType: rec.MetricType,
				Value:      rec.Value,
				Timestamp:  rec.Timestamp.AsTime(),
				DeviceType: rec.DeviceType,
				Source:     "unknown",
			})
		}

		inserted, err := s.biometricSvc.BatchAddRecords(ctx, records)
		if err != nil {
			s.log.Error("Failed to batch add records", zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to save records")
		}
		return &pb.BatchAddRecordsResponse{Count: safeIntToInt32(inserted)}, nil
	}

	if err := validateRecords(ctx, req.Records); err != nil {
		return nil, err
	}
	inserted, err := s.insertRecords(ctx, req)
	if err != nil {
		return nil, err
	}
	return &pb.BatchAddRecordsResponse{Count: safeIntToInt32(inserted)}, nil
}

func validateBatchRequest(req *pb.BatchAddRecordsRequest) error {
	if req.UserId == "" {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}
	if len(req.Records) == 0 {
		return status.Error(codes.InvalidArgument, "records cannot be empty")
	}
	return nil
}

func (s *biometricServer) checkUserExists(ctx context.Context, userID string) error {
	var userExists bool
	if err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&userExists); err != nil {
		s.log.Error("Failed to check user existence", zap.Error(err), zap.String("user_id", userID))
		return status.Error(codes.Internal, "failed to verify user")
	}
	if !userExists {
		return status.Error(codes.NotFound, "user not found")
	}
	return nil
}

func validateRecords(ctx context.Context, records []*pb.AddRecordRequest) error {
	for i, rec := range records {
		if err := ctx.Err(); err != nil {
			return status.Error(codes.Canceled, "request canceled")
		}
		if err := validator.ValidateBiometricRecord(rec); err != nil {
			return status.Errorf(codes.InvalidArgument, "record[%d]: %v", i, err)
		}
	}
	return nil
}

func (s *biometricServer) insertRecords(ctx context.Context, req *pb.BatchAddRecordsRequest) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.log.Error("Failed to begin transaction", zap.Error(err))
		return 0, status.Error(codes.Internal, "database error")
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const query = `INSERT INTO biometric_data (id, user_id, metric_type, value, timestamp, device_type, source, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
			ON CONFLICT (user_id, metric_type, timestamp, source) DO NOTHING`

	inserted := 0
	for _, rec := range req.Records {
		if err := ctx.Err(); err != nil {
			_ = tx.Rollback()
			return 0, status.Error(codes.Canceled, "request canceled")
		}

		id := uuid.New().String()
		ts := rec.Timestamp.AsTime()
		if ts.IsZero() {
			ts = time.Now()
		}

		result, err := tx.ExecContext(ctx, query,
			id, req.UserId, rec.MetricType, rec.Value, ts, rec.DeviceType, "unknown",
		)
		if err != nil {
			_ = tx.Rollback()
			s.log.Error("Failed to insert biometric record",
				zap.Error(err),
				zap.String("metric_type", rec.MetricType),
			)
			return 0, status.Error(codes.Internal, "failed to save records")
		}
		if n, _ := result.RowsAffected(); n > 0 {
			inserted++
		}
	}

	if err := tx.Commit(); err != nil {
		s.log.Error("Failed to commit transaction", zap.Error(err))
		return 0, status.Error(codes.Internal, "database commit error")
	}
	return inserted, nil
}

type recordsQuery struct {
	query string
	args  []interface{}
}

func (s *biometricServer) buildGetRecordsQuery(req *pb.GetRecordsRequest) recordsQuery {
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 100
	}
	if limit > 10000 {
		limit = 10000
	}
	offset := int(req.Offset)
	if offset < 0 {
		offset = 0
	}

	baseQuery := `SELECT id, user_id, metric_type, value, timestamp, device_type, created_at FROM biometric_data WHERE user_id = $1 AND metric_type = $2`

	limitOffsetClause := fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	switch {
	case req.From == nil && req.To == nil:
		return recordsQuery{
			query: baseQuery + ` ORDER BY timestamp DESC` + limitOffsetClause,
			args:  []interface{}{req.UserId, req.MetricType},
		}
	case req.From == nil:
		to := req.To.AsTime()
		return recordsQuery{
			query: baseQuery + ` AND timestamp <= $3 ORDER BY timestamp DESC` + limitOffsetClause,
			args:  []interface{}{req.UserId, req.MetricType, to},
		}
	case req.To == nil:
		from := req.From.AsTime()
		return recordsQuery{
			query: baseQuery + ` AND timestamp >= $3 ORDER BY timestamp DESC` + limitOffsetClause,
			args:  []interface{}{req.UserId, req.MetricType, from},
		}
	default:
		from := req.From.AsTime()
		to := req.To.AsTime()
		return recordsQuery{
			query: baseQuery + ` AND timestamp >= $3 AND timestamp <= $4 ORDER BY timestamp DESC` + limitOffsetClause,
			args:  []interface{}{req.UserId, req.MetricType, from, to},
		}
	}
}

func (s *biometricServer) GetRecords(ctx context.Context, req *pb.GetRecordsRequest) (*pb.GetRecordsResponse, error) {
	s.log.Debug("GetRecords",
		zap.String("user_id", req.UserId),
		zap.String("metric_type", req.MetricType),
	)

	if req.From != nil && req.To != nil && req.From.AsTime().After(req.To.AsTime()) {
		return nil, status.Error(codes.InvalidArgument, "from cannot be after to")
	}

	if s.biometricSvc != nil {
		limit := int(req.Limit)
		if limit <= 0 {
			limit = 100
		}
		if limit > 10000 {
			limit = 10000
		}

		records, err := s.biometricSvc.GetRecords(ctx, req.UserId, req.MetricType, limit)
		if err != nil {
			s.log.Error("Failed to query records", zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to query records")
		}

		pbRecords := make([]*pb.BiometricRecord, 0, len(records))
		for _, r := range records {
			pbRecords = append(pbRecords, &pb.BiometricRecord{
				Id:         r.ID,
				UserId:     r.UserID,
				MetricType: r.MetricType,
				Value:      r.Value,
				Timestamp:  timestamppb.New(r.Timestamp),
				DeviceType: r.DeviceType,
				CreatedAt:  timestamppb.New(r.CreatedAt),
			})
		}

		s.log.Debug("GetRecords fetched", zap.Int("count", len(pbRecords)))
		return &pb.GetRecordsResponse{Records: pbRecords}, nil
	}

	q := s.buildGetRecordsQuery(req)
	s.log.Debug("GetRecords query",
		zap.String("query", q.query),
		zap.Any("args", q.args),
	)
	rows, err := s.db.QueryContext(ctx, q.query, q.args...)
	if err != nil {
		s.log.Error("Failed to query records", zap.Error(err), zap.String("query", q.query))
		return nil, status.Error(codes.Internal, "failed to query records")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.log.Error("Failed to close rows", zap.Error(closeErr))
		}
	}()

	records := make([]*pb.BiometricRecord, 0)
	for rows.Next() {
		var record pb.BiometricRecord
		var timestamp, createdAt time.Time
		if err := rows.Scan(&record.Id, &record.UserId, &record.MetricType, &record.Value,
			&timestamp, &record.DeviceType, &createdAt); err != nil {
			s.log.Error("Failed to scan row", zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to read biometric data")
		}
		record.Timestamp = timestamppb.New(timestamp)
		record.CreatedAt = timestamppb.New(createdAt)
		records = append(records, &record)
	}

	if err := rows.Err(); err != nil {
		s.log.Error("Row iteration error", zap.Error(err))
		return nil, status.Error(codes.Internal, "error reading records")
	}

	s.log.Debug("GetRecords fetched", zap.Int("count", len(records)))
	return &pb.GetRecordsResponse{Records: records}, nil
}

func (s *biometricServer) GetLatest(ctx context.Context, req *pb.GetLatestRequest) (*pb.BiometricRecord, error) {
	s.log.Debug("GetLatest",
		zap.String("user_id", req.UserId),
		zap.String("metric_type", req.MetricType),
	)

	if s.biometricSvc != nil {
		record, err := s.biometricSvc.GetLatest(ctx, req.UserId, req.MetricType)
		if err != nil {
			s.log.Error("Failed to query latest record", zap.Error(err))
			if apperrors.IsNotFound(err) {
				return nil, status.Error(codes.NotFound, "no records found")
			}
			return nil, status.Error(codes.Internal, "failed to query record")
		}

		return &pb.BiometricRecord{
			Id:         record.ID,
			UserId:     record.UserID,
			MetricType: record.MetricType,
			Value:      record.Value,
			Timestamp:  timestamppb.New(record.Timestamp),
			DeviceType: record.DeviceType,
			CreatedAt:  timestamppb.New(record.CreatedAt),
		}, nil
	}

	var record pb.BiometricRecord
	var timestamp, createdAt time.Time

	err := s.db.QueryRowContext(ctx, `
        SELECT id, user_id, metric_type, value, timestamp, device_type, created_at
        FROM biometric_data
        WHERE user_id = $1 AND metric_type = $2
        ORDER BY timestamp DESC
        LIMIT 1
    `, req.UserId, req.MetricType).Scan(
		&record.Id, &record.UserId, &record.MetricType, &record.Value,
		&timestamp, &record.DeviceType, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "no records found")
	}
	if err != nil {
		s.log.Error("Failed to query latest record", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to query record")
	}

	record.Timestamp = timestamppb.New(timestamp)
	record.CreatedAt = timestamppb.New(createdAt)

	return &record, nil
}

func (s *biometricServer) UpdateRecord(ctx context.Context, req *pb.UpdateRecordRequest) (*pb.BiometricRecord, error) {
	s.log.Info("BIOMETRIC_UPDATE",
		zap.String("action", "BIOMETRIC_UPDATE"),
		zap.String("id", req.Id),
	)

	if s.biometricSvc != nil {
		record := &entity.BiometricRecord{
			ID:         req.Id,
			Value:      req.Value,
			Timestamp:  req.Timestamp.AsTime(),
			DeviceType: req.DeviceType,
		}

		stored, err := s.biometricSvc.UpdateRecord(ctx, record)
		if err != nil {
			s.log.Error("Failed to update record", zap.Error(err))
			if apperrors.IsNotFound(err) {
				return nil, status.Error(codes.NotFound, errRecordNotFound)
			}
			return nil, status.Error(codes.Internal, "failed to update record")
		}

		s.log.Info("BIOMETRIC_UPDATED",
			zap.String("action", "BIOMETRIC_UPDATED"),
			zap.String("id", stored.ID),
		)

		return &pb.BiometricRecord{
			Id:         stored.ID,
			UserId:     stored.UserID,
			MetricType: stored.MetricType,
			Value:      stored.Value,
			Timestamp:  timestamppb.New(stored.Timestamp),
			DeviceType: stored.DeviceType,
			CreatedAt:  timestamppb.New(stored.CreatedAt),
		}, nil
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if req.Value < 0 {
		return nil, status.Error(codes.InvalidArgument, "value cannot be negative")
	}

	ts := req.Timestamp.AsTime()
	if ts.IsZero() {
		ts = time.Now()
	}

	var record pb.BiometricRecord
	var timestamp, createdAt time.Time
	err := s.db.QueryRowContext(ctx, `
		UPDATE biometric_data
		SET value = $1, timestamp = $2, device_type = $3
		WHERE id = $4
		RETURNING id, user_id, metric_type, value, timestamp, device_type, created_at
	`, req.Value, ts, req.DeviceType, req.Id).Scan(
		&record.Id, &record.UserId, &record.MetricType, &record.Value,
		&timestamp, &record.DeviceType, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, errRecordNotFound)
	}
	if err != nil {
		s.log.Error("Failed to update record", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update record")
	}

	record.Timestamp = timestamppb.New(timestamp)
	record.CreatedAt = timestamppb.New(createdAt)

	s.log.Info("BIOMETRIC_UPDATED",
		zap.String("action", "BIOMETRIC_UPDATED"),
		zap.String("id", record.Id),
	)

	return &record, nil
}

func (s *biometricServer) DeleteRecord(ctx context.Context, req *pb.DeleteRecordRequest) (*pb.DeleteRecordResponse, error) {
	s.log.Info("BIOMETRIC_DELETE",
		zap.String("action", "BIOMETRIC_DELETE"),
		zap.String("id", req.Id),
	)

	if s.biometricSvc != nil {
		if err := s.biometricSvc.DeleteRecord(ctx, req.Id); err != nil {
			s.log.Error("Failed to delete record", zap.Error(err))
			if apperrors.IsNotFound(err) {
				return nil, status.Error(codes.NotFound, errRecordNotFound)
			}
			return nil, status.Error(codes.Internal, "failed to delete record")
		}

		s.log.Info("BIOMETRIC_DELETED",
			zap.String("action", "BIOMETRIC_DELETED"),
			zap.String("id", req.Id),
		)

		return &pb.DeleteRecordResponse{Deleted: true}, nil
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	result, err := s.db.ExecContext(ctx, `DELETE FROM biometric_data WHERE id = $1`, req.Id)
	if err != nil {
		s.log.Error("Failed to delete record", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to delete record")
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, status.Error(codes.NotFound, errRecordNotFound)
	}

	s.log.Info("BIOMETRIC_DELETED",
		zap.String("action", "BIOMETRIC_DELETED"),
		zap.String("id", req.Id),
	)

	return &pb.DeleteRecordResponse{Deleted: true}, nil
}

func createGRPCServer(log *logger.Logger, jwtPublicKeyPEM string) *grpc.Server {
	serverOpts := []grpc.ServerOption{grpc.ChainUnaryInterceptor(
		middleware.CorrelationIDGRPC(),
		middleware.GRPCAuthInterceptor(jwtPublicKeyPEM, log.Logger),
		metrics.UnaryServerInterceptor(serviceName),
	), telemetry.ServerHandlerOption()}
	return grpctls.NewServer(serverOpts...)
}

func createMetricsServer(metricsPort string) *http.Server {
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	return &http.Server{
		Addr:              ":" + metricsPort,
		Handler:           metricsMux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func startMetricsServer(srv *http.Server, log *logger.Logger) {
	go func() {
		log.Info("Metrics HTTP server starting", zap.String("port", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !strings.Contains(err.Error(), "Server closed") {
			log.Error("Metrics server failed", zap.Error(err))
		}
	}()
}

func startHealthCheckLoop(db *sql.DB, rq queue.Publisher, hs *health.Server, log *logger.Logger) {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			checkHealth(db, rq, hs, log)
		}
	}()
}

func setupGracefulShutdown(log *logger.Logger, grpcServer *grpc.Server, metricsSrv *http.Server, database *sql.DB, pgxPool *pgxpool.Pool, rabbitQueue queue.Publisher, webhookSrv *webhook.Server) context.Context {
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-ctx.Done()
		log.Info("Shutting down gRPC server...")
		grpcServer.GracefulStop()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
			log.Error("Failed to shutdown metrics server", zap.Error(err))
		}
		if webhookSrv != nil {
			if err := webhookSrv.Stop(shutdownCtx); err != nil {
				log.Error("Failed to shutdown webhook server", zap.Error(err))
			}
		}
		if rabbitQueue != nil {
			if err := rabbitQueue.Close(); err != nil {
				log.Error("Failed to close rabbit queue", zap.Error(err))
			}
		}
		if closeErr := database.Close(); closeErr != nil {
			log.Error("Failed to close database", zap.Error(closeErr))
		}
		pgxPool.Close()
		stop()
	}()

	return ctx
}

func main() {
	log := logger.New(serviceName)

	shutdownTraces := telemetry.InitTracer()
	defer func() {
		if err := shutdownTraces(context.Background()); err != nil {
			log.Warn("Failed to shutdown traces", zap.Error(err))
		}
	}()

	port, metricsPort, webhookPort, jwtPublicKeyPEM, dbCfg := loadBiometricConfig()
	database, pgxPool, biometricSvc, rabbitQueue := initBiometricDependencies(dbCfg, log)
	grpcServer := createBiometricGRPCServer(log, jwtPublicKeyPEM)
	metricsSrv := buildMetricsServer(metricsPort, log)
	webhookSrv := startWebhookServer(webhookPort, database, log)

	registerBiometricGRPC(grpcServer, database, biometricSvc, rabbitQueue, log)
	startHealthChecks(database, rabbitQueue, log)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal("Failed to listen", zap.Error(err))
	}

	setupGracefulShutdown(log, grpcServer, metricsSrv, database, pgxPool, rabbitQueue, webhookSrv)

	log.Info("Biometric service starting", zap.String("port", port))
	if err := grpcServer.Serve(lis); err != nil && !strings.Contains(err.Error(), "Server closed") {
		log.Fatal("Failed to serve", zap.Error(err))
	}
}

func checkHealth(db *sql.DB, rq queue.Publisher, hs *health.Server, log *logger.Logger) {
	healthy := true

	if db != nil {
		if err := db.PingContext(context.Background()); err != nil {
			log.Warn("Health check: database unavailable", zap.Error(err))
			healthy = false
		}
	}

	if rq != nil {
		if err := rq.Ping(); err != nil {
			log.Warn("Health check: rabbitmq unavailable", zap.Error(err))
			healthy = false
		}
	}

	status := grpc_health_v1.HealthCheckResponse_SERVING
	if !healthy {
		status = grpc_health_v1.HealthCheckResponse_NOT_SERVING
	}

	hs.SetServingStatus("", status)
	hs.SetServingStatus("biometric.BiometricService", status)
}

func loadBiometricConfig() (port, metricsPort, webhookPort, jwtPublicKeyPEM string, dbCfg db.Config) {
	config.InitViper(serviceName)
	port = config.GetEnv("BIOMETRIC_SERVICE_PORT", "50052")
	metricsPort = config.GetEnv("BIOMETRIC_METRICS_PORT", "9090")
	webhookPort = config.GetEnv("OPEN_WEARABLES_WEBHOOK_PORT", "8085")
	jwtPublicKeyPEM = config.GetEnv("JWT_PUBLIC_KEY_PEM")
	dbCfg = db.Config{
		Host:     config.GetEnv("DB_HOST"),
		Port:     config.GetEnv("DB_PORT"),
		User:     config.GetEnv("POSTGRES_USER"),
		Password: config.GetEnv("POSTGRES_PASSWORD"),
		DBName:   config.GetEnv("POSTGRES_DB"),
		SSLMode:  config.GetEnv("DB_SSLMODE"),
	}
	return port, metricsPort, webhookPort, jwtPublicKeyPEM, dbCfg
}

func initBiometricDependencies(dbCfg db.Config, log *logger.Logger) (*sql.DB, *pgxpool.Pool, service.BiometricService, queue.Publisher) {
	database, err := db.NewConnection(dbCfg)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}

	pgxPool, err := db.NewPgxPool(dbCfg)
	if err != nil {
		log.Fatal("Failed to connect to pgx pool", zap.Error(err))
	}

	biometricRepo := pgx.NewBiometricRepositoryPGX(pgxPool)
	biometricSvc := service.NewBiometricService(biometricRepo)

	rabbitURL := config.GetEnv("RABBITMQ_URL", "amqps://rabbitmq:5671/")
	queueName := "biometric_events"
	var rabbitQueue queue.Publisher
	if rabbitURL != "" {
		rabbitQueue, err = queue.NewPublisher(rabbitURL, queueName, log)
		if err != nil {
			log.Warn("Failed to connect to RabbitMQ", zap.Error(err))
		} else {
			log.Info("RabbitMQ connected", zap.String("queue", queueName))
		}
	}

	return database, pgxPool, biometricSvc, rabbitQueue
}

func createBiometricGRPCServer(log *logger.Logger, jwtPublicKeyPEM string) *grpc.Server {
	return createGRPCServer(log, jwtPublicKeyPEM)
}

func buildMetricsServer(metricsPort string, log *logger.Logger) *http.Server {
	metricsSrv := createMetricsServer(metricsPort)
	startMetricsServer(metricsSrv, log)
	return metricsSrv
}

func startWebhookServer(webhookPort string, database *sql.DB, log *logger.Logger) *webhook.Server {
	webhookSrv := webhook.NewServer(webhookPort, webhook.NewSQLDBAdapter(database), log.Logger)
	webhookSrv.Start()
	return webhookSrv
}

func registerBiometricGRPC(grpcServer *grpc.Server, database *sql.DB, biometricSvc service.BiometricService, rabbitQueue queue.Publisher, log *logger.Logger) {
	healthServer := health.NewServer()
	pb.RegisterBiometricServiceServer(grpcServer, &biometricServer{
		db:           database,
		biometricSvc: biometricSvc,
		log:          log,
		rabbitQueue:  rabbitQueue,
	})
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	startHealthCheckLoop(database, rabbitQueue, healthServer, log)
}

func startHealthChecks(db *sql.DB, rq queue.Publisher, log *logger.Logger) {
	healthServer := health.NewServer()
	startHealthCheckLoop(db, rq, healthServer, log)
}
