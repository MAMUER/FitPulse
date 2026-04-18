package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"

	biometricpb "github.com/MAMUER/Project/api/gen/biometric"
	trainingpb "github.com/MAMUER/Project/api/gen/training"
	userpb "github.com/MAMUER/Project/api/gen/user"
	"github.com/MAMUER/Project/internal/cache"
	"github.com/MAMUER/Project/internal/logger"
	"github.com/MAMUER/Project/internal/middleware"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type gateway struct {
	userClient         userpb.UserServiceClient
	biometricClient    biometricpb.BiometricServiceClient
	trainingClient     trainingpb.TrainingServiceClient
	mlClassifierURL    string
	mlGeneratorURL     string
	deviceConnectorURL string
	log                *logger.Logger
	jwtSecret          string
	db                 *sql.DB             // For server-side role re-verification
	sessionStore       *cache.SessionStore // For session management (Security #1, #3)
	// Async ML processing
	rdb     *redis.Client
	rmqCh   *amqp.Channel
	mlAsync bool
}

func main() {
	log := logger.New("gateway")
	defer func() {
		if syncErr := log.Sync(); syncErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", syncErr)
		}
	}()

	port := os.Getenv("GATEWAY_PORT")
	if port == "" {
		port = "8080"
	}

	userServiceAddr := os.Getenv("USER_SERVICE_ADDR")
	if userServiceAddr == "" {
		userServiceAddr = "localhost:50051"
	}

	biometricServiceAddr := os.Getenv("BIOMETRIC_SERVICE_ADDR")
	if biometricServiceAddr == "" {
		biometricServiceAddr = "localhost:50052"
	}

	trainingServiceAddr := os.Getenv("TRAINING_SERVICE_ADDR")
	if trainingServiceAddr == "" {
		trainingServiceAddr = "localhost:50053"
	}

	mlClassifierURL := os.Getenv("ML_CLASSIFIER_URL")
	if mlClassifierURL == "" {
		mlClassifierURL = "http://localhost:8001"
	}

	mlGeneratorURL := os.Getenv("ML_GENERATOR_URL")
	if mlGeneratorURL == "" {
		mlGeneratorURL = "http://localhost:8002"
	}

	deviceConnectorURL := os.Getenv("DEVICE_CONNECTOR_URL")
	if deviceConnectorURL == "" {
		deviceConnectorURL = "http://localhost:8082"
	}

	// Async ML processing configuration
	mlAsync := os.Getenv("ML_ASYNC") == "true" || os.Getenv("ML_ASYNC") == "True" || os.Getenv("ML_ASYNC") == "1"

	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		rabbitmqURL = "amqp://guest:guest@localhost:5672/" //nolint:gosec // G101: default dev credentials, not for production
	}

	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	// Database connection for server-side role re-verification (Security #10)
	dbURL := os.Getenv("DATABASE_URL")
	var db *sql.DB
	if dbURL != "" {
		var openErr error
		db, openErr = sql.Open("postgres", dbURL)
		if openErr != nil {
			log.Fatal("Failed to open database", zap.Error(openErr))
		}
		db.SetMaxOpenConns(5)
		db.SetMaxIdleConns(2)
		db.SetConnMaxLifetime(5 * time.Minute)
		if pingErr := db.PingContext(context.Background()); pingErr != nil {
			log.Fatal("Failed to ping database", zap.Error(pingErr))
		}
		defer func() {
			if closeErr := db.Close(); closeErr != nil {
				log.Error("Failed to close database connection", zap.Error(closeErr))
			}
		}()
	}

	// Redis client for job result storage (used in async mode)
	var rdb *redis.Client
	if mlAsync {
		rdb = redis.NewClient(&redis.Options{
			Addr: redisHost + ":6379",
		})
		if pingErr := rdb.Ping(context.Background()).Err(); pingErr != nil {
			log.Warn("Redis unavailable, async ML mode disabled", zap.Error(pingErr))
			mlAsync = false
		} else {
			log.Info("Redis connected for async job results", zap.String("host", redisHost))
		}
	}

	// Redis client for session management (Security #1, #3) — always required
	var sessionStore *cache.SessionStore
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisHost + ":6379",
	})
	if pingErr := redisClient.Ping(context.Background()).Err(); pingErr != nil {
		log.Warn("Redis unavailable for session management", zap.Error(pingErr))
	} else {
		sessionStore = cache.NewSessionStoreFromRedis(redisClient)
		log.Info("Redis connected for session management", zap.String("host", redisHost))
	}

	// RabbitMQ channel for publishing ML jobs (used in async mode)
	var rmqCh *amqp.Channel
	var rmqClose func()
	if mlAsync {
		rmqConn, rmqErr := amqp.Dial(rabbitmqURL)
		if rmqErr != nil {
			log.Warn("RabbitMQ unavailable, async ML mode disabled", zap.Error(rmqErr))
			mlAsync = false
			rdb = nil
		} else {
			rmqCh, rmqErr = rmqConn.Channel()
			if rmqErr != nil {
				log.Warn("Failed to create RabbitMQ channel, async ML mode disabled", zap.Error(rmqErr))
				mlAsync = false
				rdb = nil
				if closeErr := rmqConn.Close(); closeErr != nil {
					log.Warn("Failed to close RabbitMQ connection", zap.Error(closeErr))
				}
			} else {
				// Declare queues (idempotent)
				_, _ = rmqCh.QueueDeclare("ml.classify", true, false, false, false, nil)
				_, _ = rmqCh.QueueDeclare("ml.generate", true, false, false, false, nil)
				log.Info("RabbitMQ connected for async ML jobs", zap.String("url", rabbitmqURL))
				rmqClose = func() {
					if closeErr := rmqConn.Close(); closeErr != nil {
						log.Warn("Failed to close RabbitMQ connection", zap.Error(closeErr))
					}
				}
			}
		}
	}

	// gRPC connections
	userConn, err := grpc.NewClient(userServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true), grpc.MaxCallRecvMsgSize(10<<20)),
	)
	if err != nil {
		log.Fatal("Failed to connect to user service", zap.Error(err))
	}
	defer func() {
		if closeErr := userConn.Close(); closeErr != nil {
			log.Error("Failed to close user service connection", zap.Error(closeErr))
		}
	}()

	biometricConn, err := grpc.NewClient(biometricServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		log.Fatal("Failed to connect to biometric service", zap.Error(err))
	}
	defer func() {
		if closeErr := biometricConn.Close(); closeErr != nil {
			log.Error("Failed to close biometric service connection", zap.Error(closeErr))
		}
	}()

	trainingConn, err := grpc.NewClient(trainingServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		log.Fatal("Failed to connect to training service", zap.Error(err))
	}
	defer func() {
		if closeErr := trainingConn.Close(); closeErr != nil {
			log.Error("Failed to close training service connection", zap.Error(closeErr))
		}
	}()

	if rmqClose != nil {
		defer rmqClose()
	}

	g := &gateway{
		userClient:         userpb.NewUserServiceClient(userConn),
		biometricClient:    biometricpb.NewBiometricServiceClient(biometricConn),
		trainingClient:     trainingpb.NewTrainingServiceClient(trainingConn),
		mlClassifierURL:    mlClassifierURL,
		mlGeneratorURL:     mlGeneratorURL,
		deviceConnectorURL: deviceConnectorURL,
		log:                log,
		jwtSecret:          jwtSecret,
		db:                 db,
		sessionStore:       sessionStore,
		rdb:                rdb,
		rmqCh:              rmqCh,
		mlAsync:            mlAsync,
	}

	// Setup routes (middleware applied via r.Use() in registerRoutes)
	r := g.registerRoutes()

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Info("Gateway starting",
		zap.String("port", port),
		zap.String("ml_classifier", mlClassifierURL),
		zap.String("ml_generator", mlGeneratorURL))

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal("Failed to start server", zap.Error(err))
	}
}

// registerRoutes registers all HTTP routes on the router
func (g *gateway) registerRoutes() *mux.Router {
	r := mux.NewRouter()

	// ========== Security middleware (applied to ALL routes) ==========
	// Требование #5, #8, #12: Security headers на уровне роутера
	r.Use(middleware.RemoveServerHeader)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.RecoveryMiddleware(g.log.Logger))
	r.Use(middleware.RateLimit)
	r.Use(middleware.RequestID)
	r.Use(middleware.LoggingMiddleware(g.log.Logger))

	// Public routes
	r.HandleFunc("/api/v1/register", g.registerHandler).Methods("POST")
	r.HandleFunc("/api/v1/register/invite", g.registerWithInviteHandler).Methods("POST")
	r.HandleFunc("/api/v1/invite/validate", g.validateInviteCodeHandler).Methods("POST")
	r.HandleFunc("/api/v1/login", g.loginHandler).Methods("POST")
	r.HandleFunc("/api/v1/auth/confirm", g.confirmEmailHandler).Methods("POST")
	r.HandleFunc("/api/v1/auth/verify-status", g.checkVerificationStatusHandler).Methods("GET")
	r.HandleFunc("/health", g.healthHandler).Methods("GET")

	// Email confirmation page
	r.HandleFunc("/confirm", g.emailConfirmPageHandler).Methods("GET")

	// Device connector routes (device token auth, not JWT)
	r.HandleFunc("/api/v1/devices/register", g.deviceRegisterHandler).Methods("POST")
	r.HandleFunc("/api/v1/devices/{device_id}/ingest", g.deviceIngestHandler).Methods("POST")

	// Protected routes (JWT required)
	protected := r.PathPrefix("/api/v1").Subrouter()
	protected.Use(middleware.AuthMiddleware(g.jwtSecret, g.log.Logger))

	protected.HandleFunc("/logout", g.logoutHandler).Methods("POST")
	protected.HandleFunc("/profile", g.profileHandler).Methods("GET")
	protected.HandleFunc("/profile", g.updateProfileHandler).Methods("PUT")
	protected.HandleFunc("/profile", g.deleteProfileHandler).Methods("DELETE")
	protected.HandleFunc("/auth/change-password", g.changePasswordHandler).Methods("POST")

	protected.HandleFunc("/devices", g.getDevicesHandler).Methods("GET")

	protected.HandleFunc("/biometrics", g.addBiometricRecordHandler).Methods("POST")
	protected.HandleFunc("/biometrics", g.getBiometricRecordsHandler).Methods("GET")

	protected.HandleFunc("/training/generate", g.generatePlanHandler).Methods("POST")
	protected.HandleFunc("/training/plans", g.getPlansHandler).Methods("GET")
	protected.HandleFunc("/training/complete", g.completeWorkoutHandler).Methods("POST")
	protected.HandleFunc("/training/progress", g.getProgressHandler).Methods("GET")

	protected.HandleFunc("/ml/classify", g.classifyHandler).Methods("POST")
	protected.HandleFunc("/ml/classify/{job_id}", g.classifyStatusHandler).Methods("GET")
	protected.HandleFunc("/ml/generate-plan", g.generateMLPlanHandler).Methods("POST")
	protected.HandleFunc("/ml/generate-plan/{job_id}", g.generatePlanStatusHandler).Methods("GET")

	// Admin routes (JWT + RequirePrivilege middleware — Security #10)
	admin := r.PathPrefix("/api/v1/admin").Subrouter()
	admin.Use(middleware.AuthMiddleware(g.jwtSecret, g.log.Logger))
	if g.db != nil {
		admin.Use(middleware.RequirePrivilege(g.db, "admin", g.log.Logger))
	}
	admin.HandleFunc("/users", g.adminListUsersHandler).Methods("GET")

	// Static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static/"))))
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/")))

	return r
}
