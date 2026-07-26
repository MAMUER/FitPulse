package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"

	"github.com/MAMUER/project/cmd/device-aggregator/providers"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/metrics"
)

type aggregator struct {
	db       *sql.DB
	log      *logger.Logger
	fitbit   *providers.FitbitProvider
	garmin   *providers.GarminProvider
	withings *providers.WithingsProvider
}

func newAggregator(db *sql.DB, log *logger.Logger, fitbit *providers.FitbitProvider, garmin *providers.GarminProvider, withings *providers.WithingsProvider) *aggregator {
	return &aggregator{db: db, log: log, fitbit: fitbit, garmin: garmin, withings: withings}
}

func (a *aggregator) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"healthy","service":"device-aggregator"}`)); err != nil {
		a.log.Warn("failed to write health response", zap.Error(err))
	}
}

func (a *aggregator) handleOAuthCallback(w http.ResponseWriter, r *http.Request, exchangeFunc func(ctx context.Context, code, state string) error, providerName string) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		metrics.ErrorTotal.WithLabelValues("device-aggregator", "missing_code_or_state").Inc()
		http.Error(w, "Missing code or state", http.StatusBadRequest)
		return
	}

	if err := exchangeFunc(r.Context(), code, state); err != nil {
		metrics.ErrorTotal.WithLabelValues("device-aggregator", "oauth_exchange_error").Inc()
		a.log.Error("Failed to exchange "+providerName+" code", zap.Error(err))
		http.Error(w, "Ошибка авторизации", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "https://fittpulse.duckdns.org:30443/#devices", http.StatusFound)
}

func (a *aggregator) handleDisconnect(w http.ResponseWriter, r *http.Request, disconnectFunc func(ctx context.Context, userID string) error, providerName string) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := disconnectFunc(r.Context(), userID); err != nil {
		metrics.ErrorTotal.WithLabelValues("device-aggregator", "disconnect_error").Inc()
		a.log.Error("Failed to disconnect "+providerName, zap.Error(err))
		http.Error(w, "Ошибка отключения", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "disconnected"}); err != nil {
		a.log.Warn("failed to write disconnect response", zap.Error(err))
	}
}

func (a *aggregator) handleAuthStart(w http.ResponseWriter, r *http.Request, getAuthURL func(userID string) (string, error), providerName, redirectFragment string) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	authURL, err := getAuthURL(userID)
	if err != nil {
		metrics.ErrorTotal.WithLabelValues("device-aggregator", "auth_url_error").Inc()
		a.log.Error("Failed to get "+providerName+" auth URL", zap.Error(err))
		http.Error(w, "Ошибка авторизации", http.StatusInternalServerError)
		return
	}

	parsed, parseErr := url.Parse(authURL)
	if parseErr != nil {
		metrics.ErrorTotal.WithLabelValues("device-aggregator", "invalid_auth_url").Inc()
		a.log.Error("Invalid "+providerName+" auth URL", zap.Error(parseErr))
		http.Error(w, "Invalid redirect target", http.StatusBadRequest)
		return
	}
	if parsed.Scheme != "https" {
		metrics.ErrorTotal.WithLabelValues("device-aggregator", "invalid_redirect_scheme").Inc()
		a.log.Warn("redirect scheme not allowed", zap.String("scheme", parsed.Scheme))
		http.Error(w, "Invalid redirect target", http.StatusBadRequest)
		return
	}
	if !a.isValidRedirectHost(parsed.Host) {
		metrics.ErrorTotal.WithLabelValues("device-aggregator", "invalid_redirect_host").Inc()
		a.log.Warn("redirect host not allowed", zap.String("host", parsed.Host))
		http.Error(w, "Invalid redirect target", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "https://fittpulse.duckdns.org:30443/#devices/auth/"+redirectFragment, http.StatusFound)
}

func (a *aggregator) fitbitAuthHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	authURL, err := a.fitbit.GetAuthURL(userID)
	if err != nil {
		metrics.ErrorTotal.WithLabelValues("device-aggregator", "auth_url_error").Inc()
		a.log.Error("Failed to get auth URL", zap.Error(err))
		http.Error(w, "Ошибка авторизации", http.StatusInternalServerError)
		return
	}

	parsed, parseErr := url.Parse(authURL)
	if parseErr != nil {
		metrics.ErrorTotal.WithLabelValues("device-aggregator", "invalid_auth_url").Inc()
		a.log.Error("Invalid auth URL", zap.Error(parseErr))
		http.Error(w, "Invalid redirect target", http.StatusBadRequest)
		return
	}
	if parsed.Scheme != "https" {
		metrics.ErrorTotal.WithLabelValues("device-aggregator", "invalid_redirect_scheme").Inc()
		a.log.Warn("redirect scheme not allowed", zap.String("scheme", parsed.Scheme))
		http.Error(w, "Invalid redirect target", http.StatusBadRequest)
		return
	}
	if !a.isValidRedirectHost(parsed.Host) {
		metrics.ErrorTotal.WithLabelValues("device-aggregator", "invalid_redirect_host").Inc()
		a.log.Warn("redirect host not allowed", zap.String("host", parsed.Host))
		http.Error(w, "Invalid redirect target", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "https://fittpulse.duckdns.org:30443/#devices/auth/fitbit", http.StatusFound)
}

func (a *aggregator) fitbitCallbackHandler(w http.ResponseWriter, r *http.Request) {
	a.handleOAuthCallback(w, r, a.fitbit.ExchangeCode, "Fitbit")
}

func (a *aggregator) fitbitDisconnectHandler(w http.ResponseWriter, r *http.Request) {
	a.handleDisconnect(w, r, a.fitbit.Disconnect, "Fitbit")
}

func (a *aggregator) garminAuthHandler(w http.ResponseWriter, r *http.Request) {
	a.handleAuthStart(w, r, a.garmin.GetAuthURL, "Garmin", "garmin")
}

func (a *aggregator) garminCallbackHandler(w http.ResponseWriter, r *http.Request) {
	a.handleOAuthCallback(w, r, func(ctx context.Context, code, state string) error {
		oauthVerifier := r.URL.Query().Get("oauth_verifier")
		return a.garmin.ExchangeCode(ctx, code, oauthVerifier, state)
	}, "Garmin")
}

func (a *aggregator) garminDisconnectHandler(w http.ResponseWriter, r *http.Request) {
	a.handleDisconnect(w, r, a.garmin.Disconnect, "Garmin")
}

func (a *aggregator) withingsAuthHandler(w http.ResponseWriter, r *http.Request) {
	a.handleAuthStart(w, r, a.withings.GetAuthURL, "Withings", "withings")
}

func (a *aggregator) withingsCallbackHandler(w http.ResponseWriter, r *http.Request) {
	a.handleOAuthCallback(w, r, a.withings.ExchangeCode, "Withings")
}

func (a *aggregator) withingsDisconnectHandler(w http.ResponseWriter, r *http.Request) {
	a.handleDisconnect(w, r, a.withings.Disconnect, "Withings")
}

func (a *aggregator) listProvidersHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	providersData, err := a.fitbit.ListProviders(r.Context(), userID)
	if err != nil {
		metrics.ErrorTotal.WithLabelValues("device-aggregator", "list_providers_error").Inc()
		a.log.Error("Failed to list providers", zap.Error(err))
		http.Error(w, "Ошибка получения данных", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(providersData); err != nil {
		a.log.Warn("failed to write providers response", zap.Error(err))
	}
}

func (a *aggregator) isValidRedirectHost(host string) bool {
	return strings.HasSuffix(host, "fitbit.com") ||
		strings.HasSuffix(host, "withings.com") ||
		strings.HasSuffix(host, "withings.net") ||
		strings.HasSuffix(host, "duckdns.org")
}
