package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
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

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	biometricpb "github.com/MAMUER/project/api/gen/biometric"
	trainingpb "github.com/MAMUER/project/api/gen/training"
	userpb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/auth"
	"github.com/MAMUER/project/internal/cache"
	"github.com/MAMUER/project/internal/config"
	grpctls "github.com/MAMUER/project/internal/grpc"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/middleware"
	"github.com/MAMUER/project/internal/telemetry"
)

type gateway struct {
	userClient            userpb.UserServiceClient
	userConn              *grpc.ClientConn
	biometricAddr         string
	biometricClient       biometricpb.BiometricServiceClient
	trainingAddr          string
	trainingClient        trainingpb.TrainingServiceClient
	classifierURL         string
	mlGeneratorURL        string
	deviceConnectorURL    string
	log                   *logger.Logger
	jwtPrivateKeyPEM      string
	jwtPublicKeyPEM       string
	responseSigningSecret string
	db                    *sql.DB
	sessionStore          *cache.SessionStore
	rdb                   *redis.Client
	rmqCh                 *amqp.Channel
	mlAsync               bool
	requestDuration       *prometheus.HistogramVec
	requestTotal          *prometheus.CounterVec
	errorTotal            *prometheus.CounterVec

	googleOAuthConfig *oauth2.Config

	biometricMu      sync.Mutex
	trainingMu       sync.Mutex
	totpRateLimiters sync.Map
}

func main() {
	log := logger.New("gateway")
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

	ctx := context.Background()
	metrics := newGatewayMetrics()
	cfg := loadGatewayConfig(log)

	db, closeDB := openGatewayDatabase(log, config.GetEnv("DATABASE_URL"))
	defer closeDB()

	redisPassword := config.GetEnv("REDIS_PASSWORD")
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
	mainRouterHandler := telemetry.HTTPMiddleware(log)(mainRouter)
	startGatewayServers(log, cfg, mainRouterHandler)
}

func loadGatewayConfig(log *logger.Logger) gatewayConfig {
	cfg := gatewayConfig{
		port:                 config.GetEnv("GATEWAY_PORT", "8080"),
		userServiceAddr:      config.GetEnv("USER_SERVICE_ADDR", "localhost:50051"),
		biometricServiceAddr: config.GetEnv("BIOMETRIC_SERVICE_ADDR", "localhost:50052"),
		trainingServiceAddr:  config.GetEnv("TRAINING_SERVICE_ADDR", "localhost:50053"),
		classifierURL:        config.GetEnv("CLASSIFIER_URL", "http://classifier:8001"),
		mlGeneratorURL:       config.GetEnv("ML_GENERATOR_URL", "http://ml-generator:8002"),
		deviceConnectorURL:   config.GetEnv("DEVICE_CONNECTOR_URL", "http://localhost:8082"),
		mlAsync:              config.GetEnv("ML_ASYNC", "false") == "true",
		redisAddr:            redisAddress(),
		appBaseURL:           config.GetEnv("APP_BASE_URL"),
		googleClientID:       config.GetEnv("GOOGLE_CLIENT_ID"),
		googleClientSecret:   config.GetEnv("GOOGLE_CLIENT_SECRET"),
	}

	if err := validateMLGeneratorURL(cfg.mlGeneratorURL); err != nil {
		log.Fatal("invalid ML_GENERATOR_URL", zap.Error(err))
	}

	cfg.rabbitmqURL = config.GetEnv("RABBITMQ_URL", "amqp://localhost:5672/")

	jwtPrivateKeyPEM := config.GetEnv("JWT_PRIVATE_KEY_PEM")
	if jwtPrivateKeyPEM == "" {
		log.Fatal("JWT_PRIVATE_KEY_PEM environment variable is required")
	}
	jwtPublicKeyPEM := config.GetEnv("JWT_PUBLIC_KEY_PEM")
	if jwtPublicKeyPEM == "" {
		log.Fatal("JWT_PUBLIC_KEY_PEM environment variable is required")
	}
	responseSigningSecret := config.GetEnv("RESPONSE_SIGNING_SECRET")
	if responseSigningSecret == "" {
		log.Fatal("RESPONSE_SIGNING_SECRET environment variable is required")
	}
	cfg.jwtPrivateKeyPEM = jwtPrivateKeyPEM
	cfg.jwtPublicKeyPEM = jwtPublicKeyPEM
	cfg.responseSigningSecret = responseSigningSecret

	cfg.publicHost = extractPublicHost(cfg.appBaseURL)
	cfg.googleOAuthConfig = buildGoogleOAuthConfig(log, cfg)
	return cfg
}

func extractPublicHost(appBaseURL string) string {
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

	redirectURL := config.GetEnv("GOOGLE_REDIRECT_URL")
	if redirectURL == "" && cfg.appBaseURL != "" {
		redirectURL = cfg.appBaseURL + "/api/v1/auth/google/callback"
	}

	log.Info("Google OAuth configured", zap.String("redirect_url", redirectURL))
	return &oauth2.Config{
		ClientID:     cfg.googleClientID,
		ClientSecret: cfg.googleClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}
}

func redisAddress() string {
	redisHost := config.GetEnv("REDIS_HOST", "redis")
	return redisHost + ":6379"
}

type gatewayMetrics struct {
	requestDuration *prometheus.HistogramVec
	requestTotal    *prometheus.CounterVec
	errorTotal      *prometheus.CounterVec
}

type gatewayConfig struct {
	port                  string
	userServiceAddr       string
	biometricServiceAddr  string
	trainingServiceAddr   string
	classifierURL         string
	mlGeneratorURL        string
	deviceConnectorURL    string
	rabbitmqURL           string
	redisAddr             string
	jwtPrivateKeyPEM      string
	jwtPublicKeyPEM       string
	responseSigningSecret string
	appBaseURL            string
	publicHost            string
	googleClientID        string
	googleClientSecret    string
	mlAsync               bool
	googleOAuthConfig     *oauth2.Config
}

type gatewayTLSConfig struct {
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

func validateMLGeneratorURL(mlGeneratorURL string) error {
	parsedURL, err := url.Parse(mlGeneratorURL)
	if err != nil {
		return fmt.Errorf("parse ml generator url: %w", err)
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

func connectUserService(_ context.Context, log *logger.Logger, userServiceAddr string) (*grpc.ClientConn, userpb.UserServiceClient) {
	tlsCreds, _ := grpctls.GetClientTLSCredentials()
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithDefaultCallOptions(grpc.WaitForReady(true), grpc.MaxCallRecvMsgSize(10<<20)))
	if tlsCreds != nil {
		opts = append(opts, grpc.WithTransportCredentials(tlsCreds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	userConn, err := grpc.NewClient(userServiceAddr, opts...)
	if err != nil {
		log.Fatal("Failed to connect to user service", zap.Error(err))
	}

	return userConn, userpb.NewUserServiceClient(userConn)
}

func buildGateway(log *logger.Logger, cfg gatewayConfig, metrics gatewayMetrics, db *sql.DB, sessionStore *cache.SessionStore, rdb *redis.Client, rmqCh *amqp.Channel, userClient userpb.UserServiceClient, mlAsync bool) *gateway {
	return &gateway{
		userClient:            userClient,
		biometricAddr:         cfg.biometricServiceAddr,
		trainingAddr:          cfg.trainingServiceAddr,
		classifierURL:         cfg.classifierURL,
		mlGeneratorURL:        cfg.mlGeneratorURL,
		deviceConnectorURL:    cfg.deviceConnectorURL,
		log:                   log,
		jwtPrivateKeyPEM:      cfg.jwtPrivateKeyPEM,
		jwtPublicKeyPEM:       cfg.jwtPublicKeyPEM,
		responseSigningSecret: cfg.responseSigningSecret,
		db:                    db,
		sessionStore:          sessionStore,
		rdb:                   rdb,
		rmqCh:                 rmqCh,
		mlAsync:               mlAsync,
		requestDuration:       metrics.requestDuration,
		requestTotal:          metrics.requestTotal,
		errorTotal:            metrics.errorTotal,
		googleOAuthConfig:     cfg.googleOAuthConfig,
	}
}

func startGatewayServers(log *logger.Logger, cfg gatewayConfig, mainRouter http.Handler) {
	tlsCfg := detectTLSMode(log, cfg)
	httpsSrv := &http.Server{
		Addr:              ":8443",
		Handler:           mainRouter,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 15 * time.Second,
	}

	httpHandler := mainRouter
	if tlsCfg.available {
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
		log.Info("HTTP server starting", zap.String("port", cfg.port), zap.Bool("redirect_to_https", tlsCfg.available))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server failed", zap.Error(err))
		}
	}()

	if tlsCfg.available {
		go func() {
			log.Info("HTTPS server starting",
				zap.String("port", "8443"),
				zap.String("cert", tlsCfg.certFile),
				zap.String("classifier", cfg.classifierURL),
				zap.String("ml_generator", cfg.mlGeneratorURL))
			httpsSrv.TLSConfig = &tls.Config{
				MinVersion: tls.VersionTLS13,
				MaxVersion: tls.VersionTLS13,
			}
			if err := httpsSrv.ListenAndServeTLS(tlsCfg.certFile, tlsCfg.keyFile); err != nil && err != http.ErrServerClosed {
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
	if tlsCfg.available {
		_ = httpsSrv.Shutdown(ctx)
	}
	log.Info("Servers stopped")
}

func detectTLSMode(log *logger.Logger, _ gatewayConfig) gatewayTLSConfig {
	mode := gatewayTLSConfig{
		certFile: os.Getenv("TLS_CERT_FILE"),
		keyFile:  os.Getenv("TLS_KEY_FILE"),
	}
	if mode.certFile == "" || mode.keyFile == "" {
		return mode
	}

	if _, err := os.Stat(filepath.Clean(mode.certFile)); err != nil {
		log.Warn("TLS certificate file not found, falling back to HTTP-only mode",
			zap.String("cert_file", mode.certFile), zap.Error(err))
		return gatewayTLSConfig{}
	}
	if _, err := os.Stat(filepath.Clean(mode.keyFile)); err != nil {
		log.Warn("TLS key file not found, falling back to HTTP-only mode",
			zap.String("key_file", mode.keyFile), zap.Error(err))
		return gatewayTLSConfig{}
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
func (g *gateway) registerRoutes() *chi.Mux {
	r := chi.NewRouter()

	// ========== Security middleware (applied to ALL routes) ==========
	r.Use(middleware.RemoveServerHeader)
	r.Use(middleware.ErrorPages)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.HTMLNonceInject)
	r.Use(middleware.RecoveryMiddleware(g.log.Logger))
	r.Use(middleware.RateLimit)
	r.Use(middleware.RequestID)
	r.Use(middleware.LoggingMiddleware(g.log.Logger, g.requestDuration, g.requestTotal, g.errorTotal))
	r.Use(middleware.CorrelationIDHTTP)

	authMiddleware := middleware.AuthMiddleware(g.jwtPublicKeyPEM, g.log.Logger)

	g.registerPublicRoutes(r)
	g.registerProtectedRoutes(r, authMiddleware)
	g.registerAdminRoutes(r, authMiddleware)

	// ========== Static files ==========
	fsStatic := http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static/")))
	fsRoot := http.FileServer(http.Dir("./web/"))
	r.Get("/static/*", fsStatic.ServeHTTP)
	r.Get("/*", fsRoot.ServeHTTP)

	return r
}

// registerPublicRoutes registers routes that do not require authentication.
func (g *gateway) registerPublicRoutes(r chi.Router) {
	// ========== Public routes (без авторизации) ==========
	r.With(middleware.AuthRateLimit).Post("/api/v1/register", g.registerHandler)
	r.Post("/api/v1/register/invite", g.registerWithInviteHandler)
	r.Post("/api/v1/invite/validate", g.validateInviteCodeHandler)
	r.With(middleware.AuthRateLimit).Post("/api/v1/login", g.loginHandler)
	r.Post("/api/v1/auth/confirm", g.confirmEmailHandler)
	r.Get("/api/v1/auth/verify-status", g.checkVerificationStatusHandler)
	r.Get("/api/v1/auth/google", g.googleLoginHandler)
	r.Get("/api/v1/auth/google/callback", g.googleCallbackHandler)
	r.Get("/health", g.healthHandler)
	// Webhook endpoint - БЕЗ authMiddleware
	r.Post("/api/v1/devices/withings/webhook", g.proxyToDeviceAggregator)
	// Metrics endpoint
	r.Method("GET", "/metrics", promhttp.Handler())

	// JWKS endpoint for JWT public key distribution
	r.Get("/.well-known/jwks.json", g.jwksHandler)

	// CSP violation reports (browser Reporting API / report-uri) -> ELK
	r.Post("/api/security/csp-report", g.cspReportHandler)

	// Email confirmation page
	r.Get("/confirm", g.emailConfirmPageHandler)

	// 2FA TOTP verify (public route - uses temp_token)
	r.Post("/api/v1/auth/2fa/verify", g.verifyTOTPHandler)

	// Refresh token (public - uses opaque refresh token)
	r.Post("/api/v1/auth/refresh", g.refreshHandler)
}

// registerProtectedRoutes registers routes under /api/v1 that require authentication.
func (g *gateway) registerProtectedRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(middleware.UserRateLimit)

		// Profile
		r.Get("/profile", g.getProfileHandler)
		r.Put("/profile", g.updateProfileHandler)
		r.Delete("/profile", g.deleteProfileHandler)

		// Health features
		r.Get("/health/conditions", g.listHealthConditionsHandler)
		r.Post("/health/conditions", g.upsertHealthConditionHandler)
		r.Delete("/health/conditions/{condition_id}", g.deleteHealthConditionHandler)
		r.Get("/health/body-composition", g.listBodyCompositionHandler)
		r.Post("/health/body-composition", g.createBodyCompositionHandler)
		r.Get("/health/menstrual-cycles", g.listMenstrualCyclesHandler)
		r.Post("/health/menstrual-cycles", g.createMenstrualCycleHandler)
		r.Put("/health/menstrual-cycles/{cycle_id}", g.updateMenstrualCycleHandler)
		r.Delete("/health/menstrual-cycles/{cycle_id}", g.deleteMenstrualCycleHandler)
		r.Post("/health/sync/flo", g.syncFloHandler)
		r.Post("/health/sync/okok", g.syncOKOKHandler)

		// 2FA TOTP (protected routes - require auth)
		r.Post("/auth/2fa/setup", g.setupTOTPHandler)
		r.Post("/auth/2fa/confirm", g.confirmTOTPHandler)
		r.Get("/auth/2fa/status", g.totpStatusHandler)
		r.Post("/auth/2fa/disable", g.disableTOTPHandler)

		// Biometrics
		r.Post("/biometrics", g.addBiometricRecordHandler)
		r.Get("/biometrics", g.getBiometricRecordsHandler)

		// Training
		r.Get("/training/plans", g.listPlansHandler)
		r.Post("/training/generate", g.generatePlanHandler)
		r.Get("/training/plans/{plan_id}", g.getPlanHandler)
		r.Get("/training/progress", g.getProgressHandler)
		r.Post("/training/complete", g.completeWorkoutHandler)

		// ML
		r.Post("/ml/classify", g.classifyHandler)
		r.Post("/ml/generate-plan", g.mlGenerateHandler)

		// Devices — проксирование на device-connector
		r.Post("/devices/register", g.proxyToDeviceConnector)
		r.Post("/devices/{device_id}/ingest", g.proxyToDeviceConnector)
		r.Get("/devices", g.listDevicesHandler)
		r.Post("/devices", g.registerDeviceHandler)

		// Logout
		r.Post("/logout", g.logoutHandler)

		// Device aggregator proxy
		r.Get("/devices/fitbit/auth", g.proxyToDeviceAggregator)
		r.Get("/devices/fitbit/callback", g.proxyToDeviceAggregator)
		r.Post("/devices/fitbit/disconnect", g.proxyToDeviceAggregator)
		r.Get("/devices/providers", g.proxyToDeviceAggregator)

		r.Get("/devices/withings/auth", g.proxyToDeviceAggregator)
		r.Get("/devices/withings/callback", g.proxyToDeviceAggregator)
		r.Post("/devices/withings/disconnect", g.proxyToDeviceAggregator)
	})
}

// registerAdminRoutes registers /api/v1/admin routes that require the admin role.
func (g *gateway) registerAdminRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/admin", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(middleware.RequireRole(g.db, g.log.Logger, "admin"))
		r.Get("/users", g.adminListUsersHandler)
		r.Get("/invites", g.adminListInvitesHandler)                 // ← НОВОЕ
		r.Post("/invites", g.adminCreateInviteHandler)               // ← НОВОЕ
		r.Post("/invites/{code}/revoke", g.adminRevokeInviteHandler) // ← НОВОЕ
	})
}

// ===================== Lazy Client Getters (Advanced Decoupling) =====================

func (g *gateway) getBiometricClient() (biometricpb.BiometricServiceClient, error) {
	g.biometricMu.Lock()
	defer g.biometricMu.Unlock()

	if g.biometricClient != nil {
		return g.biometricClient, nil
	}

	if g.biometricAddr == "" {
		return nil, errors.New("biometric service address not configured")
	}

	var dialOpts []grpc.DialOption
	dialOpts = append(dialOpts, grpc.WithDefaultCallOptions(grpc.WaitForReady(true), grpc.MaxCallRecvMsgSize(10<<20)))
	dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(middleware.CorrelationIDGRPCClient()))
	if tlsCreds, err2 := grpctls.GetClientTLSCredentials(); err2 == nil && tlsCreds != nil {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(tlsCreds))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	conn, err := grpc.NewClient(g.biometricAddr, dialOpts...)
	if err != nil {
		g.log.Warn("Failed to create biometric client on demand", zap.Error(err))
		return nil, errors.New("create biometric client: " + err.Error())
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
		return nil, errors.New("training service address not configured")
	}

	var dialOpts []grpc.DialOption
	dialOpts = append(dialOpts, grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))
	if tlsCreds, err2 := grpctls.GetClientTLSCredentials(); err2 == nil && tlsCreds != nil {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(tlsCreds))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	conn, err := grpc.NewClient(g.trainingAddr, dialOpts...)
	if err != nil {
		g.log.Warn("Failed to create training client on demand", zap.Error(err))
		return nil, errors.New("create training client: " + err.Error())
	}

	g.trainingClient = trainingpb.NewTrainingServiceClient(conn)
	g.log.Info("Training client initialized on first use", zap.String("addr", g.trainingAddr))
	return g.trainingClient, nil
}

func (g *gateway) proxyToDeviceAggregator(w http.ResponseWriter, r *http.Request) {
	// SSRF protection: construct target from fixed internal service + validated path
	// Path is already validated to point to device-aggregator endpoints via route registration
	target, _ := url.JoinPath("http://device-aggregator:8083", r.URL.Path)

	// #nosec G704 -- intentional proxy to hardcoded internal service
	outReq, _ := http.NewRequestWithContext(r.Context(), r.Method, target, r.Body)
	outReq.Header = r.Header.Clone()
	outReq.Header.Set("X-User-ID", r.Header.Get("X-User-ID"))
	outReq.Header.Set("X-Correlation-ID", middleware.GetCorrelationID(r.Context()))

	client := &http.Client{Timeout: 10 * time.Second}
	// #nosec G704 -- intentional proxy to hardcoded internal service
	resp, _ := client.Do(outReq)
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

	// #nosec G704 -- intentional proxy to hardcoded internal service
	outReq, _ := http.NewRequestWithContext(r.Context(), r.Method, target, r.Body)
	outReq.Header = r.Header.Clone()
	outReq.Header.Set("X-User-ID", r.Header.Get("X-User-ID"))
	outReq.Header.Set("X-Correlation-ID", middleware.GetCorrelationID(r.Context()))

	client := &http.Client{Timeout: 10 * time.Second}
	// #nosec G704 -- intentional proxy to hardcoded internal service
	resp, err := client.Do(outReq)
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

func (g *gateway) jwksHandler(w http.ResponseWriter, r *http.Request) {
	if g.jwtPublicKeyPEM == "" {
		http.Error(w, "JWT public key not configured", http.StatusInternalServerError)
		return
	}

	body, err := auth.PublicKeyPEMToJWKS(g.jwtPublicKeyPEM)
	if err != nil {
		g.log.Error("Failed to build JWKS", zap.Error(err))
		http.Error(w, "failed to build JWKS", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
