// Package devices aggregates wearable device providers and syncs data.
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
)

// aggregator manages wearable providers and syncs data.
type aggregator struct {
	db       *sql.DB
	log      *logger.Logger
	fitbit   *providers.FitbitProvider
	garmin   *providers.GarminProvider
	withings *providers.WithingsProvider
}

// newAggregator creates a new provider aggregator.
func newAggregator(db *sql.DB, log *logger.Logger, fitbit *providers.FitbitProvider, garmin *providers.GarminProvider, withings *providers.WithingsProvider) *aggregator {
	return &aggregator{db: db, log: log, fitbit: fitbit, garmin: garmin, withings: withings}
}

// healthHandler returns service health status.
func (a *aggregator) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// SAFETY: Static JSON health response, Content-Type is application/json.
	if _, err := w.Write([]byte(`{"status":"ok","service":"device-aggregator"}`)); err != nil {
		a.log.Warn("failed to write health response", zap.Error(err))
	}
}

// handleOAuthCallback handles the common OAuth callback flow for device providers.
func (a *aggregator) handleOAuthCallback(w http.ResponseWriter, r *http.Request, exchangeFunc func(ctx context.Context, code, state string) error, providerName string) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		http.Error(w, "Missing code or state", http.StatusBadRequest)
		return
	}

	if err := exchangeFunc(r.Context(), code, state); err != nil {
		a.log.Error("Failed to exchange "+providerName+" code", zap.Error(err))
		http.Error(w, "Ошибка авторизации", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "https://fittpulse.duckdns.org:30443/#devices", http.StatusFound)
}

// handleDisconnect handles the common disconnect flow for device providers.
func (a *aggregator) handleDisconnect(w http.ResponseWriter, r *http.Request, disconnectFunc func(ctx context.Context, userID string) error, providerName string) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := disconnectFunc(r.Context(), userID); err != nil {
		a.log.Error("Failed to disconnect "+providerName, zap.Error(err))
		http.Error(w, "Ошибка отключения", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	// SAFETY: Static JSON response, Content-Type is application/json.
	if _, err := w.Write([]byte(`{"status":"disconnected"}`)); err != nil {
		a.log.Warn("failed to write disconnect response", zap.Error(err))
	}
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

	parsed, parseErr := url.Parse(authURL)
	if parseErr != nil {
		a.log.Error("Invalid auth URL", zap.Error(parseErr))
		http.Error(w, "Invalid redirect target", http.StatusBadRequest)
		return
	}
	if parsed.Scheme != "https" {
		a.log.Warn("redirect scheme not allowed", zap.String("scheme", parsed.Scheme))
		http.Error(w, "Invalid redirect target", http.StatusBadRequest)
		return
	}
	if !strings.HasSuffix(parsed.Host, "fitbit.com") && !strings.HasSuffix(parsed.Host, "duckdns.org") {
		a.log.Warn("redirect host not allowed", zap.String("host", parsed.Host))
		http.Error(w, "Invalid redirect target", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "https://fittpulse.duckdns.org:30443/#devices/auth/fitbit", http.StatusFound)
}

// fitbitCallbackHandler handles the OAuth callback from Fitbit.
func (a *aggregator) fitbitCallbackHandler(w http.ResponseWriter, r *http.Request) {
	a.handleOAuthCallback(w, r, a.fitbit.ExchangeCode, "Fitbit")
}

// fitbitDisconnectHandler disconnects a Fitbit account for the user.
func (a *aggregator) fitbitDisconnectHandler(w http.ResponseWriter, r *http.Request) {
	a.handleDisconnect(w, r, a.fitbit.Disconnect, "Fitbit")
}

// handleAuthStart handles the common OAuth start flow for device providers.
func (a *aggregator) handleAuthStart(w http.ResponseWriter, r *http.Request, getAuthURL func(userID string) (string, error), providerName, redirectFragment string) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	authURL, err := getAuthURL(userID)
	if err != nil {
		a.log.Error("Failed to get "+providerName+" auth URL", zap.Error(err))
		http.Error(w, "Ошибка авторизации", http.StatusInternalServerError)
		return
	}

	parsed, parseErr := url.Parse(authURL)
	if parseErr != nil {
		a.log.Error("Invalid "+providerName+" auth URL", zap.Error(parseErr))
		http.Error(w, "Invalid redirect target", http.StatusBadRequest)
		return
	}
	if parsed.Scheme != "https" {
		a.log.Warn("redirect scheme not allowed", zap.String("scheme", parsed.Scheme))
		http.Error(w, "Invalid redirect target", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "https://fittpulse.duckdns.org:30443/#devices/auth/"+redirectFragment, http.StatusFound)
}

// garminAuthHandler starts the Garmin OAuth 1.0a flow.
func (a *aggregator) garminAuthHandler(w http.ResponseWriter, r *http.Request) {
	a.handleAuthStart(w, r, a.garmin.GetAuthURL, "Garmin", "garmin")
}

// garminCallbackHandler handles the OAuth callback from Garmin.
func (a *aggregator) garminCallbackHandler(w http.ResponseWriter, r *http.Request) {
	a.handleOAuthCallback(w, r, func(ctx context.Context, code, state string) error {
		oauthVerifier := r.URL.Query().Get("oauth_verifier")
		return a.garmin.ExchangeCode(ctx, code, oauthVerifier, state)
	}, "Garmin")
}

// garminDisconnectHandler disconnects a Garmin account for the user.
func (a *aggregator) garminDisconnectHandler(w http.ResponseWriter, r *http.Request) {
	a.handleDisconnect(w, r, a.garmin.Disconnect, "Garmin")
}

// withingsAuthHandler starts the Withings OAuth flow.
func (a *aggregator) withingsAuthHandler(w http.ResponseWriter, r *http.Request) {
	a.handleAuthStart(w, r, a.withings.GetAuthURL, "Withings", "withings")
}

// withingsCallbackHandler handles the OAuth callback from Withings.
func (a *aggregator) withingsCallbackHandler(w http.ResponseWriter, r *http.Request) {
	a.handleOAuthCallback(w, r, a.withings.ExchangeCode, "Withings")
}

// withingsDisconnectHandler disconnects a Withings account for the user.
func (a *aggregator) withingsDisconnectHandler(w http.ResponseWriter, r *http.Request) {
	a.handleDisconnect(w, r, a.withings.Disconnect, "Withings")
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
