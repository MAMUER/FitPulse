// Package webhook provides Open Wearables webhook handling.
package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/metrics"
	"github.com/MAMUER/project/internal/sanitize"
)

// Server represents the Open Wearables webhook server
type Server struct {
	port     string
	db       DB
	log      *zap.Logger
	secret   []byte
	server   *http.Server
	shutdown context.CancelFunc
}

// NewServer creates a new webhook server
func NewServer(port string, db DB, log *zap.Logger) *Server {
	secret := []byte(os.Getenv("OPEN_WEARABLES_WEBHOOK_SECRET"))
	if len(secret) == 0 {
		log.Warn("OPEN_WEARABLES_WEBHOOK_SECRET is not set; webhook signature validation is disabled")
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{
		port:     port,
		db:       db,
		log:      log.Named("webhook"),
		secret:   secret,
		shutdown: cancel,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/integrations/open-wearables/webhook", s.handleWebhook)
	mux.HandleFunc("/api/v1/integrations/providers", s.handleListProviders)
	mux.HandleFunc("/api/v1/integrations/open-wearables/disconnect", s.handleDisconnect)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	s.server = &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	return s
}

// Start starts the webhook server
func (s *Server) Start() {
	go func() {
		s.log.Info("Open Wearables webhook server starting", zap.String("port", s.port))
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Fatal("webhook server failed", zap.Error(err))
		}
	}()
}

// Stop gracefully shuts down the webhook server
func (s *Server) Stop(ctx context.Context) error {
	if s.shutdown != nil {
		s.shutdown()
	}
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func (s *Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		WriteResponse(w, http.StatusBadRequest, WebhookResponse{
			Status:  "error",
			Message: "user_id is required",
		})
		return
	}

	storage := NewStorage(s.db, s.log)
	sources, err := storage.GetSources(r.Context(), userID)
	if err != nil {
		// codeql[go/log-injection] user_id is sanitized via sanitize.LogString
		s.log.Error("failed to get sources", zap.Error(err), zap.String("user_id", sanitize.LogString(userID)))
		WriteResponse(w, http.StatusInternalServerError, WebhookResponse{
			Status:  "error",
			Message: "failed to get sources",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"providers": sources,
	})
}

func (s *Server) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	source := r.URL.Query().Get("source")
	if userID == "" || source == "" {
		WriteResponse(w, http.StatusBadRequest, WebhookResponse{
			Status:  "error",
			Message: "user_id and source are required",
		})
		return
	}

	storage := NewStorage(s.db, s.log)
	count, err := storage.DeleteBySource(r.Context(), userID, source)
	if err != nil {
		// codeql[go/log-injection] user_id and source are sanitized via sanitize.LogString
		s.log.Error("failed to disconnect source", zap.Error(err),
			zap.String("user_id", sanitize.LogString(userID)),
			zap.String("source", sanitize.LogString(source)))
		WriteResponse(w, http.StatusInternalServerError, WebhookResponse{
			Status:  "error",
			Message: "failed to disconnect source",
		})
		return
	}

	WriteResponse(w, http.StatusOK, WebhookResponse{
		Status:  "success",
		Message: fmt.Sprintf("disconnected %s, removed %d records", source, count),
	})
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteResponse(w, http.StatusMethodNotAllowed, WebhookResponse{
			Status:  "error",
			Message: "method not allowed",
		})
		return
	}

	if len(s.secret) > 0 {
		if err := ValidateSignature(s.secret, r); err != nil {
			s.log.Warn("invalid webhook signature", zap.Error(err))
			metrics.ErrorTotal.WithLabelValues("webhook", "invalid_signature").Inc()
			WriteResponse(w, http.StatusUnauthorized, WebhookResponse{
				Status:  "error",
				Message: "invalid signature",
			})
			return
		}
	}

	body, err := readBody(r)
	if err != nil {
		s.log.Error("failed to read webhook body", zap.Error(err))
		WriteResponse(w, http.StatusBadRequest, WebhookResponse{
			Status:  "error",
			Message: "invalid request body",
		})
		return
	}

	payload, err := DecodePayload(body)
	if err != nil {
		s.log.Error("failed to decode payload", zap.Error(err))
		WriteResponse(w, http.StatusBadRequest, WebhookResponse{
			Status:  "error",
			Message: fmt.Sprintf("invalid payload: %v", err),
		})
		return
	}

	if time.Since(payload.Timestamp) > 5*time.Minute || time.Until(payload.Timestamp) > 5*time.Minute {
		s.log.Warn("webhook timestamp out of range", zap.Time("timestamp", payload.Timestamp))
		metrics.ErrorTotal.WithLabelValues("webhook", "timestamp_out_of_range").Inc()
		WriteResponse(w, http.StatusBadRequest, WebhookResponse{
			Status:  "error",
			Message: "timestamp out of range",
		})
		return
	}

	if payload.Nonce != "" {
		storage := NewStorage(s.db, s.log)
	if err := storage.CheckAndSaveNonce(r.Context(), payload.UserID, payload.Nonce, payload.Timestamp); err != nil {
		// codeql[go/log-injection] user_id is sanitized via sanitize.LogString
		s.log.Warn("webhook nonce check failed", zap.Error(err), zap.String("user_id", sanitize.LogString(payload.UserID)))
			metrics.ErrorTotal.WithLabelValues("webhook", "nonce_reused").Inc()
			WriteResponse(w, http.StatusConflict, WebhookResponse{
				Status:  "error",
				Message: "duplicate webhook nonce",
			})
			return
		}
	}

	if err := payload.Validate(); err != nil {
		// codeql[go/log-injection] user_id is sanitized via sanitize.LogString
		s.log.Error("payload validation failed", zap.Error(err), zap.String("user_id", sanitize.LogString(payload.UserID)))
		WriteResponse(w, http.StatusBadRequest, WebhookResponse{
			Status:  "error",
			Message: fmt.Sprintf("validation failed: %v", err),
		})
		return
	}

	storage := NewStorage(s.db, s.log)
	if err := storage.SaveMetrics(r.Context(), payload); err != nil {
		// codeql[go/log-injection] user_id is sanitized via sanitize.LogString
		s.log.Error("failed to save metrics", zap.Error(err), zap.String("user_id", sanitize.LogString(payload.UserID)))
		metrics.ErrorTotal.WithLabelValues("webhook", "save_failed").Inc()
		WriteResponse(w, http.StatusInternalServerError, WebhookResponse{
			Status:  "error",
			Message: "failed to save metrics",
		})
		return
	}

	metrics.RequestsTotal.WithLabelValues("POST", "/api/v1/integrations/open-wearables/webhook", "200").Inc()

	WriteResponse(w, http.StatusOK, WebhookResponse{
		Status:  "success",
		Message: fmt.Sprintf("processed %d metrics", len(payload.Metrics)),
	})
}

func readBody(r *http.Request) ([]byte, error) {
	limited := http.MaxBytesReader(nil, r.Body, 1<<20)
	defer func() { _ = limited.Close() }()
	return io.ReadAll(limited)
}
