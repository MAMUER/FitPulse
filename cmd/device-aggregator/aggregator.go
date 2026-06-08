// Package devices aggregates wearable device providers and syncs data.
package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/MAMUER/project/cmd/device-aggregator/providers"
	"github.com/MAMUER/project/internal/logger"
	"go.uber.org/zap"
)

// aggregator manages wearable providers and syncs data.
type aggregator struct {
	db     *sql.DB
	log    *logger.Logger
	fitbit *providers.FitbitProvider
}

// newAggregator creates a new provider aggregator.
func newAggregator(db *sql.DB, log *logger.Logger, fitbit *providers.FitbitProvider) *aggregator {
	return &aggregator{db: db, log: log, fitbit: fitbit}
}

// healthHandler returns service health status.
func (a *aggregator) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"ok","service":"device-aggregator"}`)); err != nil {
		a.log.Warn("failed to write health response", zap.Error(err))
	}
}

func isAllowedRedirect(log *logger.Logger, target string) bool {
	parsed, err := url.Parse(target)
	if err != nil {
		log.Warn("invalid redirect target", zap.String("target", target), zap.Error(err))
		return false
	}
	if parsed.Scheme != "" && parsed.Scheme != "https" {
		log.Warn("redirect scheme not allowed", zap.String("scheme", parsed.Scheme))
		return false
	}
	if parsed.Host != "" && !strings.HasSuffix(parsed.Host, "duckdns.org") {
		log.Warn("redirect host not allowed", zap.String("host", parsed.Host))
		return false
	}
	return true
}

// fitbitAuthHandler starts the Fitbit OAuth flow.
func (a *aggregator) fitbitAuthHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	authURL, err := a.fitbit.GetAuthURL(userID)
	if err != nil {
		a.log.Error("Failed to get auth URL", zap.Error(err))
		http.Error(w, "Ошибка авторизации", http.StatusInternalServerError)
		return
	}

	if !isAllowedRedirect(a.log, authURL) {
		http.Error(w, "Invalid redirect target", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, authURL, http.StatusFound)
}

// fitbitCallbackHandler handles the OAuth callback from Fitbit.
func (a *aggregator) fitbitCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		http.Error(w, "Missing code or state", http.StatusBadRequest)
		return
	}

	if err := a.fitbit.ExchangeCode(r.Context(), code, state); err != nil {
		a.log.Error("Failed to exchange code", zap.Error(err))
		http.Error(w, "Ошибка авторизации", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "https://fittpulse.duckdns.org:30443/#devices", http.StatusFound)
}

// fitbitDisconnectHandler disconnects a Fitbit account for the user.
func (a *aggregator) fitbitDisconnectHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := a.fitbit.Disconnect(r.Context(), userID); err != nil {
		a.log.Error("Failed to disconnect", zap.Error(err))
		http.Error(w, "Ошибка отключения", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"disconnected"}`)); err != nil {
		a.log.Warn("failed to write disconnect response", zap.Error(err))
	}
}

// listProvidersHandler returns the list of connected providers for the user.
func (a *aggregator) listProvidersHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	providersData, err := a.fitbit.ListProviders(r.Context(), userID)
	if err != nil {
		a.log.Error("Failed to list providers", zap.Error(err))
		http.Error(w, "Ошибка получения данных", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(providersData); err != nil {
		a.log.Warn("failed to write providers response", zap.Error(err))
	}
}
