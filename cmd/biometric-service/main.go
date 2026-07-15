package main

import (
	"context"
	"database/sql"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/MAMUER/project/api/gen/biometric"
	"github.com/MAMUER/project/internal/config"
	"github.com/MAMUER/project/internal/db"
	grpctls "github.com/MAMUER/project/internal/grpc"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/metrics"
	"github.com/MAMUER/project/internal/middleware"
	"github.com/MAMUER/project/internal/queue"
	"github.com/MAMUER/project/internal/telemetry"
	"github.com/MAMUER/project/internal/validator"
)

type biometricServer struct {
	pb.UnimplementedBiometricServiceServer
	db          *sql.DB
	log         *logger.Logger
	rabbitQueue queue.Publisher
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
	start := time.Now()
	s.log.Info("BIOMETRIC_DATA_RECEIVED",
		zap.String("action", "BIOMETRIC_DATA_RECEIVED"),
		zap.String("user_id", req.UserId),
		zap.String("metric_type", req.MetricType),
		zap.Float64("value", req.Value),
	)

	if err := validator.ValidateBiometricRequest(req); err != nil {
		s.log.Warn("Invalid biometric request", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
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

	id := uuid.New().String()
	timestamp := req.Timestamp.AsTime()

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO biometric_data (id, user_id, metric_type, value, timestamp, device_type)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, metric_type, timestamp, device_type) DO NOTHING
	`, id, req.UserId, req.MetricType, req.Value, timestamp, req.DeviceType)
	if err != nil {
		s.log.Error("Failed to insert record", zap.Error(err))
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

	return &pb.AddRecordResponse{Id: id}, nil
}

func (s *biometricServer) BatchAddRecords(ctx context.Context, req *pb.BatchAddRecordsRequest) (*pb.BatchAddRecordsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if len(req.Records) == 0 {
		return nil, status.Error(codes.InvalidArgument, "records cannot be empty")
	}

	var userExists bool
	if err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", req.UserId).Scan(&userExists); err != nil {
		s.log.Error("Failed to check user existence", zap.Error(err), zap.String("user_id", req.UserId))
		return nil, status.Error(codes.Internal, "failed to verify user")
	}
	if !userExists {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	for i, rec := range req.Records {
		if err := ctx.Err(); err != nil {
			return nil, status.Error(codes.Canceled, "request canceled")
		}
		if err := validator.ValidateBiometricRecord(rec); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "record[%d]: %v", i, err)
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.log.Error("Failed to begin transaction", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const query = `INSERT INTO biometric_data (id, user_id, metric_type, value, timestamp, device_type, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, metric_type, timestamp, device_type) DO NOTHING`

	inserted := 0
	for _, rec := range req.Records {
		if err := ctx.Err(); err != nil {
			_ = tx.Rollback()
			return nil, status.Error(codes.Canceled, "request canceled")
		}

		id := uuid.New().String()
		ts := rec.Timestamp.AsTime()
		if ts.IsZero() {
			ts = time.Now()
		}

		result, err := tx.ExecContext(ctx, query,
			id, req.UserId, rec.MetricType, rec.Value, ts, rec.DeviceType, time.Now(),
		)
		if err != nil {
			_ = tx.Rollback()
			s.log.Error("Failed to insert biometric record",
				zap.Error(err),
				zap.String("metric_type", rec.MetricType),
			)
			return nil, status.Error(codes.Internal, "failed to save records")
		}
		if n, _ := result.RowsAffected(); n > 0 {
			inserted++
		}
	}

	if err := tx.Commit(); err != nil {
		s.log.Error("Failed to commit transaction", zap.Error(err))
		return nil, status.Error(codes.Internal, "database commit error")
	}

	return &pb.BatchAddRecordsResponse{Count: safeIntToInt32(inserted)}, nil
}

type recordsQuery struct {
	query string
	args  []interface{}
}

func (s *biometricServer) buildGetRecordsQuery(req *pb.GetRecordsRequest) recordsQuery {
	from := req.From.AsTime()
	to := req.To.AsTime()

	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 10000 {
		limit = 10000
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	baseQuery := `
		SELECT id, user_id, metric_type, value, timestamp, device_type, created_at
		FROM biometric_data
		WHERE user_id = $1 AND metric_type = $2
	`

	switch {
	case from.IsZero() && to.IsZero():
		return recordsQuery{
			query: baseQuery + `
			ORDER BY timestamp DESC
			LIMIT $3 OFFSET $4
		`,
			args: []interface{}{req.UserId, req.MetricType, limit, offset},
		}
	case from.IsZero():
		return recordsQuery{
			query: baseQuery + ` AND timestamp <= $3
			ORDER BY timestamp DESC
			LIMIT $4 OFFSET $5
		`,
			args: []interface{}{req.UserId, req.MetricType, to, limit, offset},
		}
	case to.IsZero():
		return recordsQuery{
			query: baseQuery + ` AND timestamp >= $3
			ORDER BY timestamp DESC
			LIMIT $4 OFFSET $5
		`,
			args: []interface{}{req.UserId, req.MetricType, from, limit, offset},
		}
	default:
		return recordsQuery{
			query: baseQuery + ` AND timestamp >= $3 AND timestamp <= $4
			ORDER BY timestamp DESC
			LIMIT $5 OFFSET $6
		`,
			args: []interface{}{req.UserId, req.MetricType, from, to, limit, offset},
		}
	}
}

func (s *biometricServer) GetRecords(ctx context.Context, req *pb.GetRecordsRequest) (*pb.GetRecordsResponse, error) {
	s.log.Debug("GetRecords",
		zap.String("user_id", req.UserId),
		zap.String("metric_type", req.MetricType),
	)

	from := req.From.AsTime()
	to := req.To.AsTime()

	if !from.IsZero() && !to.IsZero() && from.After(to) {
		return nil, status.Error(codes.InvalidArgument, "from cannot be after to")
	}

	q := s.buildGetRecordsQuery(req)
	rows, err := s.db.QueryContext(ctx, q.query, q.args...)
	if err != nil {
		s.log.Error("Failed to query records", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to query records")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.log.Error("Failed to close rows", zap.Error(closeErr))
		}
	}()

	var records []*pb.BiometricRecord
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

	return &pb.GetRecordsResponse{Records: records}, nil
}

func (s *biometricServer) GetLatest(ctx context.Context, req *pb.GetLatestRequest) (*pb.BiometricRecord, error) {
	s.log.Debug("GetLatest",
		zap.String("user_id", req.UserId),
		zap.String("metric_type", req.MetricType),
	)

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
		return nil, status.Error(codes.NotFound, "record not found")
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
		return nil, status.Error(codes.NotFound, "record not found")
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
		metrics.UnaryServerInterceptor("biometric-service"),
	), telemetry.ServerHandlerOption()}
	if creds, err := grpctls.GetServerTLSCredentials(); err == nil && creds != nil {
		serverOpts = append(serverOpts, grpc.Creds(creds))
	}
	return grpc.NewServer(serverOpts...)
}

func createMetricsServer(metricsPort string) *http.Server {
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	return &http.Server{Addr: ":" + metricsPort, Handler: metricsMux}
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

func setupGracefulShutdown(log *logger.Logger, grpcServer *grpc.Server, metricsSrv *http.Server) context.Context {
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
		stop()
	}()

	return ctx
}

func main() {
	log := logger.New("biometric-service")

	shutdownTraces := telemetry.InitTracer()
	defer func() {
		if err := shutdownTraces(context.Background()); err != nil {
			log.Warn("Failed to shutdown traces", zap.Error(err))
		}
	}()

	port := config.GetEnv("BIOMETRIC_SERVICE_PORT", "50052")
	metricsPort := config.GetEnv("BIOMETRIC_METRICS_PORT", "9090")
	jwtPublicKeyPEM := config.GetEnv("JWT_PUBLIC_KEY_PEM")

	dbCfg := db.Config{
		Host:     config.GetEnv("DB_HOST"),
		Port:     config.GetEnv("DB_PORT"),
		User:     config.GetEnv("POSTGRES_USER"),
		Password: config.GetEnv("POSTGRES_PASSWORD"),
		DBName:   config.GetEnv("POSTGRES_DB"),
		SSLMode:  config.GetEnv("DB_SSLMODE"),
	}

	grpcServer := createGRPCServer(log, jwtPublicKeyPEM)

	database, err := db.NewConnection(dbCfg)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer func() {
		if closeErr := database.Close(); closeErr != nil {
			log.Error("Failed to close database", zap.Error(closeErr))
		}
	}()

	rabbitURL := config.GetEnv("RABBITMQ_URL")
	queueName := "biometric_events"
	var rabbitQueue queue.Publisher
	if rabbitURL != "" {
		rabbitQueue, err = queue.NewPublisher(rabbitURL, queueName, log.Logger)
		if err != nil {
			log.Warn("Failed to connect to RabbitMQ", zap.Error(err))
		} else {
			defer func() { _ = rabbitQueue.Close() }()
			log.Info("RabbitMQ connected", zap.String("queue", queueName))
		}
	}

	lc := net.ListenConfig{}
	lis, err := lc.Listen(context.Background(), "tcp", ":"+port)
	if err != nil {
		log.Fatal("Failed to listen", zap.Error(err))
	}

	healthServer := health.NewServer()

	pb.RegisterBiometricServiceServer(grpcServer, &biometricServer{
		db:          database,
		log:         log,
		rabbitQueue: rabbitQueue,
	})

	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	startHealthCheckLoop(database, rabbitQueue, healthServer, log)

	metricsSrv := createMetricsServer(metricsPort)
	startMetricsServer(metricsSrv, log)

	setupGracefulShutdown(log, grpcServer, metricsSrv)

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
