package withings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	headerContentType      = "Content-Type"
	contentTypeFormEncoded = "application/x-www-form-urlencoded"
)

const (
	v2Measure = "/v2/measure"
	v2Sleep   = "/v2/sleep"
	v2Activity = "/v2/activity"
)

type WithingsClient struct {
	config   *Config
	http     *http.Client
	log      *zap.Logger
}

func NewClient(config *Config, log *zap.Logger) *WithingsClient {
	return &WithingsClient{
		config: config,
		http:   &http.Client{Timeout: 10 * time.Second},
		log:    log.Named("withings"),
	}
}

func (c *WithingsClient) BuildAuthorizeURL(state string) (string, error) {
	if c.config.ClientID == "" {
		return "", errors.New("withings client_id is not configured")
	}
	u, err := url.Parse(authorizeURL)
	if err != nil {
		return "", fmt.Errorf("parse authorize url: %w", err)
	}
	q := u.Query()
	q.Set("client_id", c.config.ClientID)
	q.Set("redirect_uri", c.config.RedirectURL)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(c.config.Scopes, ","))
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (c *WithingsClient) ExchangeCode(ctx context.Context, code string) (*TokenResponse, error) {
	form := url.Values{}
	form.Set("action", "requesttoken")
	form.Set("client_id", c.config.ClientID)
	form.Set("client_secret", c.config.ClientSecret)
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", c.config.RedirectURL)
	if len(c.config.Scopes) > 0 {
		form.Set("scope", strings.Join(c.config.Scopes, ","))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set(headerContentType, contentTypeFormEncoded)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	if tokenResp.Status != 0 {
		return nil, fmt.Errorf("withings token error status=%d", tokenResp.Status)
	}

	if tokenResp.Body.AccessToken == "" {
		return nil, errors.New("withings access_token is empty")
	}

	return &tokenResp, nil
}

type WithingsMeasure struct {
	Value     float64
	Unit      int
	Type      int
	Timestamp int64
}

type WithingsMeasureGroup struct {
	Timestamp int64             `json:"date"`
	Measures  []WithingsMeasure `json:"measures"`
}

type WithingsMeasureResponse struct {
	Status int                   `json:"status"`
	Body   struct {
		MeasureGroups []WithingsMeasureGroup `json:"measuregrps"`
	} `json:"body"`
}

func (c *WithingsClient) GetMeasures(ctx context.Context, accessToken string, since time.Time) (*WithingsMeasureResponse, error) {
	u := fmt.Sprintf("%s%s", c.config.APIBaseURL, v2Measure)
	form := url.Values{}
	form.Set("action", "getmeas")
	form.Set("access_token", accessToken)
	if !since.IsZero() {
		form.Set("lastupdate", fmt.Sprintf("%d", since.Unix()))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build measure request: %w", err)
	}
	req.Header.Set(headerContentType, contentTypeFormEncoded)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("measure request failed: %w", err)
	}
	defer resp.Body.Close()

	var measureResp WithingsMeasureResponse
	if err := json.NewDecoder(resp.Body).Decode(&measureResp); err != nil {
		return nil, fmt.Errorf("decode measure response: %w", err)
	}

	if measureResp.Status != 0 {
		return nil, fmt.Errorf("withings measure error status=%d", measureResp.Status)
	}

	return &measureResp, nil
}

func (c *WithingsClient) GetSleep(ctx context.Context, accessToken string, since time.Time) (*WithingsMeasureResponse, error) {
	u := fmt.Sprintf("%s%s", c.config.APIBaseURL, v2Sleep)
	form := url.Values{}
	form.Set("action", "get")
	form.Set("access_token", accessToken)
	if !since.IsZero() {
		form.Set("lastupdate", fmt.Sprintf("%d", since.Unix()))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build sleep request: %w", err)
	}
	req.Header.Set(headerContentType, contentTypeFormEncoded)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sleep request failed: %w", err)
	}
	defer resp.Body.Close()

	var sleepResp WithingsMeasureResponse
	if err := json.NewDecoder(resp.Body).Decode(&sleepResp); err != nil {
		return nil, fmt.Errorf("decode sleep response: %w", err)
	}

	if sleepResp.Status != 0 {
		return nil, fmt.Errorf("withings sleep error status=%d", sleepResp.Status)
	}

	return &sleepResp, nil
}

func (c *WithingsClient) GetActivity(ctx context.Context, accessToken string, since time.Time) (*WithingsMeasureResponse, error) {
	u := fmt.Sprintf("%s%s", c.config.APIBaseURL, v2Activity)
	form := url.Values{}
	form.Set("action", "getactivity")
	form.Set("access_token", accessToken)
	if !since.IsZero() {
		form.Set("lastupdate", fmt.Sprintf("%d", since.Unix()))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build activity request: %w", err)
	}
	req.Header.Set(headerContentType, contentTypeFormEncoded)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("activity request failed: %w", err)
	}
	defer resp.Body.Close()

	var activityResp WithingsMeasureResponse
	if err := json.NewDecoder(resp.Body).Decode(&activityResp); err != nil {
		return nil, fmt.Errorf("decode activity response: %w", err)
	}

	if activityResp.Status != 0 {
		return nil, fmt.Errorf("withings activity error status=%d", activityResp.Status)
	}

	return &activityResp, nil
}

func (c *WithingsClient) RefreshAccessToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	form := url.Values{}
	form.Set("action", "requesttoken")
	form.Set("client_id", c.config.ClientID)
	form.Set("client_secret", c.config.ClientSecret)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build refresh request: %w", err)
	}
	req.Header.Set(headerContentType, contentTypeFormEncoded)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode refresh response: %w", err)
	}

	if tokenResp.Status != 0 {
		return nil, fmt.Errorf("withings refresh error status=%d", tokenResp.Status)
	}

	return &tokenResp, nil
}
