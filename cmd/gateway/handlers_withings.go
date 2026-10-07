package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/MAMUER/project/internal/middleware"
	"go.uber.org/zap"
)

const (
	errWithingsNotConfigured = "Withings integration is not configured"
	withingsOAuthStateCookie = "withings_oauth_state"
)

// @Summary      Initiate Withings OAuth login
// @Description  Redirects user to Withings consent screen
// @Tags         Integrations
// @Produce      html
// @Success      307  {string}  string
// @Failure      501  {object}  map[string]interface{}
// @Router       /api/v1/integrations/withings/auth [get]
func (g *gateway) withingsLoginHandler(w http.ResponseWriter, r *http.Request) {
	if g.withingsClient == nil {
		g.log.Error(errWithingsNotConfigured)
		http.Error(w, errWithingsNotConfigured, http.StatusNotImplemented)
		return
	}

	state, err := generateOAuthState()
	if err != nil {
		g.log.Error("Failed to generate Withings OAuth state", zap.Error(err))
		http.Error(w, "failed to generate oauth state", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     withingsOAuthStateCookie,
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		Expires:  time.Now().Add(10 * time.Minute),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	redirectURL, err := g.withingsClient.BuildAuthorizeURL(state)
	if err != nil {
		g.log.Error("Failed to build Withings authorize URL", zap.Error(err))
		http.Error(w, "failed to build authorize url", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// @Summary      Withings OAuth callback
// @Description  Handles Withings OAuth callback, exchanges authorization code for tokens and stores device
// @Tags         Integrations
// @Accept       json
// @Produce      json
// @Param        state  query  string  false  "OAuth state parameter"
// @Param        code   query  string  false  "Authorization code from Withings"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/v1/integrations/withings/callback [get]
func (g *gateway) withingsCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if g.withingsClient == nil {
		g.log.Error(errWithingsNotConfigured)
		http.Error(w, errWithingsNotConfigured, http.StatusNotImplemented)
		return
	}

	state := r.URL.Query().Get("state")
	cookie, err := r.Cookie(withingsOAuthStateCookie)
	if err != nil || state == "" || cookie == nil || cookie.Value == "" || cookie.Value != state {
		g.log.Error("Invalid Withings OAuth state")
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     withingsOAuthStateCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		g.log.Error("Missing Withings authorization code")
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tokenResp, err := g.withingsClient.ExchangeCode(r.Context(), code)
	if err != nil {
		g.log.Error("Failed to exchange Withings code", zap.Error(err))
		http.Error(w, "failed to exchange authorization code", http.StatusBadRequest)
		return
	}

	deviceType := "withings"
	deviceName := "Withings"

	_, err = g.deviceClient.CreateOrUpdateDevice(r.Context(), userID, deviceType, deviceName, fmt.Sprintf(`{"access_token":"%s","refresh_token":"%s","expires_in":%d}`, tokenResp.Body.AccessToken, tokenResp.Body.RefreshToken, tokenResp.Body.ExpiresIn))
	if err != nil {
		g.log.Error("Failed to store Withings device", zap.Error(err))
		http.Error(w, "failed to store device", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"source":  "withings",
		"user_id": userID,
	})
}

// @Summary      Sync Withings data
// @Description  Pulls latest data from Withings and stores it
// @Tags         Integrations
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/v1/integrations/withings/sync [post]
func (g *gateway) withingsSyncHandler(w http.ResponseWriter, r *http.Request) {
	if g.withingsClient == nil {
		g.log.Error(errWithingsNotConfigured)
		http.Error(w, errWithingsNotConfigured, http.StatusNotImplemented)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	device, err := g.deviceClient.GetByType(r.Context(), userID, "withings")
	if err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			http.Error(w, "Withings device not connected", http.StatusNotFound)
		} else {
			g.log.Error("Failed to load Withings device", zap.Error(err))
			http.Error(w, "failed to load device", http.StatusInternalServerError)
		}
		return
	}

	if device.Token == "" {
		http.Error(w, "Withings device token is empty", http.StatusBadRequest)
		return
	}

	token, err := g.parseWithingsToken(device.Token)
	if err != nil {
		http.Error(w, "invalid device token", http.StatusBadRequest)
		return
	}

	lastSync := lastSyncTime(device.LastSync)
	synced := g.syncWithingsMetrics(r.Context(), token.AccessToken, lastSync)

	if updateErr := g.deviceClient.UpdateLastSync(r.Context(), device.ID); updateErr != nil {
		g.log.Warn("Failed to update Withings device last_sync", zap.Error(updateErr))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"source":  "withings",
		"user_id": userID,
		"synced":  synced,
	})
}

func (g *gateway) parseWithingsToken(token string) (struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}, error) {
	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal([]byte(token), &parsed); err != nil {
		g.log.Error("Failed to parse Withings device token", zap.Error(err))
		return parsed, err
	}
	return parsed, nil
}

func (g *gateway) syncWithingsMetrics(ctx context.Context, accessToken string, lastSync time.Time) int {
	synced := 0

	measures, err := g.withingsClient.GetMeasures(ctx, accessToken, lastSync)
	if err == nil {
		synced += len(measures.Body.MeasureGroups)
	} else {
		g.log.Warn("Withings measures sync failed", zap.Error(err))
	}

	sleep, err := g.withingsClient.GetSleep(ctx, accessToken, lastSync)
	if err == nil {
		synced += len(sleep.Body.MeasureGroups)
	} else {
		g.log.Warn("Withings sleep sync failed", zap.Error(err))
	}

	activity, err := g.withingsClient.GetActivity(ctx, accessToken, lastSync)
	if err == nil {
		synced += len(activity.Body.MeasureGroups)
	} else {
		g.log.Warn("Withings activity sync failed", zap.Error(err))
	}

	return synced
}

func lastSyncTime(lastSync interface{}) time.Time {
	var ts time.Time
	switch v := lastSync.(type) {
	case time.Time:
		ts = v
	case string:
		if parsed, err := time.Parse(time.RFC3339, v); err == nil {
			ts = parsed
		}
	}
	return ts
}
