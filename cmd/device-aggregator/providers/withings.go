package providers

import (
	"context"
	"database/sql"
	"encoding/base64"
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

	"github.com/MAMUER/project/internal/crypto"
)

// WithingsProvider implements OAuth 2.0 flow for Withings API
type WithingsProvider struct {
	clientID     string
	clientSecret string
	callbackURL  string
	db           *sql.DB
	log          *zap.Logger
	encryptor    *crypto.AESGCMEncryptor
}

// NewWithingsProvider returns a new Withings provider.
func NewWithingsProvider(db *sql.DB, log *zap.Logger, encryptor *crypto.AESGCMEncryptor) *WithingsProvider {
	return &WithingsProvider{
		clientID:     os.Getenv("WITHINGS_CLIENT_ID"),
		clientSecret: os.Getenv("WITHINGS_CLIENT_SECRET"),
		callbackURL:  os.Getenv("WITHINGS_CALLBACK_URL"),
		db:           db,
		log:          log,
		encryptor:    encryptor,
	}
}

// GetAuthURL returns a Withings OAuth authorization URL for the given user.
func (p *WithingsProvider) GetAuthURL(userID string) (string, error) {
	state := uuid.New().String()

	if _, err := p.db.ExecContext(context.Background(), `
		INSERT INTO oauth_states (state, user_id, provider, expires_at)
		VALUES ($1, $2, 'withings', NOW() + INTERVAL '10 minutes')
	`, state, userID); err != nil {
		return "", fmt.Errorf("save oauth state: %w", err)
	}

	authURL := fmt.Sprintf(
		"https://account.withings.com/oauth2_user/authorize2?client_id=%s&redirect_uri=%s&scope=user.metrics&state=%s&mode=0",
		p.clientID,
		url.QueryEscape(p.callbackURL),
		state,
	)

	return authURL, nil
}

// ExchangeCode exchanges an authorization code for tokens and stores the account.
func (p *WithingsProvider) ExchangeCode(ctx context.Context, code, state string) error {
	var userID string
	err := p.db.QueryRowContext(ctx, `
		SELECT user_id FROM oauth_states 
		WHERE state = $1 AND provider = 'withings' AND expires_at > NOW()
	`, state).Scan(&userID)
	if err != nil {
		return fmt.Errorf("invalid or expired state: %w", err)
	}

	if _, err := p.db.ExecContext(ctx, `DELETE FROM oauth_states WHERE state = $1`, state); err != nil {
		p.log.Warn("failed to delete oauth state", zap.Error(err))
	}

	tokenResp, err := p.exchangeForAccessToken(code)
	if err != nil {
		return err
	}

	profile, err := p.getUserProfile(tokenResp.AccessToken)
	if err != nil {
		return fmt.Errorf("get user profile: %w", err)
	}

	encryptedRefresh, err := p.encryptRefreshToken(tokenResp.RefreshToken)
	if err != nil {
		return fmt.Errorf("encrypt refresh token: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	_, err = p.db.ExecContext(ctx, `
		INSERT INTO device_provider_accounts 
		(user_id, provider, provider_user_id, access_token, refresh_token, token_expires_at, scopes, is_active)
		VALUES ($1, 'withings', $2, $3, $4, $5, ARRAY['user.metrics'], TRUE)
		ON CONFLICT (user_id, provider) 
		DO UPDATE SET 
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			token_expires_at = EXCLUDED.token_expires_at,
			is_active = TRUE,
			updated_at = NOW()
	`, userID, profile.UserID, tokenResp.AccessToken, encryptedRefresh, expiresAt)

	if err != nil {
		return fmt.Errorf("upsert withings account: %w", err)
	}
	return nil
}

func (p *WithingsProvider) exchangeForAccessToken(code string) (*WithingsTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", p.callbackURL)

	req, err := http.NewRequestWithContext(context.Background(), "POST",
		"https://account.withings.com/oauth2/token",
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}

	var tokenResp WithingsTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	return &tokenResp, nil
}

func (p *WithingsProvider) getUserProfile(accessToken string) (*WithingsProfile, error) {
	req, err := http.NewRequestWithContext(context.Background(), "GET",
		"https://wbsapi.withings.net/v2/measure?action=getdevice&access_token="+accessToken, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute profile request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read profile response: %w", err)
	}

	var withingsResp struct {
		Body struct {
			Devices []struct {
				DeviceID string `json:"deviceid"`
				Model    string `json:"model"`
				Type     string `json:"type"`
			} `json:"devices"`
		} `json:"body"`
		Status int `json:"status"`
	}

	if err := json.Unmarshal(body, &withingsResp); err != nil {
		return nil, fmt.Errorf("decode profile response: %w", err)
	}

	if withingsResp.Status != 0 {
		return nil, fmt.Errorf("withings API error: %d", withingsResp.Status)
	}

	userID := "withings-" + uuid.New().String()
	if len(withingsResp.Body.Devices) > 0 {
		userID = withingsResp.Body.Devices[0].DeviceID
	}

	return &WithingsProfile{UserID: userID}, nil
}

// Disconnect deactivates a Withings connection for the user.
func (p *WithingsProvider) Disconnect(ctx context.Context, userID string) error {
	_, err := p.db.ExecContext(ctx, `
		UPDATE device_provider_accounts 
		SET is_active = FALSE, updated_at = NOW()
		WHERE user_id = $1 AND provider = 'withings'
	`, userID)
	if err != nil {
		return fmt.Errorf("disconnect withings: %w", err)
	}
	return nil
}

// ListProviders returns list of connected Withings accounts.
func (p *WithingsProvider) ListProviders(ctx context.Context, userID string) (map[string]interface{}, error) {
	return listAccountProviders(ctx, p.db, userID, "withings")
}

func (p *WithingsProvider) encryptRefreshToken(token string) (string, error) {
	if p.encryptor == nil {
		return "", fmt.Errorf("encryptor not initialized")
	}
	ciphertext, err := p.encryptor.Encrypt([]byte(token))
	if err != nil {
		return "", fmt.Errorf("encrypt refresh token: %w", err)
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// WithingsTokenResponse represents Withings token response.
type WithingsTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken  string `json:"refresh_token"`
	ExpiresIn     int    `json:"expires_in"`
	TokenType     string `json:"token_type"`
	Scope         string `json:"scope"`
}

// WithingsProfile represents Withings user profile.
type WithingsProfile struct {
	UserID string `json:"userid"`
}
