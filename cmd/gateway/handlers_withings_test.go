package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/withings"
)

type withingsRewriteTransport struct {
	orig    http.RoundTripper
	apiHost string
}

func (rt *withingsRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.Host, "wbsapi.withings.net") {
		req.URL.Scheme = "http"
		req.URL.Host = rt.apiHost
	}
	return rt.orig.RoundTrip(req)
}

func withingsInstallRewriteTransport(t *testing.T, apiHost string) {
	t.Helper()
	orig := http.DefaultTransport
	http.DefaultTransport = &withingsRewriteTransport{orig: orig, apiHost: apiHost}
	t.Cleanup(func() { http.DefaultTransport = orig })
}

func withingsNewDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func withingsExpectDevice(mock sqlmock.Sqlmock, token string) {
	mock.ExpectQuery("SELECT id, user_id, device_type").
		WithArgs("user-1", "withings").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "device_type", "device_name", "token", "is_connected", "last_sync"}).
			AddRow("dev-1", "user-1", "withings", "Withings", token, true, time.Now()))
}

func withingsTokenServer(t *testing.T, status int, accessToken string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": status,
			"body": map[string]interface{}{
				"access_token":  accessToken,
				"refresh_token": "refresh-token",
				"expires_in":    7200,
				"token_type":    "Bearer",
			},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func withingsMeasuresServer(t *testing.T) *httptest.Server {
	t.Helper()
	groups := map[string]int{"/v2/measure": 2, "/v2/sleep": 1, "/v2/activity": 3}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		count := groups[r.URL.Path]
		measureGroups := make([]map[string]int64, 0, count)
		for i := 0; i < count; i++ {
			measureGroups = append(measureGroups, map[string]int64{"date": int64(1700000000 + i)})
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": 0,
			"body":   map[string]interface{}{"measuregrps": measureGroups},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newWithingsGateway(t *testing.T, apiBase string, db *sql.DB) *gateway {
	t.Helper()
	g := newTestGateway()
	g.withingsClient = withings.NewClient(&withings.Config{
		ClientID:     "withings-client-id",
		ClientSecret: "withings-client-secret",
		RedirectURL:  "http://localhost/api/v1/integrations/withings/callback",
		APIBaseURL:   apiBase,
		Scopes:       withings.DefaultScopes(),
	}, zap.NewNop())
	if db != nil {
		g.deviceClient = newGatewayDeviceClient(db)
	}
	return g
}

func withingsGatewayWithoutClient() *gateway {
	return newTestGateway()
}

func withingsRequest(method, target string) *http.Request {
	return httptest.NewRequest(method, target, nil)
}

func withingsAuthedRequest(target string) *http.Request {
	return httptest.NewRequestWithContext(withUserID(context.Background(), "user-1"), "GET", target, nil)
}

func TestWithingsLoginHandler(t *testing.T) {
	t.Run("not configured", func(t *testing.T) {
		w := httptest.NewRecorder()

		withingsGatewayWithoutClient().withingsLoginHandler(w, withingsRequest("GET", "/api/v1/integrations/withings/auth"))

		assert.Equal(t, http.StatusNotImplemented, w.Code)
		assert.Contains(t, w.Body.String(), errWithingsNotConfigured)
	})

	t.Run("missing client id", func(t *testing.T) {
		g := newWithingsGateway(t, "http://127.0.0.1:1", nil)
		g.withingsClient = withings.NewClient(&withings.Config{
			ClientSecret: "secret",
			RedirectURL:  "http://localhost/callback",
			APIBaseURL:   "http://127.0.0.1:1",
		}, zap.NewNop())

		w := httptest.NewRecorder()

		g.withingsLoginHandler(w, withingsRequest("GET", "/api/v1/integrations/withings/auth"))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "failed to build authorize url")
	})

	t.Run("redirects with state cookie", func(t *testing.T) {
		g := newWithingsGateway(t, "http://127.0.0.1:1", nil)

		w := httptest.NewRecorder()

		g.withingsLoginHandler(w, withingsRequest("GET", "/api/v1/integrations/withings/auth"))

		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)

		location := w.Header().Get("Location")
		assert.Contains(t, location, "oauth2_user/authorize2")
		assert.Contains(t, location, "withings-client-id")
		assert.Contains(t, location, "state=")

		cookies := w.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, withingsOAuthStateCookie, cookies[0].Name)
		assert.NotEmpty(t, cookies[0].Value)
		assert.True(t, cookies[0].HttpOnly)
		assert.True(t, cookies[0].Secure)
	})
}

func TestWithingsCallbackHandler(t *testing.T) {
	t.Run("not configured", func(t *testing.T) {
		w := httptest.NewRecorder()

		withingsGatewayWithoutClient().withingsCallbackHandler(w, withingsRequest("GET", "/api/v1/integrations/withings/callback?state=s&code=c"))

		assert.Equal(t, http.StatusNotImplemented, w.Code)
	})

	t.Run("missing state cookie", func(t *testing.T) {
		g := newWithingsGateway(t, "http://127.0.0.1:1", nil)

		w := httptest.NewRecorder()

		g.withingsCallbackHandler(w, withingsRequest("GET", "/api/v1/integrations/withings/callback?state=s&code=c"))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid oauth state")
	})

	t.Run("state mismatch", func(t *testing.T) {
		g := newWithingsGateway(t, "http://127.0.0.1:1", nil)

		w := httptest.NewRecorder()
		req := withingsRequest("GET", "/api/v1/integrations/withings/callback?state=other&code=c")
		req.AddCookie(&http.Cookie{Name: withingsOAuthStateCookie, Value: "expected-state"})

		g.withingsCallbackHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty state param", func(t *testing.T) {
		g := newWithingsGateway(t, "http://127.0.0.1:1", nil)

		w := httptest.NewRecorder()
		req := withingsRequest("GET", "/api/v1/integrations/withings/callback?code=c")
		req.AddCookie(&http.Cookie{Name: withingsOAuthStateCookie, Value: "expected-state"})

		g.withingsCallbackHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty state cookie", func(t *testing.T) {
		g := newWithingsGateway(t, "http://127.0.0.1:1", nil)

		w := httptest.NewRecorder()
		req := withingsRequest("GET", "/api/v1/integrations/withings/callback?state=s&code=c")
		req.AddCookie(&http.Cookie{Name: withingsOAuthStateCookie, Value: ""})

		g.withingsCallbackHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing authorization code", func(t *testing.T) {
		g := newWithingsGateway(t, "http://127.0.0.1:1", nil)

		w := httptest.NewRecorder()
		req := withingsRequest("GET", "/api/v1/integrations/withings/callback?state=s")
		req.AddCookie(&http.Cookie{Name: withingsOAuthStateCookie, Value: "s"})

		g.withingsCallbackHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "missing authorization code")
	})

	t.Run("unauthorized", func(t *testing.T) {
		g := newWithingsGateway(t, "http://127.0.0.1:1", nil)

		w := httptest.NewRecorder()
		req := withingsRequest("GET", "/api/v1/integrations/withings/callback?state=s&code=c")
		req.AddCookie(&http.Cookie{Name: withingsOAuthStateCookie, Value: "s"})

		g.withingsCallbackHandler(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("token exchange error", func(t *testing.T) {
		srv := withingsTokenServer(t, 503, "")
		withingsInstallRewriteTransport(t, srv.Listener.Addr().String())
		g := newWithingsGateway(t, srv.URL, nil)

		w := httptest.NewRecorder()
		req := withingsAuthedRequest("/api/v1/integrations/withings/callback?state=s&code=c")
		req.AddCookie(&http.Cookie{Name: withingsOAuthStateCookie, Value: "s"})

		g.withingsCallbackHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "failed to exchange authorization code")
	})

	t.Run("device store error", func(t *testing.T) {
		srv := withingsTokenServer(t, 0, "access-token")
		withingsInstallRewriteTransport(t, srv.Listener.Addr().String())
		db, mock := withingsNewDB(t)
		mock.ExpectExec("INSERT INTO devices").WillReturnError(assert.AnError)

		g := newWithingsGateway(t, srv.URL, db)

		w := httptest.NewRecorder()
		req := withingsAuthedRequest("/api/v1/integrations/withings/callback?state=s&code=c")
		req.AddCookie(&http.Cookie{Name: withingsOAuthStateCookie, Value: "s"})

		g.withingsCallbackHandler(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "failed to store device")
	})

	t.Run("success", func(t *testing.T) {
		srv := withingsTokenServer(t, 0, "access-token")
		withingsInstallRewriteTransport(t, srv.Listener.Addr().String())
		db, mock := withingsNewDB(t)
		mock.ExpectExec("INSERT INTO devices").
			WithArgs(sqlmock.AnyArg(), "user-1", "withings", "Withings", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		g := newWithingsGateway(t, srv.URL, db)

		w := httptest.NewRecorder()
		req := withingsAuthedRequest("/api/v1/integrations/withings/callback?state=s&code=c")
		req.AddCookie(&http.Cookie{Name: withingsOAuthStateCookie, Value: "s"})

		g.withingsCallbackHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
		assert.Equal(t, "ok", payload["status"])
		assert.Equal(t, "withings", payload["source"])
		assert.Equal(t, "user-1", payload["user_id"])
		assert.NoError(t, mock.ExpectationsWereMet())

		for _, cookie := range w.Result().Cookies() {
			if cookie.Name == withingsOAuthStateCookie {
				assert.Empty(t, cookie.Value)
				assert.Equal(t, -1, cookie.MaxAge)
			}
		}
	})
}

func TestWithingsSyncHandler(t *testing.T) {
	t.Run("not configured", func(t *testing.T) {
		w := httptest.NewRecorder()

		withingsGatewayWithoutClient().withingsSyncHandler(w, withingsAuthedRequest("/api/v1/integrations/withings/sync"))

		assert.Equal(t, http.StatusNotImplemented, w.Code)
	})

	t.Run("unauthorized", func(t *testing.T) {
		g := newWithingsGateway(t, "http://127.0.0.1:1", nil)

		w := httptest.NewRecorder()

		g.withingsSyncHandler(w, withingsRequest("POST", "/api/v1/integrations/withings/sync"))

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("device not found", func(t *testing.T) {
		db, mock := withingsNewDB(t)
		mock.ExpectQuery("SELECT id, user_id, device_type").WillReturnError(sql.ErrNoRows)

		g := newWithingsGateway(t, "http://127.0.0.1:1", db)

		w := httptest.NewRecorder()

		g.withingsSyncHandler(w, withingsAuthedRequest("/api/v1/integrations/withings/sync"))

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Withings device not connected")
	})

	t.Run("device load error", func(t *testing.T) {
		db, mock := withingsNewDB(t)
		mock.ExpectQuery("SELECT id, user_id, device_type").WillReturnError(assert.AnError)

		g := newWithingsGateway(t, "http://127.0.0.1:1", db)

		w := httptest.NewRecorder()

		g.withingsSyncHandler(w, withingsAuthedRequest("/api/v1/integrations/withings/sync"))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "failed to load device")
	})

	t.Run("empty device token", func(t *testing.T) {
		db, mock := withingsNewDB(t)
		withingsExpectDevice(mock, "")

		g := newWithingsGateway(t, "http://127.0.0.1:1", db)

		w := httptest.NewRecorder()

		g.withingsSyncHandler(w, withingsAuthedRequest("/api/v1/integrations/withings/sync"))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "token is empty")
	})

	t.Run("invalid device token", func(t *testing.T) {
		db, mock := withingsNewDB(t)
		withingsExpectDevice(mock, "not-json")

		g := newWithingsGateway(t, "http://127.0.0.1:1", db)

		w := httptest.NewRecorder()

		g.withingsSyncHandler(w, withingsAuthedRequest("/api/v1/integrations/withings/sync"))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid device token")
	})

	t.Run("syncs all metrics", func(t *testing.T) {
		srv := withingsMeasuresServer(t)
		db, mock := withingsNewDB(t)
		withingsExpectDevice(mock, `{"access_token":"access-1","refresh_token":"refresh-1","expires_in":3600}`)
		mock.ExpectExec("UPDATE devices SET last_sync").WillReturnResult(sqlmock.NewResult(0, 1))

		g := newWithingsGateway(t, srv.URL, db)

		w := httptest.NewRecorder()

		g.withingsSyncHandler(w, withingsAuthedRequest("/api/v1/integrations/withings/sync"))

		assert.Equal(t, http.StatusOK, w.Code)

		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
		assert.Equal(t, "ok", payload["status"])
		assert.Equal(t, "user-1", payload["user_id"])
		assert.Equal(t, float64(6), payload["synced"])
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("upstream failures are tolerated", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		db, mock := withingsNewDB(t)
		withingsExpectDevice(mock, `{"access_token":"access-1"}`)
		mock.ExpectExec("UPDATE devices SET last_sync").WillReturnError(assert.AnError)

		g := newWithingsGateway(t, srv.URL, db)

		w := httptest.NewRecorder()

		g.withingsSyncHandler(w, withingsAuthedRequest("/api/v1/integrations/withings/sync"))

		assert.Equal(t, http.StatusOK, w.Code)

		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
		assert.Equal(t, float64(0), payload["synced"])
	})
}

func TestParseWithingsToken(t *testing.T) {
	g := newWithingsGateway(t, "http://127.0.0.1:1", nil)

	parsed, err := g.parseWithingsToken(`{"access_token":"a","refresh_token":"r","expires_in":60}`)
	require.NoError(t, err)
	assert.Equal(t, "a", parsed.AccessToken)
	assert.Equal(t, "r", parsed.RefreshToken)
	assert.Equal(t, 60, parsed.ExpiresIn)

	_, err = g.parseWithingsToken(`{`)
	require.Error(t, err)
}

func TestLastSyncTime(t *testing.T) {
	now := time.Now()

	assert.Equal(t, now, lastSyncTime(now))
	assert.Equal(t, now.Unix(), lastSyncTime(now.Format(time.RFC3339)).Unix())
	assert.True(t, lastSyncTime("not-a-time").IsZero())
	assert.True(t, lastSyncTime(nil).IsZero())
	assert.True(t, lastSyncTime(42).IsZero())
}
