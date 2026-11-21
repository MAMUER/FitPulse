package main

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roundTripFunc lets tests intercept http.DefaultClient without a real network.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// stubDefaultTransport replaces http.DefaultClient for the duration of the test.
func stubDefaultTransport(t *testing.T, fn roundTripFunc) {
	t.Helper()
	original := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: fn}
	t.Cleanup(func() { http.DefaultClient = original })
}

func telegramResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		Header:     make(http.Header),
	}
}

// TestNotifyOncall_SendsMessage — при заданных TELEGRAM_* уходит POST на
// api.telegram.org с chat_id и текстом жалобы.
func TestNotifyOncall_SendsMessage(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("TELEGRAM_CHAT_ID", "4242")
	t.Setenv("TELEGRAM_BOT_TOKEN_FILE", "")
	t.Setenv("TELEGRAM_CHAT_ID_FILE", "")

	var gotURL, gotBody string
	stubDefaultTransport(t, func(r *http.Request) (*http.Response, error) {
		gotURL = r.URL.String()
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		gotBody = string(body)
		return telegramResponse(http.StatusOK), nil
	})

	g := newTestGateway()
	g.notifyOncall("u1", "u2", "spam")

	assert.Contains(t, gotURL, "api.telegram.org/bottest-token/sendMessage")
	decoded, err := url.QueryUnescape(gotBody)
	require.NoError(t, err)
	assert.Contains(t, decoded, "chat_id=4242")
	assert.Contains(t, decoded, "reporter=u1")
	assert.Contains(t, decoded, "reported=u2")
	assert.Contains(t, decoded, "reason=spam")
}

// TestNotifyOncall_TransportError — сетевая ошибка не должна ронять обработчик.
func TestNotifyOncall_TransportError(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("TELEGRAM_CHAT_ID", "4242")
	t.Setenv("TELEGRAM_BOT_TOKEN_FILE", "")
	t.Setenv("TELEGRAM_CHAT_ID_FILE", "")

	stubDefaultTransport(t, func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network down")
	})

	g := newTestGateway()
	assert.NotPanics(t, func() { g.notifyOncall("u1", "u2", "spam") })
}

// TestNotifyOncall_NoCredentials — без кредов сетевой вызов не выполняется.
func TestNotifyOncall_NoCredentials(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")
	t.Setenv("TELEGRAM_BOT_TOKEN_FILE", "")
	t.Setenv("TELEGRAM_CHAT_ID_FILE", "")

	called := false
	stubDefaultTransport(t, func(*http.Request) (*http.Response, error) {
		called = true
		return telegramResponse(http.StatusOK), nil
	})

	g := newTestGateway()
	g.notifyOncall("u1", "u2", "spam")

	assert.False(t, called, "без кредов сетевой вызов выполняться не должен")
}
