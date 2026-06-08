package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"

	biometricpb "github.com/MAMUER/project/api/gen/biometric"
	trainingpb "github.com/MAMUER/project/api/gen/training"
	userpb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/cache"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/middleware"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type gateway struct {
	userClient         userpb.UserServiceClient
	userConn           *grpc.ClientConn
	biometricAddr      string
	biometricClient    biometricpb.BiometricServiceClient
	trainingAddr       string
	trainingClient     trainingpb.TrainingServiceClient
	mlClassifierURL    string
	mlGeneratorURL     string
	deviceConnectorURL string
	log                *logger.Logger
	jwtSecret          string
	db                 *sql.DB
	sessionStore       *cache.SessionStore
	rdb                *redis.Client
	rmqCh              *amqp.Channel
	mlAsync            bool
	requestDuration    *prometheus.HistogramVec
	requestTotal       *prometheus.CounterVec
	errorTotal         *prometheus.CounterVec

	biometricMu sync.Mutex
	trainingMu  sync.Mutex
}

//nolint:gocognit,gocyclo,funlen // Complexity due to sequential init of multiple services
func main() {
	log := logger.New("gateway")
	defer func() {
		if syncErr := log.Sync(); syncErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", syncErr)
		}
	}()

	// Initialize Prometheus metrics
	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "endpoint", "method", "status"},
	)
	requestTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "request_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"service", "endpoint", "method"},
	)
	errorTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "error_total",
			Help: "Total number of errors",
		},
		[]string{"service", "error_code", "endpoint"},
	)

	prometheus.MustRegister(requestDuration, requestTotal, errorTotal)

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
		mlGeneratorURL = "http://ml-generator:8002"
	}

	parsedURL, err := url.Parse(mlGeneratorURL)
	if err != nil {
		log.Fatal("invalid ML_GENERATOR_URL", zap.Error(err))
	}

	allowedHosts := map[string]bool{
		"ml-generator:8002": true,
		"ml-generator":      true,
		"generator:8002":    true,
		"generator":         true,
		"localhost:8002":    true,
		"localhost:8001":    true,
		"localhost":         true,
	}
	if !allowedHosts[parsedURL.Host] && !allowedHosts[parsedURL.Hostname()] {
		log.Fatal("ML_GENERATOR_URL host not allowed", zap.String("host", parsedURL.Host))
	}

	deviceConnectorURL := os.Getenv("DEVICE_CONNECTOR_URL")
	if deviceConnectorURL == "" {
		deviceConnectorURL = "http://localhost:8082"
	}

	mlAsync := os.Getenv("ML_ASYNC") == "true" || os.Getenv("ML_ASYNC") == "True" || os.Getenv("ML_ASYNC") == "1"

	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		rabbitmqURL = "amqp://localhost:5672/"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisHost := os.Getenv("REDIS_HOST")
		if redisHost == "" {
			redisHost = "redis"
		}
		redisAddr = redisHost + ":6379"
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
			Addr: redisAddr,
		})
		if pingErr := rdb.Ping(context.Background()).Err(); pingErr != nil {
			log.Warn("Redis unavailable, async ML mode disabled", zap.Error(pingErr))
			mlAsync = false
		} else {
			log.Info("Redis connected for async job results", zap.String("addr", redisAddr))
		}
	}

	// Redis client for session management (Security #1, #3) — always required
	var sessionStore *cache.SessionStore
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	if pingErr := redisClient.Ping(context.Background()).Err(); pingErr != nil {
		log.Warn("Redis unavailable for session management", zap.Error(pingErr))
	} else {
		sessionStore = cache.NewSessionStoreFromRedis(redisClient)
		log.Info("Redis connected for session management", zap.String("addr", redisAddr))
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

	// gRPC connections — User service is critical
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

	if rmqClose != nil {
		defer rmqClose()
	}

	g := &gateway{
		userClient:         userpb.NewUserServiceClient(userConn),
		userConn:           userConn,
		biometricAddr:      biometricServiceAddr,
		trainingAddr:       trainingServiceAddr,
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
		requestDuration:    requestDuration,
		requestTotal:       requestTotal,
		errorTotal:         errorTotal,
	}

	// Setup routes (middleware applied via r.Use() in registerRoutes)
	r := g.registerRoutes()

	// ========== Определяем режим работы ==========
	tlsCertFile := os.Getenv("TLS_CERT_FILE")
	tlsKeyFile := os.Getenv("TLS_KEY_FILE")
	tlsAvailable := tlsCertFile != "" && tlsKeyFile != ""
	if tlsAvailable {
		if _, err := os.Stat(filepath.Clean(tlsCertFile)); err != nil { //gosec:G703
			log.Warn("TLS certificate file not found, falling back to HTTP-only mode",
				zap.String("cert_file", tlsCertFile),
				zap.Error(err))
			tlsAvailable = false
		}
	}

	// ========== HTTPS server (main) ==========
	httpsSrv := &http.Server{
		Addr:              ":8443",
		Handler:           r,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 15 * time.Second,
	}

	// ========== HTTP server ==========
	var httpSrv *http.Server
	if tlsAvailable {
		// HTTPS mode: HTTP port redirects to HTTPS
		httpSrv = &http.Server{
			Addr:              ":" + port,
			ReadHeaderTimeout: 15 * time.Second,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				target := "https://" + r.Host
				// Заменяем порт 8080 на 8443 в редиректе
				if port != "" && port != "80" && port != "443" {
					host := r.Host
					if idx := len(host) - len(":"+port); idx > 0 && host[idx:] == ":"+port {
						host = host[:idx] + ":8443"
					}
					target = "https://" + host + r.URL.RequestURI()
				}
				http.Redirect(w, r, target, http.StatusMovedPermanently)
			}),
		}
	} else {
		// HTTP-only mode (test/local): HTTP port serves directly
		httpSrv = &http.Server{
			Addr:              ":" + port,
			Handler:           r,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
			ReadHeaderTimeout: 15 * time.Second,
		}
	}

	// ========== Start servers ==========
	if tlsAvailable {
		// HTTPS mode
		go func() {
			log.Info("HTTP redirect server starting", zap.String("port", port))
			if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Error("HTTP redirect server failed", zap.Error(err))
			}
		}()

		go func() {
			log.Info("HTTPS server starting",
				zap.String("port", "8443"),
				zap.String("cert", tlsCertFile),
				zap.String("ml_classifier", mlClassifierURL),
				zap.String("ml_generator", mlGeneratorURL))
			if err := httpsSrv.ListenAndServeTLS(tlsCertFile, tlsKeyFile); err != nil && err != http.ErrServerClosed {
				log.Fatal("Failed to start HTTPS server", zap.Error(err))
			}
		}()
	} else {
		// HTTP-only mode
		log.Info("Starting in HTTP-only mode (no TLS certificates)",
			zap.String("port", port),
			zap.String("ml_classifier", mlClassifierURL),
			zap.String("ml_generator", mlGeneratorURL))
		go func() {
			if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal("Failed to start HTTP server", zap.Error(err))
			}
		}()
	}

	// ========== Graceful shutdown ==========
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = httpSrv.Shutdown(ctx)
	_ = httpsSrv.Shutdown(ctx)
	log.Info("Servers stopped")
}

// registerRoutes registers all HTTP routes on the router
func (g *gateway) registerRoutes() *mux.Router {
	r := mux.NewRouter()

	// ========== Security middleware (applied to ALL routes) ==========
	r.Use(middleware.RemoveServerHeader)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.RecoveryMiddleware(g.log.Logger))
	r.Use(middleware.RateLimit)
	r.Use(middleware.RequestID)
	r.Use(middleware.LoggingMiddleware(g.log.Logger, g.requestDuration, g.requestTotal, g.errorTotal))
	r.Use(middleware.CorrelationIDHTTP)
	// ========== Public routes (без авторизации) ==========
	r.HandleFunc("/api/v1/register", g.registerHandler).Methods("POST")
	r.HandleFunc("/api/v1/register/invite", g.registerWithInviteHandler).Methods("POST")
	r.HandleFunc("/api/v1/invite/validate", g.validateInviteCodeHandler).Methods("POST")
	r.HandleFunc("/api/v1/login", g.loginHandler).Methods("POST")
	r.HandleFunc("/api/v1/auth/confirm", g.confirmEmailHandler).Methods("POST")
	r.HandleFunc("/api/v1/auth/verify-status", g.checkVerificationStatusHandler).Methods("GET")
	r.HandleFunc("/health", g.healthHandler).Methods("GET")
	// Webhook endpoint - БЕЗ authMiddleware
	r.HandleFunc("/api/v1/devices/withings/webhook", g.proxyToDeviceAggregator).Methods("POST")
	// Metrics endpoint
	r.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// Email confirmation page
	r.HandleFunc("/confirm", g.emailConfirmPageHandler).Methods("GET")

	authMiddleware := middleware.AuthMiddleware(g.jwtSecret, g.log.Logger)

	protected := r.PathPrefix("/api/v1").Subrouter()
	protected.Use(authMiddleware)

	// Profile
	protected.HandleFunc("/profile", g.getProfileHandler).Methods("GET")
	protected.HandleFunc("/profile", g.updateProfileHandler).Methods("PUT")

	// Biometrics
	protected.HandleFunc("/biometrics", g.addBiometricRecordHandler).Methods("POST")
	protected.HandleFunc("/biometrics", g.getBiometricRecordsHandler).Methods("GET")

	// Training
	protected.HandleFunc("/training/plans", g.listPlansHandler).Methods("GET")
	protected.HandleFunc("/training/plans/generate", g.generatePlanHandler).Methods("POST")
	protected.HandleFunc("/training/plans/{plan_id}", g.getPlanHandler).Methods("GET")
	protected.HandleFunc("/training/progress", g.getProgressHandler).Methods("GET")
	protected.HandleFunc("/training/workouts/{workout_id}/complete", g.completeWorkoutHandler).Methods("POST")

	// ML
	protected.HandleFunc("/ml/classify", g.mlClassifyHandler).Methods("POST")
	protected.HandleFunc("/ml/generate", g.mlGenerateHandler).Methods("POST")

	// Devices
	protected.HandleFunc("/devices", g.listDevicesHandler).Methods("GET")
	protected.HandleFunc("/devices", g.registerDeviceHandler).Methods("POST")

	// Logout
	protected.HandleFunc("/logout", g.logoutHandler).Methods("POST")

	// Device aggregator proxy
	protected.HandleFunc("/devices/fitbit/auth", g.proxyToDeviceAggregator).Methods("GET")
	protected.HandleFunc("/devices/fitbit/callback", g.proxyToDeviceAggregator).Methods("GET")
	protected.HandleFunc("/devices/fitbit/disconnect", g.proxyToDeviceAggregator).Methods("POST")
	protected.HandleFunc("/devices/providers", g.proxyToDeviceAggregator).Methods("GET")

	protected.HandleFunc("/devices/withings/auth", g.proxyToDeviceAggregator).Methods("GET")
	protected.HandleFunc("/devices/withings/callback", g.proxyToDeviceAggregator).Methods("GET")
	protected.HandleFunc("/devices/withings/disconnect", g.proxyToDeviceAggregator).Methods("POST")
	// ========== Admin routes (требуют роль admin) ==========
	admin := r.PathPrefix("/api/v1/admin").Subrouter()
	admin.Use(authMiddleware)
	admin.Use(middleware.RequireRole("admin"))
	admin.HandleFunc("/users", g.adminListUsersHandler).Methods("GET")
	admin.HandleFunc("/invites", g.adminListInvitesHandler).Methods("GET")                 // ← НОВОЕ
	admin.HandleFunc("/invites", g.adminCreateInviteHandler).Methods("POST")               // ← НОВОЕ
	admin.HandleFunc("/invites/{code}/revoke", g.adminRevokeInviteHandler).Methods("POST") // ← НОВОЕ

	// ========== Static files ==========
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static/"))))
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/")))

	return r
}

// ===================== Lazy Client Getters (Advanced Decoupling) =====================

func (g *gateway) getBiometricClient() (biometricpb.BiometricServiceClient, error) {
	g.biometricMu.Lock()
	defer g.biometricMu.Unlock()

	if g.biometricClient != nil {
		return g.biometricClient, nil
	}

	if g.biometricAddr == "" {
		return nil, fmt.Errorf("biometric service address not configured")
	}

	conn, err := grpc.NewClient(g.biometricAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		grpc.WithUnaryInterceptor(middleware.CorrelationIDGRPCClient()),
	)
	if err != nil {
		g.log.Warn("Failed to create biometric client on demand", zap.Error(err))
		return nil, err
	}

	g.biometricClient = biometricpb.NewBiometricServiceClient(conn)
	g.log.Info("Biometric client initialized on first use", zap.String("addr", g.biometricAddr))
	return g.biometricClient, nil
}

func (g *gateway) getTrainingClient() (trainingpb.TrainingServiceClient, error) {
	g.trainingMu.Lock()
	defer g.trainingMu.Unlock()

	if g.trainingClient != nil {
		return g.trainingClient, nil
	}

	if g.trainingAddr == "" {
		return nil, fmt.Errorf("training service address not configured")
	}

	conn, err := grpc.NewClient(g.trainingAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		grpc.WithUnaryInterceptor(middleware.CorrelationIDGRPCClient()),
	)
	if err != nil {
		g.log.Warn("Failed to create training client on demand", zap.Error(err))
		return nil, err
	}

	g.trainingClient = trainingpb.NewTrainingServiceClient(conn)
	g.log.Info("Training client initialized on first use", zap.String("addr", g.trainingAddr))
	return g.trainingClient, nil
}

func (g *gateway) proxyToDeviceAggregator(w http.ResponseWriter, r *http.Request) {
	// SSRF protection: construct target from fixed internal service + validated path
	// Path is already validated to point to device-aggregator endpoints via route registration
	target, _ := url.JoinPath("http://device-aggregator:8083", r.URL.Path)

	outReq, _ := http.NewRequestWithContext(r.Context(), r.Method, target, r.Body) // #nosec G704
	outReq.Header = r.Header.Clone()
	outReq.Header.Set("X-User-ID", r.Header.Get("X-User-ID"))
	outReq.Header.Set("X-Correlation-ID", middleware.GetCorrelationID(r.Context()))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, _ := client.Do(outReq) // #nosec G704
	if resp == nil {
		http.Error(w, "Сервис aggregator недоступен", http.StatusServiceUnavailable)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
