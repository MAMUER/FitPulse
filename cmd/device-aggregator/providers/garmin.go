package providers

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" // #nosec G505 - required by OAuth 1.0a HMAC-SHA1
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/crypto"
)

// GarminProvider implements OAuth 1.0a flow for Garmin Health API
type GarminProvider struct {
	consumerKey    string
	consumerSecret string
	callbackURL    string
	db             *sql.DB
	log            *zap.Logger
	encryptor      *crypto.AESGCMEncryptor
	requestToken   string
	requestSecret  string
}

// NewGarminProvider returns a new Garmin provider.
func NewGarminProvider(db *sql.DB, log *zap.Logger, encryptor *crypto.AESGCMEncryptor) *GarminProvider {
	return &GarminProvider{
		consumerKey:    os.Getenv("GARMIN_CONSUMER_KEY"),
		consumerSecret: os.Getenv("GARMIN_CONSUMER_SECRET"),
		callbackURL:    os.Getenv("GARMIN_CALLBACK_URL"),
		db:             db,
		log:            log,
		encryptor:      encryptor,
	}
}

func (p *GarminProvider) GetAuthURL(userID string) (string, error) {
	state := uuid.New().String()

	if _, err := p.db.ExecContext(context.Background(), `
		INSERT INTO oauth_states (state, user_id, provider, expires_at)
		VALUES ($1, $2, 'garmin', NOW() + INTERVAL '10 minutes')
	`, state, userID); err != nil {
		return "", fmt.Errorf("save oauth state: %w", err)
	}

	reqToken, reqSecret, err := p.getRequestToken()
	if err != nil {
		return "", fmt.Errorf("get request token: %w", err)
	}
	p.requestToken = reqToken
	p.requestSecret = reqSecret

	authURL := "https://connectapi.garmin.com/oauth-service/oauth/authorize?oauth_token=" + url.QueryEscape(reqToken) + "&oauth_callback=" + url.QueryEscape(p.callbackURL)

	return authURL, nil
}

func (p *GarminProvider) getRequestToken() (string, string, error) {
	oauthParams := url.Values{}
	oauthParams.Set("oauth_consumer_key", p.consumerKey)
	oauthParams.Set("oauth_signature_method", "HMAC-SHA1")
	oauthParams.Set("oauth_timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	oauthParams.Set("oauth_nonce", uuid.New().String())
	oauthParams.Set("oauth_version", "1.0")
	oauthParams.Set("oauth_callback", p.callbackURL)

	signature := p.sign("POST", "https://connectapi.garmin.com/oauth-service/oauth/request_token", oauthParams)
	oauthParams.Set("oauth_signature", signature)

	req, err := http.NewRequestWithContext(context.Background(), "POST",
		"https://connectapi.garmin.com/oauth-service/oauth/request_token",
		strings.NewReader(oauthParams.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "OAuth "+oauthParams.Encode())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("request token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("read request token response: %w", err)
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		return "", "", fmt.Errorf("parse request token response: %w", err)
	}

	return values.Get("oauth_token"), values.Get("oauth_token_secret"), nil
}

func (p *GarminProvider) ExchangeCode(ctx context.Context, oauthToken, oauthVerifier, state string) error {
	var userID string
	err := p.db.QueryRowContext(ctx, `
		SELECT user_id FROM oauth_states 
		WHERE state = $1 AND provider = 'garmin' AND expires_at > NOW()
	`, state).Scan(&userID)
	if err != nil {
		return fmt.Errorf("invalid or expired state: %w", err)
	}

	if _, delErr := p.db.ExecContext(ctx, `DELETE FROM oauth_states WHERE state = $1`, state); delErr != nil {
		p.log.Warn("failed to delete oauth state", zap.Error(delErr))
	}

	tokenResp, err := p.exchangeForAccessToken(oauthToken, oauthVerifier)
	if err != nil {
		return err
	}

	profile, err := p.getUserProfile(tokenResp.AccessToken, tokenResp.AccessTokenSecret)
	if err != nil {
		return fmt.Errorf("get user profile: %w", err)
	}

	encryptedRefresh, err := p.encryptRefreshToken(tokenResp.AccessTokenSecret)
	if err != nil {
		return fmt.Errorf("encrypt refresh token: %w", err)
	}

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
	`, userID, profile.UserID, tokenResp.AccessToken, encryptedRefresh)

	if err != nil {
		return fmt.Errorf("upsert garmin account: %w", err)
	}
	return nil
}

func (p *GarminProvider) exchangeForAccessToken(oauthToken, oauthVerifier string) (*GarminTokenResponse, error) {
	oauthParams := url.Values{}
	oauthParams.Set("oauth_consumer_key", p.consumerKey)
	oauthParams.Set("oauth_signature_method", "HMAC-SHA1")
	oauthParams.Set("oauth_timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	oauthParams.Set("oauth_nonce", uuid.New().String())
	oauthParams.Set("oauth_version", "1.0")
	oauthParams.Set("oauth_token", oauthToken)
	oauthParams.Set("oauth_verifier", oauthVerifier)

	signingKey := url.QueryEscape(p.consumerSecret) + "&" + url.QueryEscape(p.requestSecret)
	signature := p.signWithKey("POST", "https://connectapi.garmin.com/oauth-service/oauth/access_token", oauthParams, signingKey)
	oauthParams.Set("oauth_signature", signature)

	req, err := http.NewRequestWithContext(context.Background(), "POST",
		"https://connectapi.garmin.com/oauth-service/oauth/access_token",
		strings.NewReader(oauthParams.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "OAuth "+oauthParams.Encode())
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

	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}

	return &GarminTokenResponse{
		AccessToken:       values.Get("oauth_token"),
		AccessTokenSecret: values.Get("oauth_token_secret"),
	}, nil
}

func (p *GarminProvider) getUserProfile(accessToken, accessTokenSecret string) (*GarminProfile, error) {
	oauthParams := url.Values{}
	oauthParams.Set("oauth_consumer_key", p.consumerKey)
	oauthParams.Set("oauth_signature_method", "HMAC-SHA1")
	oauthParams.Set("oauth_timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	oauthParams.Set("oauth_nonce", uuid.New().String())
	oauthParams.Set("oauth_version", "1.0")
	oauthParams.Set("oauth_token", accessToken)

	signingKey := url.QueryEscape(p.consumerSecret) + "&" + url.QueryEscape(accessTokenSecret)
	signature := p.signWithKey("GET", "https://connectapi.garmin.com/userprofile-service/userprofile/user-profile", oauthParams, signingKey)
	oauthParams.Set("oauth_signature", signature)

	req, err := http.NewRequestWithContext(context.Background(), "GET",
		"https://connectapi.garmin.com/userprofile-service/userprofile/user-profile", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "OAuth "+oauthParams.Encode())

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
	var profile GarminProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("decode profile response: %w", err)
	}
	return &profile, nil
}

func (p *GarminProvider) Disconnect(ctx context.Context, userID string) error {
	_, err := p.db.ExecContext(ctx, `
		UPDATE device_provider_accounts 
		SET is_active = FALSE, updated_at = NOW()
		WHERE user_id = $1 AND provider = 'garmin'
	`, userID)
	if err != nil {
		return fmt.Errorf("disconnect garmin: %w", err)
	}
	return nil
}

func (p *GarminProvider) ListProviders(ctx context.Context, userID string) (map[string]interface{}, error) {
	return listAccountProviders(ctx, p.db, userID, "garmin")
}

func (p *GarminProvider) encryptRefreshToken(token string) (string, error) {
	if p.encryptor == nil {
		return "", errors.New("encryptor not initialized")
	}
	ciphertext, err := p.encryptor.Encrypt([]byte(token))
	if err != nil {
		return "", fmt.Errorf("encrypt refresh token: %w", err)
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (p *GarminProvider) sign(method, baseURL string, params url.Values) string {
	signingKey := url.QueryEscape(p.consumerSecret) + "&"
	return p.signWithKey(method, baseURL, params, signingKey)
}

func (p *GarminProvider) signWithKey(method, baseURL string, params url.Values, signingKey string) string {
	signatureBase := strings.ToUpper(method) + "&" + url.QueryEscape(baseURL) + "&" + url.QueryEscape(params.Encode())
	mac := hmac.New(sha1.New, []byte(signingKey))
	mac.Write([]byte(signatureBase))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
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

var _ = rand.Read
