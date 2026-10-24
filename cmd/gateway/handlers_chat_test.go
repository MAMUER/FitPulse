package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/chat"
)

func chatRequest(t *testing.T, userID, body string) *http.Request {
	t.Helper()
	ctx := context.Background()
	if userID != "" {
		ctx = withUserID(ctx, userID)
	}
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/chat", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func chatDecodeResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

type chatFailingWriter struct{}

func (chatFailingWriter) Header() http.Header        { return http.Header{} }
func (chatFailingWriter) Write([]byte) (int, error)  { return 0, errors.New("write failed") }
func (chatFailingWriter) WriteHeader(statusCode int) {}

func TestChatHandler_Success(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.chatHandler(w, chatRequest(t, "user-1", `{"message":"Как отменить тренировку?"}`))

	assert.Equal(t, http.StatusOK, w.Code)
	resp := chatDecodeResponse(t, w)
	assert.Equal(t, "ok", resp["status"])
	assert.Equal(t, "Как отменить тренировку?", resp["message"])
	assert.Equal(t, "user-1", resp["user_id"])
	assert.Equal(t, chat.FindAnswer("Как отменить тренировку?"), resp["answer"])
	assert.NotNil(t, resp["timestamp"])
}

func TestChatHandler_SuccessUnknownQuestion(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.chatHandler(w, chatRequest(t, "user-42", `{"message":"zzzz"}`))

	assert.Equal(t, http.StatusOK, w.Code)
	resp := chatDecodeResponse(t, w)
	assert.Equal(t, "user-42", resp["user_id"])
	assert.Equal(t, chat.FindAnswer("zzzz"), resp["answer"])
}

func TestChatHandler_InvalidJSON(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.chatHandler(w, chatRequest(t, "user-1", `{invalid`))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), errBadRequest)
}

func TestChatHandler_EmptyBody(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.chatHandler(w, chatRequest(t, "user-1", ""))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), errBadRequest)
}

func TestChatHandler_EmptyMessage(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.chatHandler(w, chatRequest(t, "user-1", `{"message":""}`))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "message обязателен")
}

func TestChatHandler_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.chatHandler(w, chatRequest(t, "", `{"message":"привет"}`))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), msgUnauthorized)
}

func TestChatHandler_EncodeError(t *testing.T) {
	g := newTestGateway()

	assert.NotPanics(t, func() {
		g.chatHandler(chatFailingWriter{}, chatRequest(t, "user-1", `{"message":"привет"}`))
	})
}
