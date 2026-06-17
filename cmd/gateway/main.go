package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
	oauth2google "golang.org/x/oauth2/google"

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
	classifierURL      string
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

	googleOAuthConfig *oauth2.Config

	biometricMu      sync.Mutex
	trainingMu       sync.Mutex
	totpRateLimiters sync.Map
}

//nolint:gocognit,gocyclo,funlen // Complexity due to sequential init of multiple services
func getEnvOrFile(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	filePath := os.Getenv(key + "_FILE")
	if filePath == "" {
		return ""
	}

	value, err := readSecretFile(filePath)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(value))
}

func readSecretFile(filePath string) ([]byte, error) {
	cleanPath := filepath.Clean(filePath)
	if cleanPath == "." || cleanPath == string(filepath.Separator) {
		return nil, fmt.Errorf("secret file path is invalid")
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("secret file path is a directory")
	}

	return os.ReadFile(cleanPath) // #nosec G304 -- path is checked with os.Stat before reading
}

func main() {
	log := logger.New("gateway")
	defer func() {
		if syncErr := log.Sync(); syncErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", syncErr)
		}
	}()

	ctx := context.Background()
	metrics := newGatewayMetrics()
	cfg := loadGatewayConfig(log)

	db, closeDB := openGatewayDatabase(log, os.Getenv("DATABASE_URL"))
	defer closeDB()

	redisPassword := getEnvOrFile("REDIS_PASSWORD")
	const redisMaxRetries = 10
	const redisRetryDelay = 3 * time.Second

	mlAsync := cfg.mlAsync
	var asyncRDB *redis.Client
	if mlAsync {
		var redisConnected bool
		asyncRDB, redisConnected = connectRedis(ctx, log, cfg.redisAddr, redisPassword, 1, redisMaxRetries, redisRetryDelay)
		if !redisConnected {
			mlAsync = false
		}
	}

	rmqCh, rmqClose, rabbitMQConnected := connectRabbitMQ(log, cfg.rabbitmqURL, mlAsync)
	if !rabbitMQConnected {
		mlAsync = false
		if asyncRDB != nil {
			if closeErr := asyncRDB.Close(); closeErr != nil {
				log.Warn("Failed to close async Redis client", zap.Error(closeErr))
			}
		}
	}
	if rmqClose != nil {
		defer rmqClose()
	}

	rdb, redisConnected := connectRedis(ctx, log, cfg.redisAddr, redisPassword, 0, redisMaxRetries, redisRetryDelay)
	var sessionStore *cache.SessionStore
	if redisConnected {
		sessionStore = cache.NewSessionStoreFromRedis(rdb)
	}

	userConn, userClient := connectUserService(ctx, log, cfg.userServiceAddr)
	defer func() {
		if closeErr := userConn.Close(); closeErr != nil {
			log.Error("Failed to close user service connection", zap.Error(closeErr))
		}
	}()

	g := buildGateway(log, cfg, metrics, db, sessionStore, rdb, rmqCh, userClient, mlAsync)
	mainRouter := g.registerRoutes()
	startGatewayServers(log, cfg, mainRouter)
}

type gatewayMetrics struct {
	requestDuration *prometheus.HistogramVec
	requestTotal    *prometheus.CounterVec
	errorTotal      *prometheus.CounterVec
}

type gatewayConfig struct {
	port                 string
	userServiceAddr      string
	biometricServiceAddr string
	trainingServiceAddr  string
	classifierURL        string
	mlGeneratorURL       string
	deviceConnectorURL   string
	rabbitmqURL          string
	redisAddr            string
	jwtSecret            string
	appBaseURL           string
	publicHost           string
	googleClientID       string
	googleClientSecret   string
	mlAsync              bool
	googleOAuthConfig    *oauth2.Config
}

type tlsMode struct {
	available bool
	certFile  string
	keyFile   string
}

func newGatewayMetrics() gatewayMetrics {
	metrics := gatewayMetrics{
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "request_duration_seconds",
				Help:    "Duration of HTTP requests",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"service", "endpoint", "method", "status"},
		),
		requestTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "request_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"service", "endpoint", "method"},
		),
		errorTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "error_total",
				Help: "Total number of errors",
			},
			[]string{"service", "error_code", "endpoint"},
		),
	}

	prometheus.MustRegister(metrics.requestDuration, metrics.requestTotal, metrics.errorTotal)
	return metrics
}

func loadGatewayConfig(log *logger.Logger) gatewayConfig {
	cfg := gatewayConfig{
		port:                 defaultEnv("GATEWAY_PORT", "8080"),
		userServiceAddr:      defaultEnv("USER_SERVICE_ADDR", "localhost:50051"),
		biometricServiceAddr: defaultEnv("BIOMETRIC_SERVICE_ADDR", "localhost:50052"),
		trainingServiceAddr:  defaultEnv("TRAINING_SERVICE_ADDR", "localhost:50053"),
		classifierURL:        defaultEnv("CLASSIFIER_URL", "http://classifier:8001"),
		mlGeneratorURL:       defaultEnv("ML_GENERATOR_URL", "http://ml-generator:8002"),
		deviceConnectorURL:   defaultEnv("DEVICE_CONNECTOR_URL", "http://localhost:8082"),
		mlAsync:              envBool("ML_ASYNC"),
		redisAddr:            redisAddress(),
		appBaseURL:           os.Getenv("APP_BASE_URL"),
		googleClientID:       os.Getenv("GOOGLE_CLIENT_ID"),
		googleClientSecret:   os.Getenv("GOOGLE_CLIENT_SECRET"),
	}

	if err := validateMLGeneratorURL(cfg.mlGeneratorURL); err != nil {
		log.Fatal("invalid ML_GENERATOR_URL", zap.Error(err))
	}

	rabbitmqURL := getEnvOrFile("RABBITMQ_URL")
	if rabbitmqURL == "" {
		rabbitmqURL = "amqp://localhost:5672/"
	}
	cfg.rabbitmqURL = rabbitmqURL

	jwtSecret := getEnvOrFile("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	cfg.jwtSecret = jwtSecret

	cfg.publicHost = publicHostFromBaseURL(cfg.appBaseURL)
	cfg.googleOAuthConfig = buildGoogleOAuthConfig(log, cfg)
	return cfg
}

func defaultEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string) bool {
	value := os.Getenv(key)
	return value == "true" || value == "True" || value == "1"
}

func redisAddress() string {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "redis"
	}
	return redisHost + ":6379"
}

func publicHostFromBaseURL(appBaseURL string) string {
	if appBaseURL == "" {
		return ""
	}
	parsedAppURL, err := url.Parse(appBaseURL)
	if err != nil {
		return ""
	}
	return parsedAppURL.Host
}

func buildGoogleOAuthConfig(log *logger.Logger, cfg gatewayConfig) *oauth2.Config {
	if cfg.googleClientID == "" || cfg.googleClientSecret == "" {
		log.Warn("Google OAuth not configured: GOOGLE_CLIENT_ID or GOOGLE_CLIENT_SECRET missing")
		return nil
	}

	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
	if redirectURL == "" && cfg.appBaseURL != "" {
		redirectURL = cfg.appBaseURL + "/api/v1/auth/google/callback"
	}

	log.Info("Google OAuth configured", zap.String("redirect_url", redirectURL))
	return &oauth2.Config{
		ClientID:     cfg.googleClientID,
		ClientSecret: cfg.googleClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint:     oauth2google.Endpoint,
	}
}

func validateMLGeneratorURL(mlGeneratorURL string) error {
	parsedURL, err := url.Parse(mlGeneratorURL)
	if err != nil {
		return err
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
		return fmt.Errorf("host %q is not allowed", parsedURL.Host)
	}
	return nil
}

func openGatewayDatabase(log *logger.Logger, dbURL string) (*sql.DB, func()) {
	if dbURL == "" {
		return nil, func() {}
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to open database", zap.Error(err))
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	if pingErr := db.PingContext(context.Background()); pingErr != nil {
		log.Fatal("Failed to ping database", zap.Error(pingErr))
	}

	return db, func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Error("Failed to close database connection", zap.Error(closeErr))
		}
	}
}

func connectRedis(ctx context.Context, log *logger.Logger, redisAddr, password string, dbNum, maxRetries int, retryDelay time.Duration) (*redis.Client, bool) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: password,
		DB:       dbNum,
	})
	connected := waitForRedis(ctx, log, rdb, redisAddr, maxRetries, retryDelay)
	if !connected {
		if closeErr := rdb.Close(); closeErr != nil {
			log.Warn("Failed to close Redis client", zap.Error(closeErr))
		}
		return nil, false
	}
	return rdb, true
}

func waitForRedis(ctx context.Context, log *logger.Logger, rdb *redis.Client, redisAddr string, maxRetries int, retryDelay time.Duration) bool {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		pingErr := rdb.Ping(ctx).Err()
		if pingErr == nil {
			log.Info("Redis connected", zap.String("addr", redisAddr), zap.Int("attempt", attempt))
			return true
		}

		if attempt < maxRetries {
			log.Warn("Redis unavailable, retrying",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", maxRetries),
				zap.Duration("retry_delay", retryDelay),
				zap.Error(pingErr))
			time.Sleep(retryDelay)
			continue
		}

		log.Warn("Redis unavailable after all retries",
			zap.Int("attempts", maxRetries),
			zap.Error(pingErr))
	}
	return false
}

func connectRabbitMQ(log *logger.Logger, rabbitmqURL string, mlAsync bool) (*amqp.Channel, func(), bool) {
	if !mlAsync {
		return nil, nil, false
	}

	rmqConn, err := amqp.Dial(rabbitmqURL)
	if err != nil {
		log.Warn("RabbitMQ unavailable, async ML mode disabled", zap.Error(err))
		return nil, nil, false
	}

	rmqCh, err := rmqConn.Channel()
	if err != nil {
		log.Warn("Failed to create RabbitMQ channel, async ML mode disabled", zap.Error(err))
		if closeErr := rmqConn.Close(); closeErr != nil {
			log.Warn("Failed to close RabbitMQ connection", zap.Error(closeErr))
		}
		return nil, nil, false
	}

	_, _ = rmqCh.QueueDeclare("ml.classify", true, false, false, false, nil)
	_, _ = rmqCh.QueueDeclare("ml.generate", true, false, false, false, nil)
	log.Info("RabbitMQ connected for async ML jobs", zap.String("url", rabbitmqURL))

	return rmqCh, func() {
		if closeErr := rmqConn.Close(); closeErr != nil {
			log.Warn("Failed to close RabbitMQ connection", zap.Error(closeErr))
		}
	}, true
}

func connectUserService(ctx context.Context, log *logger.Logger, userServiceAddr string) (*grpc.ClientConn, userpb.UserServiceClient) {
	userConn, err := grpc.NewClient(userServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true), grpc.MaxCallRecvMsgSize(10<<20)),
	)
	if err != nil {
		log.Fatal("Failed to connect to user service", zap.Error(err))
	}

	return userConn, userpb.NewUserServiceClient(userConn)
}

func buildGateway(log *logger.Logger, cfg gatewayConfig, metrics gatewayMetrics, db *sql.DB, sessionStore *cache.SessionStore, rdb *redis.Client, rmqCh *amqp.Channel, userClient userpb.UserServiceClient, mlAsync bool) *gateway {
	return &gateway{
		userClient:         userClient,
		biometricAddr:      cfg.biometricServiceAddr,
		trainingAddr:       cfg.trainingServiceAddr,
		classifierURL:      cfg.classifierURL,
		mlGeneratorURL:     cfg.mlGeneratorURL,
		deviceConnectorURL: cfg.deviceConnectorURL,
		log:                log,
		jwtSecret:          cfg.jwtSecret,
		db:                 db,
		sessionStore:       sessionStore,
		rdb:                rdb,
		rmqCh:              rmqCh,
		mlAsync:            mlAsync,
		requestDuration:    metrics.requestDuration,
		requestTotal:       metrics.requestTotal,
		errorTotal:         metrics.errorTotal,
		googleOAuthConfig:  cfg.googleOAuthConfig,
	}
}

func startGatewayServers(log *logger.Logger, cfg gatewayConfig, mainRouter *mux.Router) {
	tls := detectTLSMode(log, cfg)
	httpsSrv := &http.Server{
		Addr:              ":8443",
		Handler:           mainRouter,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 15 * time.Second,
	}

	var httpHandler http.Handler = mainRouter
	if tls.available {
		httpHandler = buildHTTPRedirectHandler(cfg.publicHost, cfg.port)
	} else {
		log.Info("TLS is not available, serving application directly over HTTP")
	}

	httpSrv := &http.Server{
		Addr:              ":" + cfg.port,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		ReadHeaderTimeout: 15 * time.Second,
		Handler:           httpHandler,
	}

	go func() {
		log.Info("HTTP server starting", zap.String("port", cfg.port), zap.Bool("redirect_to_https", tls.available))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server failed", zap.Error(err))
		}
	}()

	if tls.available {
		go func() {
			log.Info("HTTPS server starting",
				zap.String("port", "8443"),
				zap.String("cert", tls.certFile),
				zap.String("classifier", cfg.classifierURL),
				zap.String("ml_generator", cfg.mlGeneratorURL))
			if err := httpsSrv.ListenAndServeTLS(tls.certFile, tls.keyFile); err != nil && err != http.ErrServerClosed {
				log.Fatal("Failed to start HTTPS server", zap.Error(err))
			}
		}()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = httpSrv.Shutdown(ctx)
	if tls.available {
		_ = httpsSrv.Shutdown(ctx)
	}
	log.Info("Servers stopped")
}

func detectTLSMode(log *logger.Logger, cfg gatewayConfig) tlsMode {
	mode := tlsMode{
		certFile: os.Getenv("TLS_CERT_FILE"),
		keyFile:  os.Getenv("TLS_KEY_FILE"),
	}
	if mode.certFile == "" || mode.keyFile == "" {
		return mode
	}

	if _, err := os.Stat(filepath.Clean(mode.certFile)); err != nil {
		log.Warn("TLS certificate file not found, falling back to HTTP-only mode",
			zap.String("cert_file", mode.certFile), zap.Error(err))
		return tlsMode{}
	}
	if _, err := os.Stat(filepath.Clean(mode.keyFile)); err != nil {
		log.Warn("TLS key file not found, falling back to HTTP-only mode",
			zap.String("key_file", mode.keyFile), zap.Error(err))
		return tlsMode{}
	}

	mode.available = true
	return mode
}

func buildHTTPRedirectHandler(publicHost, port string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok","service":"gateway"}`))
			return
		}

		host := publicHost
		if host == "" {
			host = r.Host
			if h, _, err := net.SplitHostPort(host); err == nil {
				host = h
			}
			if host == "" || net.ParseIP(host) != nil {
				http.Error(w, "Invalid Host", http.StatusBadRequest)
				return
			}
		}

		requestURI := r.URL.RequestURI()
		if strings.HasPrefix(requestURI, "//") || strings.HasPrefix(requestURI, "\\") {
			http.Error(w, "Invalid request URI", http.StatusBadRequest)
			return
		}

		redirectURL := &url.URL{Scheme: "https", Host: host, Path: r.URL.Path, RawQuery: r.URL.RawQuery, Fragment: r.URL.Fragment}
		if port != "" && port != "80" && port != "443" {
			redirectURL.Host = host + ":8443"
		}

		target := redirectURL.String()
		parsed, err := url.Parse(target)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() != host {
			http.Error(w, "Invalid redirect target", http.StatusBadRequest)
			return
		}

		w.Header().Set("Location", target)
		w.WriteHeader(http.StatusMovedPermanently)
	})
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
	r.HandleFunc("/api/v1/auth/google", g.googleLoginHandler).Methods("GET")
	r.HandleFunc("/api/v1/auth/google/callback", g.googleCallbackHandler).Methods("GET")
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

	// 2FA TOTP (protected routes - require auth)
	protected.HandleFunc("/auth/2fa/setup", g.setupTOTPHandler).Methods("POST")
	protected.HandleFunc("/auth/2fa/confirm", g.confirmTOTPHandler).Methods("POST")
	protected.HandleFunc("/auth/2fa/status", g.totpStatusHandler).Methods("GET")
	protected.HandleFunc("/auth/2fa/disable", g.disableTOTPHandler).Methods("POST")

	// 2FA TOTP verify (public route - uses temp_token)
	r.HandleFunc("/api/v1/auth/2fa/verify", g.verifyTOTPHandler).Methods("POST")

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
	protected.HandleFunc("/ml/classify", g.classifyHandler).Methods("POST")
	protected.HandleFunc("/ml/generate", g.mlGenerateHandler).Methods("POST")

	// Devices — проксирование на device-connector
	protected.HandleFunc("/devices/register", g.proxyToDeviceConnector).Methods("POST")
	protected.HandleFunc("/devices/{device_id}/ingest", g.proxyToDeviceConnector).Methods("POST")
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

func (g *gateway) proxyToDeviceConnector(w http.ResponseWriter, r *http.Request) {
	// SSRF protection: construct target from fixed internal service + validated path
	target, _ := url.JoinPath(g.deviceConnectorURL, r.URL.Path)

	outReq, _ := http.NewRequestWithContext(r.Context(), r.Method, target, r.Body) // #nosec G704
	outReq.Header = r.Header.Clone()
	outReq.Header.Set("X-User-ID", r.Header.Get("X-User-ID"))
	outReq.Header.Set("X-Correlation-ID", middleware.GetCorrelationID(r.Context()))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(outReq) // #nosec G704
	if err != nil || resp == nil {
		g.log.Error("Failed to proxy to device-connector", zap.Error(err))
		http.Error(w, "Сервис device-connector недоступен", http.StatusServiceUnavailable)
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
