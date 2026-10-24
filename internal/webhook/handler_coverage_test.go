package webhook

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestServerStartWithNilServer(t *testing.T) {
	t.Run("nil server logs an error and returns", func(t *testing.T) {
		server := NewServer("8085", &mockDB{}, zap.NewNop())
		server.server = nil
		server.Start()
	})
}

func TestServerRoutesThroughMux(t *testing.T) {
	server := NewServer("8085", &mockDB{}, zap.NewNop())
	handler := server.server.Handler
	require.NotNil(t, handler)

	t.Run("health endpoint", func(t *testing.T) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "ok", w.Body.String())
	})

	t.Run("metrics endpoint", func(t *testing.T) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("providers endpoint", func(t *testing.T) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/integrations/providers?user_id=user-1", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "providers")
	})

	t.Run("disconnect endpoint", func(t *testing.T) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/integrations/open-wearables/disconnect?user_id=user-1&source=open_wearables", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "disconnected")
	})

	t.Run("webhook endpoint with wrong method", func(t *testing.T) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/integrations/open-wearables/webhook", nil))
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("unknown endpoint", func(t *testing.T) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/unknown", nil))
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestReadBodyCloseError(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Body = &fakeCloseErrorBodyForHandlerTest{}

	body, err := readBody(r)
	require.Error(t, err)
	assert.Empty(t, body)
	assert.Contains(t, err.Error(), "close request body")
}

func TestReadBodySuccess(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"ok":true}`)))

	body, err := readBody(r)
	require.NoError(t, err)
	assert.JSONEq(t, `{"ok":true}`, string(body))
}

func TestReadBodyTooLarge(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", io.NopCloser(&limitedReaderForHandlerTest{remaining: 2 << 20}))

	body, err := readBody(r)
	require.Error(t, err)
	assert.NotEmpty(t, body)
}

type limitedReaderForHandlerTest struct {
	remaining int
}

func (l *limitedReaderForHandlerTest) Read(p []byte) (int, error) {
	if l.remaining <= 0 {
		return 0, io.EOF
	}
	if len(p) > l.remaining {
		p = p[:l.remaining]
	}
	for i := range p {
		p[i] = 'a'
	}
	l.remaining -= len(p)
	return len(p), nil
}

type fakeCloseErrorBodyForHandlerTest struct{}

func (f *fakeCloseErrorBodyForHandlerTest) Read(p []byte) (int, error) {
	return 0, io.EOF
}

func (f *fakeCloseErrorBodyForHandlerTest) Close() error {
	return errors.New("close failed")
}

func TestServerHandleWebhookInvalidRequestBody(t *testing.T) {
	server := NewServer("8085", &mockDB{}, zap.NewNop())
	server.secret = nil

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/open-wearables/webhook", nil)
	r.Body = &errorReader{err: errors.New("read failed")}

	server.handleWebhook(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request body")
}

func TestServerHandleWebhookNonceBeginTxFailure(t *testing.T) {
	server := NewServer("8085", &mockDBWithNonceBeginError{}, zap.NewNop())

	w := httptest.NewRecorder()
	now := time.Now().UTC().Format(time.RFC3339)
	body := []byte(`{"user_id":"user-1","source":"open_wearables","timestamp":"` + now + `","nonce":"n1","metrics":[{"metric_type":"heart_rate","value":72.0}]}`)

	r := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/open-wearables/webhook", bytes.NewReader(body))
	r.Header.Set("X-Open-Wearables-Signature", signTestBody(t, server.secret, body))

	server.handleWebhook(w, r)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "duplicate webhook nonce")
}

func TestServerStopIsIdempotent(t *testing.T) {
	server := NewServer("8085", &mockDB{}, zap.NewNop())

	require.NoError(t, server.Stop(context.Background()))
	require.NoError(t, server.Stop(context.Background()))
}

type mockDBWithNonceBeginError struct{}

func (m *mockDBWithNonceBeginError) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	return nil, errors.New("database error")
}

func (m *mockDBWithNonceBeginError) ExecContext(ctx context.Context, query string, args ...interface{}) (Rower, error) {
	return &mockRower{}, nil
}

func (m *mockDBWithNonceBeginError) GetSources(ctx context.Context, userID string) ([]SourceInfo, error) {
	return nil, nil
}

func (m *mockDBWithNonceBeginError) DeleteBySource(ctx context.Context, userID, source string) (int64, error) {
	return 0, nil
}

type failingHealthWriter struct {
	http.ResponseWriter
}

func (f *failingHealthWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestServerHealthEndpointWriteError(t *testing.T) {
	mockDB := &mockDB{}
	log := zap.NewNop()
	server := NewServer("8085", mockDB, log)

	w := &failingHealthWriter{ResponseWriter: httptest.NewRecorder()}
	r := httptest.NewRequest(http.MethodGet, "/health", nil)

	server.server.Handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.ResponseWriter.(*httptest.ResponseRecorder).Code)
}

type mockDBWithPurgeError struct{}

func (m *mockDBWithPurgeError) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	return &mockTx{}, nil
}

func (m *mockDBWithPurgeError) ExecContext(ctx context.Context, query string, args ...interface{}) (Rower, error) {
	return &mockRower{}, nil
}

func (m *mockDBWithPurgeError) GetSources(ctx context.Context, userID string) ([]SourceInfo, error) {
	return nil, nil
}

func (m *mockDBWithPurgeError) DeleteBySource(ctx context.Context, userID, source string) (int64, error) {
	return 0, nil
}

type failingProvidersWriter struct {
	http.ResponseWriter
}

func (f *failingProvidersWriter) WriteHeader(int) {}
func (f *failingProvidersWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestServerHandleListProvidersWriteError(t *testing.T) {
	mockDB := &mockDB{}
	log := zap.NewNop()
	server := NewServer("8085", mockDB, log)

	w := &failingProvidersWriter{ResponseWriter: httptest.NewRecorder()}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/providers?user_id=user-1", nil)

	server.handleListProviders(w, r)

	rec := w.ResponseWriter.(*httptest.ResponseRecorder)
	assert.Equal(t, http.StatusOK, rec.Code)
}
