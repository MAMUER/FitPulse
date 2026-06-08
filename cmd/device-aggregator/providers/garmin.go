package providers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// GarminProvider implements OAuth flow for Garmin Health API
// Note: Garmin uses OAuth 1.0a which is more complex than OAuth 2.0
type GarminProvider struct {
	consumerKey    string
	consumerSecret string
	callbackURL    string
	db             *sql.DB
	log            *zap.Logger
}

// NewGarminProvider returns a new Garmin provider.
func NewGarminProvider(db *sql.DB, log *zap.Logger) *GarminProvider {
	return &GarminProvider{
		consumerKey:    os.Getenv("GARMIN_CONSUMER_KEY"),
		consumerSecret: os.Getenv("GARMIN_CONSUMER_SECRET"),
		callbackURL:    os.Getenv("GARMIN_CALLBACK_URL"),
		db:             db,
		log:            log,
	}
}

// GetAuthURL returns URL to start Garmin OAuth flow
// Note: Garmin Health API requires OAuth 1.0a
func (p *GarminProvider) GetAuthURL(userID string) (string, error) {
	state := uuid.New().String()

	// Save state to database
	if _, err := p.db.ExecContext(context.Background(), `
		INSERT INTO oauth_states (state, user_id, provider, expires_at)
		VALUES ($1, $2, 'garmin', NOW() + INTERVAL '10 minutes')
	`, state, userID); err != nil {
		return "", err
	}

	// Garmin Health API OAuth 1.0a flow
	// This is a simplified version - full implementation requires OAuth 1.0a signature
	authURL := fmt.Sprintf(
		"https://connectapi.garmin.com/oauth-service/oauth/authorize?oauth_token=%s&oauth_callback=%s",
		"PLACEHOLDER_TOKEN", // Will be replaced with actual request token
		url.QueryEscape(p.callbackURL),
	)

	return authURL, nil
}

// ExchangeCode exchanges OAuth token for access token
func (p *GarminProvider) ExchangeCode(ctx context.Context, oauthToken, oauthVerifier, state string) error {
	// Verify state
	var userID string
	err := p.db.QueryRowContext(ctx, `
		SELECT user_id FROM oauth_states 
		WHERE state = $1 AND provider = 'garmin' AND expires_at > NOW()
	`, state).Scan(&userID)
	if err != nil {
		return fmt.Errorf("invalid or expired state: %w", err)
	}

	// Delete used state
	if _, delErr := p.db.ExecContext(ctx, `DELETE FROM oauth_states WHERE state = $1`, state); delErr != nil {
		p.log.Warn("failed to delete oauth state", zap.Error(delErr))
	}

	// Exchange for access token (OAuth 1.0a)
	tokenResp, err := p.exchangeForAccessToken(oauthToken, oauthVerifier)
	if err != nil {
		return err
	}

	// Get Garmin user info
	profile, err := p.getUserProfile(tokenResp.AccessToken, tokenResp.AccessTokenSecret)
	if err != nil {
		return err
	}

	// Save to database
	_, err = p.db.ExecContext(ctx, `
		INSERT INTO device_provider_accounts 
		(user_id, provider, provider_user_id, access_token, refresh_token, token_expires_at, scopes, is_active)
		VALUES ($1, 'garmin', $2, $3, $4, NULL, ARRAY['activity', 'sleep', 'heart_rate'], TRUE)
		ON CONFLICT (user_id, provider) 
		DO UPDATE SET 
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			is_active = TRUE,
			updated_at = NOW()
	`, userID, profile.UserID, tokenResp.AccessToken, tokenResp.AccessTokenSecret)

	return err
}

func (p *GarminProvider) exchangeForAccessToken(oauthToken, oauthVerifier string) (*GarminTokenResponse, error) {
	// OAuth 1.0a token exchange - requires signature
	// This is a placeholder - full implementation needed
	data := url.Values{}
	data.Set("oauth_token", oauthToken)
	data.Set("oauth_verifier", oauthVerifier)

	req, err := http.NewRequestWithContext(context.Background(), "POST", "https://connectapi.garmin.com/oauth-service/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create token request: %w", err)
	}
	req.SetBasicAuth(p.consumerKey, p.consumerSecret)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}
	var tokenResp GarminTokenResponse
	// Garmin returns URL-encoded response, not JSON
	// Parse: oauth_token=xxx&oauth_token_secret=yyy
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	tokenResp.AccessToken = values.Get("oauth_token")
	tokenResp.AccessTokenSecret = values.Get("oauth_token_secret")

	return &tokenResp, nil
}

func (p *GarminProvider) getUserProfile(accessToken, accessTokenSecret string) (*GarminProfile, error) {
	_ = accessTokenSecret // OAuth 1.0a secret used for request signing (placeholder implementation)
	req, err := http.NewRequestWithContext(context.Background(), "GET", "https://connectapi.garmin.com/userprofile-service/userprofile/user-profile", nil)
	if err != nil {
		return nil, fmt.Errorf("create profile request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read profile response: %w", err)
	}
	var profile GarminProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("decode profile response: %w", err)
	}
	return &profile, nil
}

// Disconnect deactivates Garmin connection
func (p *GarminProvider) Disconnect(ctx context.Context, userID string) error {
	_, err := p.db.ExecContext(ctx, `
		UPDATE device_provider_accounts 
		SET is_active = FALSE, updated_at = NOW()
		WHERE user_id = $1 AND provider = 'garmin'
	`, userID)
	return err
}

// ListProviders returns list of connected Garmin accounts.
func (p *GarminProvider) ListProviders(ctx context.Context, userID string) (map[string]interface{}, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT provider, provider_user_id, is_active, last_sync_at, created_at
		FROM device_provider_accounts
		WHERE user_id = $1 AND provider = 'garmin'
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	type ProviderInfo struct {
		Provider       string     `json:"provider"`
		ProviderUserID string     `json:"provider_user_id"`
		IsActive       bool       `json:"is_active"`
		LastSyncAt     *time.Time `json:"last_sync_at,omitempty"`
		CreatedAt      time.Time  `json:"created_at"`
	}

	var providers []ProviderInfo
	for rows.Next() {
		var p ProviderInfo
		if err := rows.Scan(&p.Provider, &p.ProviderUserID, &p.IsActive, &p.LastSyncAt, &p.CreatedAt); err != nil {
			continue
		}
		providers = append(providers, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan providers: %w", err)
	}

	return map[string]interface{}{
		"providers": providers,
		"total":     len(providers),
	}, nil
}

// GarminTokenResponse represents Garmin token response.
type GarminTokenResponse struct {
	AccessToken       string
	AccessTokenSecret string
}

// GarminProfile represents Garmin user profile.
type GarminProfile struct {
	UserID string `json:"id"`
}
