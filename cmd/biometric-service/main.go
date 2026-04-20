package main

import (
	"context"
	"database/sql"
	"net"
	"os"
	"time"

	pb "github.com/MAMUER/Project/api/gen/biometric"
	"github.com/MAMUER/Project/internal/db"
	"github.com/MAMUER/Project/internal/logger"
	"github.com/MAMUER/Project/internal/metrics"
	"github.com/MAMUER/Project/internal/queue"
	"github.com/MAMUER/Project/internal/validator"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type biometricServer struct {
	pb.UnimplementedBiometricServiceServer
	db          *sql.DB
	log         *logger.Logger
	rabbitQueue queue.Publisher // ← ИНТЕРФЕЙС, не *queue.Publisher!
}

// safeIntToInt32 safely converts int to int32
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
	s.log.Info("AddRecord",
		zap.String("user_id", req.UserId),
		zap.String("metric_type", req.MetricType),
		zap.Float64("value", req.Value),
	)

	// Валидация входных данных
	if err := validator.ValidateBiometricRequest(req); err != nil { // ← Используем из пакета
		s.log.Warn("Invalid biometric request", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	id := uuid.New().String()
	timestamp := req.Timestamp.AsTime()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO biometric_data (id, user_id, metric_type, value, timestamp, device_type)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, req.UserId, req.MetricType, req.Value, timestamp, req.DeviceType)
	if err != nil {
		s.log.Error("Failed to insert record", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to insert record")
	}

	event := map[string]interface{}{
		"user_id":     req.UserId,
		"metric_type": req.MetricType,
		"value":       req.Value,
		"timestamp":   timestamp,
	}

	if s.rabbitQueue != nil {
		if err := s.rabbitQueue.Publish(ctx, event); err != nil {
			s.log.Warn("Failed to publish to queue", zap.Error(err))
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

	// Валидируем КАЖДУЮ запись перед транзакцией
	for i, rec := range req.Records {
		if err := ctx.Err(); err != nil {
			return nil, status.Error(codes.Canceled, "request cancelled")
		}
		if err := validator.ValidateBiometricRecord(rec); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "record[%d]: %v", i, err)
		}
	}

	// Начинаем транзакцию
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.log.Error("Failed to begin transaction", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}
	defer func() {
		// Rollback безопасен если транзакция уже закоммичена
		_ = tx.Rollback()
	}()

	const query = `INSERT INTO biometric_data (id, user_id, metric_type, value, timestamp, device_type, created_at) 
        VALUES ($1, $2, $3, $4, $5, $6, $7)`

	for _, rec := range req.Records {
		if err := ctx.Err(); err != nil {
			_ = tx.Rollback()
			return nil, status.Error(codes.Canceled, "request cancelled")
		}

		id := uuid.New().String()
		ts := rec.Timestamp.AsTime()
		if ts.IsZero() {
			ts = time.Now()
		}

		_, err := tx.ExecContext(ctx, query,
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
	}

	if err := tx.Commit(); err != nil {
		s.log.Error("Failed to commit transaction", zap.Error(err))
		return nil, status.Error(codes.Internal, "database commit error")
	}

	return &pb.BatchAddRecordsResponse{Count: safeIntToInt32(len(req.Records))}, nil
}

func (s *biometricServer) GetRecords(ctx context.Context, req *pb.GetRecordsRequest) (*pb.GetRecordsResponse, error) {
	s.log.Debug("GetRecords",
		zap.String("user_id", req.UserId),
		zap.String("metric_type", req.MetricType),
	)

	from := req.From.AsTime()
	to := req.To.AsTime()

	// Use fixed parameter positions with COALESCE for optional filters
	// This avoids SQL string concatenation while handling null time filters
	var rows *sql.Rows
	var err error
	if from.IsZero() && to.IsZero() {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, user_id, metric_type, value, timestamp, device_type, created_at
			FROM biometric_data
			WHERE user_id = $1 AND metric_type = $2
			ORDER BY timestamp DESC LIMIT $3
		`, req.UserId, req.MetricType, req.Limit)
	} else if from.IsZero() {
		// Only to filter
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, user_id, metric_type, value, timestamp, device_type, created_at
			FROM biometric_data
			WHERE user_id = $1 AND metric_type = $2 AND timestamp <= $3
			ORDER BY timestamp DESC LIMIT $4
		`, req.UserId, req.MetricType, to, req.Limit)
	} else if to.IsZero() {
		// Only from filter
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, user_id, metric_type, value, timestamp, device_type, created_at
			FROM biometric_data
			WHERE user_id = $1 AND metric_type = $2 AND timestamp >= $3
			ORDER BY timestamp DESC LIMIT $4
		`, req.UserId, req.MetricType, from, req.Limit)
	} else {
		// Both filters
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, user_id, metric_type, value, timestamp, device_type, created_at
			FROM biometric_data
			WHERE user_id = $1 AND metric_type = $2 AND timestamp >= $3 AND timestamp <= $4
			ORDER BY timestamp DESC LIMIT $5
		`, req.UserId, req.MetricType, from, to, req.Limit)
	}

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

func main() {
	log := logger.New("biometric-service")

	port := os.Getenv("BIOMETRIC_SERVICE_PORT")
	if port == "" {
		port = "50052"
	}

	dbCfg := db.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(metrics.UnaryServerInterceptor("biometric-service")),
	)

	database, err := db.NewConnection(dbCfg)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer func() {
		if closeErr := database.Close(); closeErr != nil {
			log.Error("Failed to close database", zap.Error(closeErr))
		}
	}()

	rabbitURL := os.Getenv("RABBITMQ_URL")
	queueName := "biometric_events"
	var rabbitQueue queue.Publisher // ← ИНТЕРФЕЙС
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

	pb.RegisterBiometricServiceServer(grpcServer, &biometricServer{
		db:          database,
		log:         log,
		rabbitQueue: rabbitQueue, // ← Передаём интерфейс
	})

	log.Info("Biometric service starting", zap.String("port", port))
	if err := grpcServer.Serve(lis); err != nil { // ← grpcServer, не `s`
		log.Fatal("Failed to serve", zap.Error(err))
	}
}
