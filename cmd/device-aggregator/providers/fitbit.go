// Package providers contains wearable device OAuth integrations.
package providers

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/config"
	"github.com/MAMUER/project/internal/crypto"
)

// FitbitProvider implements OAuth 2.0 flow for Fitbit API.
type FitbitProvider struct {
	clientID     string
	clientSecret string
	redirectURI  string
	db           *sql.DB
	log          *zap.Logger
	encryptor    *crypto.AESGCMEncryptor
}

// NewFitbitProvider returns a new Fitbit provider bound to the given database, logger and encryptor.
func NewFitbitProvider(db *sql.DB, log *zap.Logger, encryptor *crypto.AESGCMEncryptor) *FitbitProvider {
	return &FitbitProvider{
		clientID:     config.GetEnv("FITBIT_CLIENT_ID"),
		clientSecret: config.GetEnv("FITBIT_CLIENT_SECRET"),
		redirectURI:  config.GetEnv("FITBIT_REDIRECT_URI"),
		db:           db,
		log:          log,
		encryptor:    encryptor,
	}
}

// GetAuthURL returns a Fitbit OAuth authorization URL for the given user.
func (p *FitbitProvider) GetAuthURL(userID string) (string, error) {
	state := uuid.New().String()

	if _, err := p.db.ExecContext(context.Background(), `
		INSERT INTO oauth_states (state, user_id, provider, expires_at)
		VALUES ($1, $2, 'fitbit', NOW() + INTERVAL '10 minutes')
	`, state, userID); err != nil {
		return "", fmt.Errorf("insert oauth state: %w", err)
	}

	scopes := "activity heartrate sleep profile weight"
	return fmt.Sprintf(
		"https://www.fitbit.com/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		p.clientID,
		url.QueryEscape(p.redirectURI),
		url.QueryEscape(scopes),
		state,
	), nil
}

// ExchangeCode exchanges an authorization code for tokens and stores the account.
func (p *FitbitProvider) ExchangeCode(ctx context.Context, code, state string) error {
	var userID string
	if err := p.db.QueryRowContext(ctx, `
		SELECT user_id FROM oauth_states 
		WHERE state = $1 AND provider = 'fitbit' AND expires_at > NOW()
	`, state).Scan(&userID); err != nil {
		return fmt.Errorf("invalid or expired state: %w", err)
	}

	if _, err := p.db.ExecContext(ctx, `DELETE FROM oauth_states WHERE state = $1`, state); err != nil {
		p.log.Warn("failed to delete oauth state", zap.Error(err))
	}

	tokenResp, err := p.exchangeCodeForTokens(code)
	if err != nil {
		return fmt.Errorf("exchange code for tokens: %w", err)
	}

	profile, err := p.getProfile(tokenResp.AccessToken)
	if err != nil {
		return fmt.Errorf("get profile: %w", err)
	}

	encryptedRefresh, err := p.encryptRefreshToken(tokenResp.RefreshToken)
	if err != nil {
		return fmt.Errorf("encrypt refresh token: %w", err)
	}

	_, err = p.db.ExecContext(ctx, `
		INSERT INTO device_provider_accounts
		(user_id, provider, provider_user_id, access_token, refresh_token, token_expires_at, scopes, is_active)
		VALUES ($1, 'fitbit', $2, $3, $4, $5, $6, TRUE)
		ON CONFLICT (user_id, provider)
		DO UPDATE SET
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			token_expires_at = EXCLUDED.token_expires_at,
			is_active = TRUE,
			updated_at = NOW()
	`, userID, profile.User.EncodedID, tokenResp.AccessToken, encryptedRefresh,
		time.Now().Add(time.Duration(tokenResp.ExpiresIn)*time.Second),
		strings.Split(tokenResp.Scope, " "))
	if err != nil {
		return fmt.Errorf("upsert device provider account: %w", err)
	}

	return nil
}

func (p *FitbitProvider) exchangeCodeForTokens(code string) (*FitbitTokenResponse, error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", p.redirectURI)

	req, err := http.NewRequestWithContext(context.Background(), "POST", "https://api.fitbit.com/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create token request: %w", err)
	}
	req.SetBasicAuth(p.clientID, p.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do token request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fitbit API error: %s", string(body))
	}

	var tokenResp FitbitTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	return &tokenResp, nil
}

func (p *FitbitProvider) getProfile(accessToken string) (*FitbitProfile, error) {
	req, err := http.NewRequestWithContext(context.Background(), "GET", "https://api.fitbit.com/1/user/-/profile.json", nil)
	if err != nil {
		return nil, fmt.Errorf("create profile request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read profile response: %w", err)
	}
	var profile FitbitProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("decode profile response: %w", err)
	}
	return &profile, nil
}

// Disconnect marks a Fitbit connection as inactive.
func (p *FitbitProvider) Disconnect(ctx context.Context, userID string) error {
	_, err := p.db.ExecContext(ctx, `
		UPDATE device_provider_accounts 
		SET is_active = FALSE, updated_at = NOW()
		WHERE user_id = $1 AND provider = 'fitbit'
	`, userID)
	return fmt.Errorf("disconnect fitbit account: %w", err)
}

// ListProviders returns connected Fitbit accounts for the user.
func (p *FitbitProvider) ListProviders(ctx context.Context, userID string) (map[string]interface{}, error) {
	return listAccountProviders(ctx, p.db, userID, "fitbit")
}

// listAccountProviders queries device_provider_accounts for the given provider
// and returns a summary map.
func listAccountProviders(ctx context.Context, db *sql.DB, userID, providerName string) (map[string]interface{}, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT provider, provider_user_id, is_active, last_sync_at, created_at
		FROM device_provider_accounts
		WHERE user_id = $1 AND provider = $2
		ORDER BY created_at DESC
	`, userID, providerName)
	if err != nil {
		return nil, fmt.Errorf("query provider accounts: %w", err)
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

func (p *FitbitProvider) encryptRefreshToken(token string) (string, error) {
	if p.encryptor == nil {
		return "", errors.New("encryptor not initialized")
	}
	ciphertext, err := p.encryptor.Encrypt([]byte(token))
	if err != nil {
		return "", fmt.Errorf("encrypt refresh token: %w", err)
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// FitbitTokenResponse represents Fitbit token response.
type FitbitTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

// FitbitProfile represents Fitbit user profile.
type FitbitProfile struct {
	User struct {
		EncodedID string `json:"encodedId"`
	} `json:"user"`
}
