// Package webhook provides Open Wearables webhook handling.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func parseTime(t *testing.T, value string) time.Time { //nolint:unparam
	t.Helper()
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("failed to parse time: %v", err)
	}
	return ts
}

func TestDecodePayload(t *testing.T) {
	body := []byte(`{"user_id":"user-1","source":"open_wearables","timestamp":"2024-01-01T00:00:00Z","metrics":[{"metric_type":"heart_rate","value":72.0}]}`)
	payload, err := DecodePayload(body)
	assert.NoError(t, err)
	assert.Equal(t, "user-1", payload.UserID)
	assert.Equal(t, SourceTypeOpenWearables, payload.Source)
	assert.Len(t, payload.Metrics, 1)
	assert.Equal(t, MetricTypeHeartRate, payload.Metrics[0].Type)
	assert.Equal(t, 72.0, payload.Metrics[0].Value)
}

func TestValidatePayload(t *testing.T) {
	t.Run("valid payload", func(t *testing.T) {
		payload := &OpenWearablesWebhookPayload{
			UserID:    "user-1",
			Source:    SourceTypeOpenWearables,
			Timestamp: parseTime(t, "2024-01-01T00:00:00Z"),
			Metrics:   []OpenWearablesMetric{{Type: MetricTypeHeartRate, Value: 72.0}},
		}
		assert.NoError(t, payload.Validate())
	})

	t.Run("empty user_id", func(t *testing.T) {
		payload := &OpenWearablesWebhookPayload{
			Source:    SourceTypeOpenWearables,
			Timestamp: parseTime(t, "2024-01-01T00:00:00Z"),
			Metrics:   []OpenWearablesMetric{{Type: MetricTypeHeartRate, Value: 72.0}},
		}
		assert.ErrorIs(t, payload.Validate(), ErrEmptyUserID)
	})

	t.Run("empty metrics", func(t *testing.T) {
		payload := &OpenWearablesWebhookPayload{
			UserID:    "user-1",
			Source:    SourceTypeOpenWearables,
			Timestamp: parseTime(t, "2024-01-01T00:00:00Z"),
			Metrics:   []OpenWearablesMetric{},
		}
		assert.ErrorIs(t, payload.Validate(), ErrEmptyMetrics)
	})

	t.Run("invalid source", func(t *testing.T) {
		payload := &OpenWearablesWebhookPayload{
			UserID:    "user-1",
			Source:    "unknown_source",
			Timestamp: parseTime(t, "2024-01-01T00:00:00Z"),
			Metrics:   []OpenWearablesMetric{{Type: MetricTypeHeartRate, Value: 72.0}},
		}
		assert.ErrorIs(t, payload.Validate(), ErrInvalidSourceType)
	})

	t.Run("invalid metric type", func(t *testing.T) {
		payload := &OpenWearablesWebhookPayload{
			UserID:    "user-1",
			Source:    SourceTypeOpenWearables,
			Timestamp: parseTime(t, "2024-01-01T00:00:00Z"),
			Metrics:   []OpenWearablesMetric{{Type: "invalid_type", Value: 72.0}},
		}
		assert.ErrorIs(t, payload.Validate(), ErrInvalidMetricType)
	})

	t.Run("metric timestamp defaults to payload timestamp", func(t *testing.T) {
		ts := parseTime(t, "2024-01-01T00:00:00Z")
		payload := &OpenWearablesWebhookPayload{
			UserID:    "user-1",
			Source:    SourceTypeOpenWearables,
			Timestamp: ts,
			Metrics:   []OpenWearablesMetric{{Type: MetricTypeHeartRate, Value: 72.0, Timestamp: time.Time{}}},
		}
		assert.NoError(t, payload.Validate())
		assert.Equal(t, ts, payload.Metrics[0].Timestamp)
	})
}

func TestWriteResponse(t *testing.T) {
	w := httptest.NewRecorder()
	WriteResponse(w, http.StatusOK, WebhookResponse{Status: "success"})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "success")
}

func TestLogFields(t *testing.T) {
	payload := &OpenWearablesWebhookPayload{
		UserID:    "user-1",
		Source:    SourceTypeOpenWearables,
		Timestamp: parseTime(t, "2024-01-01T00:00:00Z"),
		Metrics:   []OpenWearablesMetric{{Type: MetricTypeHeartRate, Value: 72.0}},
	}
	fields := LogFields(payload)
	assert.Len(t, fields, 4)
}

func TestValidateSignature(t *testing.T) {
	secret := []byte("super-secret")
	body := []byte(`{"user_id":"user-1","source":"open_wearables","timestamp":"2024-01-01T00:00:00Z","metrics":[{"metric_type":"heart_rate","value":72.0}]}`)

	t.Run("valid signature", func(t *testing.T) {
		mac := hmac.New(sha256.New, secret)
		_, _ = mac.Write(body)
		signature := hex.EncodeToString(mac.Sum(nil))
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		r.Header.Set("X-Open-Wearables-Signature", signature)
		assert.NoError(t, ValidateSignature(secret, r))
	})

	t.Run("invalid signature", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		r.Header.Set("X-Open-Wearables-Signature", "bad")
		assert.ErrorIs(t, ValidateSignature(secret, r), ErrInvalidSignature)
	})

	t.Run("missing signature", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		assert.ErrorIs(t, ValidateSignature(secret, r), ErrInvalidSignature)
	})
}

func TestServerHandleWebhook(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		mockDB := &mockDB{}
		log := zap.NewNop()
		server := NewServer("8085", mockDB, log)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/open-wearables/webhook", nil)
		server.handleWebhook(w, r)
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("invalid payload", func(t *testing.T) {
		mockDB := &mockDB{}
		log := zap.NewNop()
		server := NewServer("8085", mockDB, log)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/open-wearables/webhook", bytes.NewReader([]byte("invalid")))
		server.handleWebhook(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("timestamp out of range", func(t *testing.T) {
		mockDB := &mockDB{}
		log := zap.NewNop()
		server := NewServer("8085", mockDB, log)
		w := httptest.NewRecorder()
		old := time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339)
		body := []byte(fmt.Sprintf(`{"user_id":"user-1","source":"open_wearables","timestamp":"%s","nonce":"n1","metrics":[{"metric_type":"heart_rate","value":72.0}]}`, old))
		r := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/open-wearables/webhook", bytes.NewReader(body))
		server.handleWebhook(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("nonce reused", func(t *testing.T) {
		mockDB := &mockDBWithNonce{}
		log := zap.NewNop()
		server := NewServer("8085", mockDB, log)
		w := httptest.NewRecorder()
		now := time.Now().UTC().Format(time.RFC3339)
		body := []byte(fmt.Sprintf(`{"user_id":"user-1","source":"open_wearables","timestamp":"%s","nonce":"n1","metrics":[{"metric_type":"heart_rate","value":72.0}]}`, now))
		r := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/open-wearables/webhook", bytes.NewReader(body))
		server.handleWebhook(w, r)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("valid payload", func(t *testing.T) {
		mockDB := &mockDB{}
		log := zap.NewNop()
		server := NewServer("8085", mockDB, log)
		w := httptest.NewRecorder()
		now := time.Now().UTC().Format(time.RFC3339)
		body := []byte(fmt.Sprintf(`{"user_id":"user-1","source":"open_wearables","timestamp":"%s","metrics":[{"metric_type":"heart_rate","value":72.0}]}`, now))
		r := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/open-wearables/webhook", bytes.NewReader(body))
		server.handleWebhook(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestCheckAndSaveNonce(t *testing.T) {
	t.Run("new nonce", func(t *testing.T) {
		store := NewStorage(&mockDBForNewNonce{}, zap.NewNop())
		err := store.CheckAndSaveNonce(context.Background(), "user-1", "new-nonce", time.Now())
		assert.NoError(t, err)
	})

	t.Run("reused nonce", func(t *testing.T) {
		store := NewStorage(&mockDBWithNonce{}, zap.NewNop())
		err := store.CheckAndSaveNonce(context.Background(), "user-1", "n1", time.Now())
		assert.Error(t, err)
	})
}

type mockDB struct{}

func (m *mockDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	return &mockTx{}, nil
}

func (m *mockDB) GetSources(ctx context.Context, userID string) ([]SourceInfo, error) {
	return []SourceInfo{{Source: "open_wearables", SourceName: "open_wearables", ConnectedAt: time.Now()}}, nil
}

func (m *mockDB) DeleteBySource(ctx context.Context, userID, source string) (int64, error) {
	return 1, nil
}

type mockDBWithNonce struct{}

func (m *mockDBWithNonce) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	return &mockTxWithNonce{}, nil
}

func (m *mockDBWithNonce) GetSources(ctx context.Context, userID string) ([]SourceInfo, error) {
	return []SourceInfo{{Source: "open_wearables", SourceName: "open_wearables", ConnectedAt: time.Now()}}, nil
}

func (m *mockDBWithNonce) DeleteBySource(ctx context.Context, userID, source string) (int64, error) {
	return 1, nil
}

type mockDBForNewNonce struct{}

func (m *mockDBForNewNonce) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	return &mockTxForNewNonce{}, nil
}

func (m *mockDBForNewNonce) GetSources(ctx context.Context, userID string) ([]SourceInfo, error) {
	return []SourceInfo{{Source: "open_wearables", SourceName: "open_wearables", ConnectedAt: time.Now()}}, nil
}

func (m *mockDBForNewNonce) DeleteBySource(ctx context.Context, userID, source string) (int64, error) {
	return 1, nil
}

type mockTx struct{}

func (m *mockTx) ExecContext(ctx context.Context, query string, args ...interface{}) (Result, error) {
	return &mockResult{}, nil
}

func (m *mockTx) QueryRowContext(ctx context.Context, query string, args ...interface{}) RowScanner {
	return nil
}

func (m *mockTx) Commit() error {
	return nil
}

func (m *mockTx) Rollback() error {
	return nil
}

type mockTxWithNonce struct{}

func (m *mockTxWithNonce) ExecContext(ctx context.Context, query string, args ...interface{}) (Result, error) {
	return &mockResult{}, nil
}

func (m *mockTxWithNonce) QueryRowContext(ctx context.Context, query string, args ...interface{}) RowScanner {
	return sqlRowWithTime{time: time.Now()}
}

func (m *mockTxWithNonce) Commit() error {
	return nil
}

func (m *mockTxWithNonce) Rollback() error {
	return nil
}

type sqlRowNoRows struct{}

func (r sqlRowNoRows) Scan(dest ...interface{}) error {
	return sql.ErrNoRows
}

type mockTxForNewNonce struct{}

func (m *mockTxForNewNonce) ExecContext(ctx context.Context, query string, args ...interface{}) (Result, error) {
	return &mockResult{}, nil
}

func (m *mockTxForNewNonce) QueryRowContext(ctx context.Context, query string, args ...interface{}) RowScanner {
	return sqlRowNoRows{}
}

func (m *mockTxForNewNonce) Commit() error {
	return nil
}

func (m *mockTxForNewNonce) Rollback() error {
	return nil
}

type sqlRowWithTime struct {
	time time.Time
}

func (r sqlRowWithTime) Scan(dest ...interface{}) error {
	if len(dest) == 0 {
		return errors.New("no dest")
	}
	p, ok := dest[0].(*time.Time)
	if !ok {
		return errors.New("dest is not *time.Time")
	}
	*p = r.time
	return nil
}

type mockResult struct{}

func (m *mockResult) RowsAffected() (int64, error) {
	return 1, nil
}
