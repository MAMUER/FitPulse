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
	"go.uber.org/zap"
	"google.golang.org/grpc"

	userpb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/chat"
)

type chatUserServiceClient struct {
	mockUserServiceClient
	profile *userpb.UserProfile
	err     error
}

func (m *chatUserServiceClient) GetProfile(ctx context.Context, req *userpb.GetProfileRequest, opts ...grpc.CallOption) (*userpb.UserProfile, error) {
	return m.profile, m.err
}

type chatLLMSender struct {
	answer   string
	err      error
	sentMsgs []string
}

func (s *chatLLMSender) Send(ctx context.Context, userID, message string) (string, error) {
	s.sentMsgs = append(s.sentMsgs, userID, message)
	if s.err != nil {
		return "", s.err
	}
	return s.answer, nil
}

type chatFailingWriter struct{}

func (chatFailingWriter) Header() http.Header        { return http.Header{} }
func (chatFailingWriter) Write([]byte) (int, error)  { return 0, errors.New("write failed") }
func (chatFailingWriter) WriteHeader(statusCode int) {}

func chatLLMService(t *testing.T, sender chat.ExternalLLMSender) *chat.ExternalLLMService {
	t.Helper()
	svc := chat.NewExternalLLMService(chat.ExternalLLMConfig{
		Enabled:  true,
		Provider: "stub",
		Model:    "stub-model",
	}, zap.NewNop())
	require.NotNil(t, svc)
	svc.SetClient(sender)
	return svc
}

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
	g.chatHandler(w, chatRequest(t, "user-1", ``))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChatHandler_MissingMessage(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.chatHandler(w, chatRequest(t, "user-1", `{}`))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "message")
}

func TestChatHandler_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	g.chatHandler(w, chatRequest(t, "", `{"message":"привет"}`))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), msgUnauthorized)
}

func TestChatHandler_LLMAnswerUsed(t *testing.T) {
	g := newTestGateway()
	g.userClient = &chatUserServiceClient{
		profile: &userpb.UserProfile{UserId: "user-1", AiAssistantEnabled: true},
	}
	sender := &chatLLMSender{answer: "external llm answer"}
	g.externalLLM = chatLLMService(t, sender)

	w := httptest.NewRecorder()
	g.chatHandler(w, chatRequest(t, "user-1", `{"message":"Как отменить тренировку?"}`))

	assert.Equal(t, http.StatusOK, w.Code)
	resp := chatDecodeResponse(t, w)
	assert.Equal(t, "external llm answer", resp["answer"])
	assert.Equal(t, []string{"user-1", "Как отменить тренировку?"}, sender.sentMsgs)
}

func TestChatHandler_LLMFallbackOnError(t *testing.T) {
	g := newTestGateway()
	g.userClient = &chatUserServiceClient{
		profile: &userpb.UserProfile{UserId: "user-1", AiAssistantEnabled: true},
	}
	sender := &chatLLMSender{err: errors.New("llm down")}
	g.externalLLM = chatLLMService(t, sender)

	w := httptest.NewRecorder()
	g.chatHandler(w, chatRequest(t, "user-1", `{"message":"Как отменить тренировку?"}`))

	assert.Equal(t, http.StatusOK, w.Code)
	resp := chatDecodeResponse(t, w)
	assert.Equal(t, chat.FindAnswer("Как отменить тренировку?"), resp["answer"])
}

func TestChatHandler_LLMEmptyAnswerFallsBack(t *testing.T) {
	g := newTestGateway()
	g.userClient = &chatUserServiceClient{
		profile: &userpb.UserProfile{UserId: "user-1", AiAssistantEnabled: true},
	}
	sender := &chatLLMSender{answer: ""}
	g.externalLLM = chatLLMService(t, sender)

	w := httptest.NewRecorder()
	g.chatHandler(w, chatRequest(t, "user-1", `{"message":"Как отменить тренировку?"}`))

	assert.Equal(t, http.StatusOK, w.Code)
	resp := chatDecodeResponse(t, w)
	assert.Equal(t, chat.FindAnswer("Как отменить тренировку?"), resp["answer"])
}

func TestChatHandler_LLMProfileError(t *testing.T) {
	g := newTestGateway()
	g.userClient = &chatUserServiceClient{err: errors.New("profile boom")}
	sender := &chatLLMSender{answer: "external llm answer"}
	g.externalLLM = chatLLMService(t, sender)

	w := httptest.NewRecorder()
	g.chatHandler(w, chatRequest(t, "user-1", `{"message":"Как отменить тренировку?"}`))

	assert.Equal(t, http.StatusOK, w.Code)
	resp := chatDecodeResponse(t, w)
	assert.Equal(t, chat.FindAnswer("Как отменить тренировку?"), resp["answer"])
	assert.Empty(t, sender.sentMsgs)
}

func TestChatHandler_LLMAssistantDisabled(t *testing.T) {
	g := newTestGateway()
	g.userClient = &chatUserServiceClient{
		profile: &userpb.UserProfile{UserId: "user-1", AiAssistantEnabled: false},
	}
	sender := &chatLLMSender{answer: "external llm answer"}
	g.externalLLM = chatLLMService(t, sender)

	w := httptest.NewRecorder()
	g.chatHandler(w, chatRequest(t, "user-1", `{"message":"Как отменить тренировку?"}`))

	assert.Equal(t, http.StatusOK, w.Code)
	resp := chatDecodeResponse(t, w)
	assert.Equal(t, chat.FindAnswer("Как отменить тренировку?"), resp["answer"])
	assert.Empty(t, sender.sentMsgs)
}

func TestChatHandler_LLMNilProfile(t *testing.T) {
	g := newTestGateway()
	g.userClient = &chatUserServiceClient{profile: nil, err: nil}
	sender := &chatLLMSender{answer: "external llm answer"}
	g.externalLLM = chatLLMService(t, sender)

	w := httptest.NewRecorder()
	g.chatHandler(w, chatRequest(t, "user-1", `{"message":"Как отменить тренировку?"}`))

	assert.Equal(t, http.StatusOK, w.Code)
	resp := chatDecodeResponse(t, w)
	assert.Equal(t, chat.FindAnswer("Как отменить тренировку?"), resp["answer"])
	assert.Empty(t, sender.sentMsgs)
}

func TestChatHandler_EncodeError(t *testing.T) {
	g := newTestGateway()

	assert.NotPanics(t, func() {
		g.chatHandler(chatFailingWriter{}, chatRequest(t, "user-1", `{"message":"привет"}`))
	})
}
