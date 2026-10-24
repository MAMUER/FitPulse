package withings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	headerContentType      = "Content-Type"
	contentTypeFormEncoded = "application/x-www-form-urlencoded"
)

const (
	v2Measure  = "/v2/measure"
	v2Sleep    = "/v2/sleep"
	v2Activity = "/v2/activity"
)

// postForm sends a form-encoded POST request and returns the response body.
func (c *WithingsClient) postForm(ctx context.Context, u string, form url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set(headerContentType, contentTypeFormEncoded)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	return resp, nil
}

type WithingsClient struct {
	config *Config
	http   *http.Client
	log    *zap.Logger
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

	resp, err := c.postForm(ctx, tokenURL, form)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

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
	Status int `json:"status"`
	Body   struct {
		MeasureGroups []WithingsMeasureGroup `json:"measuregrps"`
	} `json:"body"`
}

// postV2 отправляет form-запрос к одному из v2 эндпоинтов Withings
// (measure / sleep / activity), принимающих action и access_token.
func (c *WithingsClient) postV2(ctx context.Context, endpoint string, action, accessToken string, since time.Time) (*WithingsMeasureResponse, error) {
	u := c.config.APIBaseURL + endpoint
	form := url.Values{}
	form.Set("action", action)
	form.Set("access_token", accessToken)
	if !since.IsZero() {
		form.Set("lastupdate", strconv.FormatInt(since.Unix(), 10))
	}

	resp, err := c.postForm(ctx, u, form)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var out WithingsMeasureResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode %s response: %w", action, err)
	}

	if out.Status != 0 {
		return nil, fmt.Errorf("withings %s error status=%d", action, out.Status)
	}

	return &out, nil
}

func (c *WithingsClient) GetMeasures(ctx context.Context, accessToken string, since time.Time) (*WithingsMeasureResponse, error) {
	return c.postV2(ctx, v2Measure, "getmeas", accessToken, since)
}

func (c *WithingsClient) GetSleep(ctx context.Context, accessToken string, since time.Time) (*WithingsMeasureResponse, error) {
	return c.postV2(ctx, v2Sleep, "get", accessToken, since)
}

func (c *WithingsClient) GetActivity(ctx context.Context, accessToken string, since time.Time) (*WithingsMeasureResponse, error) {
	return c.postV2(ctx, v2Activity, "getactivity", accessToken, since)
}

func (c *WithingsClient) RefreshAccessToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	form := url.Values{}
	form.Set("action", "requesttoken")
	form.Set("client_id", c.config.ClientID)
	form.Set("client_secret", c.config.ClientSecret)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)

	resp, err := c.postForm(ctx, tokenURL, form)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode refresh response: %w", err)
	}

	if tokenResp.Status != 0 {
		return nil, fmt.Errorf("withings refresh error status=%d", tokenResp.Status)
	}

	return &tokenResp, nil
}
