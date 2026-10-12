package withings

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type rewriteTransport struct {
	target *url.URL
}

func (rt rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = rt.target.Scheme
	cloned.URL.Host = rt.target.Host
	cloned.Host = ""
	return http.DefaultTransport.RoundTrip(cloned)
}

type errorTransport struct{}

func (errorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("dial tcp refused")
}

type formRecorder struct {
	calls  int
	method string
	path   string
	form   url.Values
}

func newTestConfig(apiBase string) Config {
	return Config{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "https://fitpulse.example/callback",
		APIBaseURL:   apiBase,
		Scopes:       []string{"user.metrics", "user.activity"},
	}
}

func newTestClient(cfg Config, rt http.RoundTripper) *WithingsClient {
	if rt == nil {
		rt = http.DefaultTransport
	}
	return &WithingsClient{
		config: &cfg,
		http:   &http.Client{Timeout: 5 * time.Second, Transport: rt},
		log:    zap.NewNop(),
	}
}

func newAPIServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *formRecorder) {
	t.Helper()
	recorder := &formRecorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder.calls++
		recorder.method = r.Method
		recorder.path = r.URL.Path
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		values, err := url.ParseQuery(string(body))
		require.NoError(t, err)
		recorder.form = values
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	recorder.form = url.Values{}
	return srv, recorder
}

func newClientForServer(t *testing.T, handler http.HandlerFunc) (*WithingsClient, *formRecorder) {
	t.Helper()
	srv, recorder := newAPIServer(t, handler)
	cfg := newTestConfig(srv.URL)
	client := newTestClient(cfg, rewriteTransport{target: mustParseURL(t, srv.URL)})
	return client, recorder
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	require.NoError(t, json.NewEncoder(w).Encode(payload))
}

func tokenPayload(access, refresh string) map[string]interface{} {
	return map[string]interface{}{
		"status": 0,
		"body": map[string]interface{}{
			"access_token":  access,
			"refresh_token": refresh,
			"expires_in":    7200,
			"token_type":    "Bearer",
		},
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	return parsed
}

func canceledContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestNewClient(t *testing.T) {
	cfg := newTestConfig("https://api.withings.example")

	client := NewClient(&cfg, zap.NewNop())

	require.NotNil(t, client)
	assert.Equal(t, &cfg, client.config)
	assert.NotNil(t, client.http)
	assert.Equal(t, "withings", client.log.Name())
}

func TestBuildAuthorizeURL(t *testing.T) {
	t.Run("missing client id", func(t *testing.T) {
		cfg := newTestConfig("https://api.withings.example")
		cfg.ClientID = ""
		client := newTestClient(cfg, nil)

		got, err := client.BuildAuthorizeURL("state-token")
		require.Error(t, err)
		assert.Empty(t, got)
		assert.Contains(t, err.Error(), "client_id is not configured")
	})

	t.Run("builds url with all parameters", func(t *testing.T) {
		cfg := newTestConfig("https://api.withings.example")
		cfg.Scopes = []string{"user.metrics", "user.activity", "user.sleep"}
		client := newTestClient(cfg, nil)

		got, err := client.BuildAuthorizeURL("state-xyz")
		require.NoError(t, err)

		parsed, err := url.Parse(got)
		require.NoError(t, err)
		assert.Equal(t, "account.withings.com", parsed.Host)
		assert.Equal(t, "/oauth2_user/authorize2", parsed.Path)

		q := parsed.Query()
		assert.Equal(t, "client-id", q.Get("client_id"))
		assert.Equal(t, "https://fitpulse.example/callback", q.Get("redirect_uri"))
		assert.Equal(t, "code", q.Get("response_type"))
		assert.Equal(t, "user.metrics,user.activity,user.sleep", q.Get("scope"))
		assert.Equal(t, "state-xyz", q.Get("state"))
	})

	t.Run("empty scopes produce empty scope parameter", func(t *testing.T) {
		cfg := newTestConfig("https://api.withings.example")
		cfg.Scopes = nil
		client := newTestClient(cfg, nil)

		got, err := client.BuildAuthorizeURL("state")
		require.NoError(t, err)

		parsed, err := url.Parse(got)
		require.NoError(t, err)
		assert.Equal(t, "", parsed.Query().Get("scope"))
	})
}

func TestExchangeCode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, recorder := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]interface{}{
				"status": 0,
				"body": map[string]interface{}{
					"access_token":  "access-1",
					"refresh_token": "refresh-1",
					"expires_in":    3600,
					"token_type":    "Bearer",
					"scope":         "user.metrics",
				},
			})
		})

		token, err := client.ExchangeCode(context.Background(), "auth-code-123")

		require.NoError(t, err)
		require.NotNil(t, token)
		assert.Equal(t, 0, token.Status)
		assert.Equal(t, "access-1", token.Body.AccessToken)
		assert.Equal(t, "refresh-1", token.Body.RefreshToken)
		assert.Equal(t, 3600, token.Body.ExpiresIn)
		assert.Equal(t, "Bearer", token.Body.TokenType)
		assert.Equal(t, "user.metrics", token.Body.Scope)

		require.Equal(t, 1, recorder.calls)
		assert.Equal(t, http.MethodPost, recorder.method)
		assert.Equal(t, "/v2/oauth2", recorder.path)
		assert.Equal(t, "requesttoken", recorder.form.Get("action"))
		assert.Equal(t, "authorization_code", recorder.form.Get("grant_type"))
		assert.Equal(t, "auth-code-123", recorder.form.Get("code"))
		assert.Equal(t, "client-id", recorder.form.Get("client_id"))
		assert.Equal(t, "client-secret", recorder.form.Get("client_secret"))
		assert.Equal(t, "https://fitpulse.example/callback", recorder.form.Get("redirect_uri"))
		assert.Equal(t, "user.metrics,user.activity", recorder.form.Get("scope"))
	})

	t.Run("omits scope when none configured", func(t *testing.T) {
		client, recorder := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, tokenPayload("access-1", "refresh-1"))
		})
		client.config.Scopes = nil

		_, err := client.ExchangeCode(context.Background(), "code")
		require.NoError(t, err)
		_, ok := recorder.form["scope"]
		assert.False(t, ok)
	})

	t.Run("non-zero status", func(t *testing.T) {
		client, _ := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]interface{}{"status": 503})
		})

		token, err := client.ExchangeCode(context.Background(), "code")
		require.Error(t, err)
		assert.Nil(t, token)
		assert.Contains(t, err.Error(), "withings token error status=503")
	})

	t.Run("empty access token", func(t *testing.T) {
		client, _ := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]interface{}{"status": 0, "body": map[string]interface{}{}})
		})

		token, err := client.ExchangeCode(context.Background(), "code")
		require.Error(t, err)
		assert.Nil(t, token)
		assert.Contains(t, err.Error(), "access_token is empty")
	})

	t.Run("http error status", func(t *testing.T) {
		client, _ := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusInternalServerError, map[string]interface{}{"status": 500})
		})

		token, err := client.ExchangeCode(context.Background(), "code")
		require.Error(t, err)
		assert.Nil(t, token)
		assert.Contains(t, err.Error(), "token error status=500")
	})

	t.Run("malformed json", func(t *testing.T) {
		client, _ := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": 0, "body": `))
		})

		token, err := client.ExchangeCode(context.Background(), "code")
		require.Error(t, err)
		assert.Nil(t, token)
		assert.Contains(t, err.Error(), "decode token response")
	})

	t.Run("transport error", func(t *testing.T) {
		client := newTestClient(newTestConfig("https://wbsapi.withings.net"), errorTransport{})

		token, err := client.ExchangeCode(context.Background(), "code")
		require.Error(t, err)
		assert.Nil(t, token)
		assert.Contains(t, err.Error(), "token request failed")
	})

	t.Run("canceled context", func(t *testing.T) {
		client, _ := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, tokenPayload("access-1", "refresh-1"))
		})

		token, err := client.ExchangeCode(canceledContext(t), "code")
		require.Error(t, err)
		assert.Nil(t, token)
		assert.Contains(t, err.Error(), "token request failed")
	})
}

func TestRefreshAccessToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, recorder := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, tokenPayload("access-2", "refresh-2"))
		})

		token, err := client.RefreshAccessToken(context.Background(), "refresh-token-9")

		require.NoError(t, err)
		require.NotNil(t, token)
		assert.Equal(t, "access-2", token.Body.AccessToken)
		assert.Equal(t, "refresh-2", token.Body.RefreshToken)

		assert.Equal(t, "/v2/oauth2", recorder.path)
		assert.Equal(t, "requesttoken", recorder.form.Get("action"))
		assert.Equal(t, "refresh_token", recorder.form.Get("grant_type"))
		assert.Equal(t, "refresh-token-9", recorder.form.Get("refresh_token"))
		assert.Equal(t, "client-id", recorder.form.Get("client_id"))
		assert.Equal(t, "client-secret", recorder.form.Get("client_secret"))
	})

	t.Run("non-zero status", func(t *testing.T) {
		client, _ := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]interface{}{"status": 401})
		})

		token, err := client.RefreshAccessToken(context.Background(), "refresh")
		require.Error(t, err)
		assert.Nil(t, token)
		assert.Contains(t, err.Error(), "withings refresh error status=401")
	})

	t.Run("malformed json", func(t *testing.T) {
		client, _ := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("not-json"))
		})

		token, err := client.RefreshAccessToken(context.Background(), "refresh")
		require.Error(t, err)
		assert.Nil(t, token)
		assert.Contains(t, err.Error(), "decode refresh response")
	})

	t.Run("transport error", func(t *testing.T) {
		client := newTestClient(newTestConfig("https://wbsapi.withings.net"), errorTransport{})

		token, err := client.RefreshAccessToken(context.Background(), "refresh")
		require.Error(t, err)
		assert.Nil(t, token)
		assert.Contains(t, err.Error(), "refresh request failed")
	})

	t.Run("canceled context", func(t *testing.T) {
		client, _ := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, tokenPayload("access", "refresh"))
		})

		token, err := client.RefreshAccessToken(canceledContext(t), "refresh")
		require.Error(t, err)
		assert.Nil(t, token)
		assert.Contains(t, err.Error(), "refresh request failed")
	})
}

func TestGetMeasures(t *testing.T) {
	t.Run("success with since", func(t *testing.T) {
		client, recorder := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]interface{}{
				"status": 0,
				"body": map[string]interface{}{
					"measuregrps": []map[string]interface{}{
						{
							"date": 1700000000,
							"measures": []map[string]interface{}{
								{"value": 73500, "unit": -3, "type": 1},
							},
						},
					},
				},
			})
		})

		resp, err := client.GetMeasures(context.Background(), "access-token", time.Unix(1700000000, 0))

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Body.MeasureGroups, 1)
		group := resp.Body.MeasureGroups[0]
		assert.Equal(t, int64(1700000000), group.Timestamp)
		require.Len(t, group.Measures, 1)
		assert.Equal(t, float64(73500), group.Measures[0].Value)
		assert.Equal(t, -3, group.Measures[0].Unit)
		assert.Equal(t, 1, group.Measures[0].Type)

		assert.Equal(t, "/v2/measure", recorder.path)
		assert.Equal(t, "getmeas", recorder.form.Get("action"))
		assert.Equal(t, "access-token", recorder.form.Get("access_token"))
		assert.Equal(t, "1700000000", recorder.form.Get("lastupdate"))
	})

	t.Run("zero since omits lastupdate", func(t *testing.T) {
		client, recorder := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]interface{}{"status": 0, "body": map[string]interface{}{}})
		})

		resp, err := client.GetMeasures(context.Background(), "access-token", time.Time{})

		require.NoError(t, err)
		require.NotNil(t, resp)
		_, ok := recorder.form["lastupdate"]
		assert.False(t, ok)
	})

	t.Run("non-zero status", func(t *testing.T) {
		client, _ := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]interface{}{"status": 401})
		})

		resp, err := client.GetMeasures(context.Background(), "access-token", time.Time{})
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "withings getmeas error status=401")
	})

	t.Run("malformed json", func(t *testing.T) {
		client, _ := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("{invalid"))
		})

		resp, err := client.GetMeasures(context.Background(), "access-token", time.Time{})
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "decode getmeas response")
	})

	t.Run("transport error", func(t *testing.T) {
		client := newTestClient(newTestConfig("https://wbsapi.withings.net"), errorTransport{})

		resp, err := client.GetMeasures(context.Background(), "access-token", time.Time{})
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "getmeas request failed")
	})

	t.Run("canceled context", func(t *testing.T) {
		client, _ := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]interface{}{"status": 0})
		})

		resp, err := client.GetMeasures(canceledContext(t), "access-token", time.Time{})
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "getmeas request failed")
	})
}

func TestGetSleep(t *testing.T) {
	t.Run("success with series", func(t *testing.T) {
		client, recorder := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]interface{}{
				"status": 0,
				"body": map[string]interface{}{
					"series": []map[string]interface{}{
						{"startdate": 1700000000, "enddate": 1700003600, "state": 0},
					},
				},
			})
		})

		resp, err := client.GetSleep(context.Background(), "access-token", time.Unix(1700000000, 0))

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Body.MeasureGroups)

		assert.Equal(t, "/v2/sleep", recorder.path)
		assert.Equal(t, "get", recorder.form.Get("action"))
		assert.Equal(t, "1700000000", recorder.form.Get("lastupdate"))
	})

	t.Run("empty series", func(t *testing.T) {
		client, _ := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]interface{}{"status": 0, "body": map[string]interface{}{}})
		})

		resp, err := client.GetSleep(context.Background(), "access-token", time.Time{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Body.MeasureGroups)
	})

	t.Run("error status", func(t *testing.T) {
		client, _ := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]interface{}{"status": 1000})
		})

		resp, err := client.GetSleep(context.Background(), "access-token", time.Time{})
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "withings get error status=1000")
	})

	t.Run("malformed json", func(t *testing.T) {
		client, _ := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("null"))
		})

		resp, err := client.GetSleep(context.Background(), "access-token", time.Time{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, 0, resp.Status)
	})

	t.Run("transport error", func(t *testing.T) {
		client := newTestClient(newTestConfig("https://wbsapi.withings.net"), errorTransport{})

		resp, err := client.GetSleep(context.Background(), "access-token", time.Time{})
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "get request failed")
	})
}

func TestGetActivity(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, recorder := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]interface{}{
				"status": 0,
				"body": map[string]interface{}{
					"activities": []map[string]interface{}{
						{"date": "2026-01-02", "steps": 8500, "calories": 2100.5, "distance": 6200},
					},
				},
			})
		})

		resp, err := client.GetActivity(context.Background(), "access-token", time.Time{})

		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, "/v2/activity", recorder.path)
		assert.Equal(t, "getactivity", recorder.form.Get("action"))
		_, ok := recorder.form["lastupdate"]
		assert.False(t, ok)
	})

	t.Run("error status", func(t *testing.T) {
		client, _ := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]interface{}{"status": 401})
		})

		resp, err := client.GetActivity(context.Background(), "access-token", time.Time{})
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "withings getactivity error status=401")
	})

	t.Run("malformed json", func(t *testing.T) {
		client, _ := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("<html/>"))
		})

		resp, err := client.GetActivity(context.Background(), "access-token", time.Time{})
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "decode getactivity response")
	})

	t.Run("http error status", func(t *testing.T) {
		client, _ := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		})

		resp, err := client.GetActivity(context.Background(), "access-token", time.Time{})
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "decode getactivity response")
	})

	t.Run("transport error", func(t *testing.T) {
		client := newTestClient(newTestConfig("https://wbsapi.withings.net"), errorTransport{})

		resp, err := client.GetActivity(context.Background(), "access-token", time.Time{})
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "getactivity request failed")
	})

	t.Run("canceled context", func(t *testing.T) {
		client, _ := newClientForServer(t, func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]interface{}{"status": 0})
		})

		resp, err := client.GetActivity(canceledContext(t), "access-token", time.Time{})
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "getactivity request failed")
	})
}

func TestPostV2UsesAPIBaseURL(t *testing.T) {
	srv, recorder := newAPIServer(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusOK, map[string]interface{}{"status": 0, "body": map[string]interface{}{}})
	})
	cfg := newTestConfig(srv.URL)
	cfg.APIBaseURL = srv.URL + "/api"
	client := newTestClient(cfg, rewriteTransport{target: mustParseURL(t, srv.URL)})

	resp, err := client.GetMeasures(context.Background(), "access-token", time.Time{})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "/api/v2/measure", recorder.path)
}
