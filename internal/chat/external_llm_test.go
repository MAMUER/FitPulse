package chat

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeExternalLLMSenderForTest struct {
	response  string
	err       error
	calls     int
	gotCtx    context.Context
	gotUserID string
	gotMsg    string
}

func (f *fakeExternalLLMSenderForTest) Send(ctx context.Context, userID, message string) (string, error) {
	f.calls++
	f.gotCtx = ctx
	f.gotUserID = userID
	f.gotMsg = message
	return f.response, f.err
}

func testExternalLLMConfigForTest() ExternalLLMConfig {
	return ExternalLLMConfig{
		Enabled:     true,
		Provider:    "openai",
		APIKey:      "test-key",
		Model:       "gpt-test",
		Timeout:     5 * time.Second,
		MaxTokens:   256,
		Temperature: 0.5,
	}
}

func TestNewExternalLLMService(t *testing.T) {
	t.Run("returns nil when disabled", func(t *testing.T) {
		cfg := testExternalLLMConfigForTest()
		cfg.Enabled = false

		service := NewExternalLLMService(cfg, zap.NewNop())
		assert.Nil(t, service)
	})

	t.Run("returns service when enabled", func(t *testing.T) {
		cfg := testExternalLLMConfigForTest()

		service := NewExternalLLMService(cfg, zap.NewNop())
		require.NotNil(t, service)
		assert.Equal(t, cfg, service.config)
		assert.Nil(t, service.client)
	})
}

func TestExternalLLMServiceSetClient(t *testing.T) {
	service := NewExternalLLMService(testExternalLLMConfigForTest(), zap.NewNop())
	require.NotNil(t, service)

	client := &fakeExternalLLMSenderForTest{response: "hello"}
	service.SetClient(client)
	assert.Same(t, client, service.client)
}

func TestExternalLLMServiceIsUserAllowed(t *testing.T) {
	t.Run("all users allowed when list is empty", func(t *testing.T) {
		service := NewExternalLLMService(testExternalLLMConfigForTest(), zap.NewNop())
		require.NotNil(t, service)
		assert.True(t, service.IsUserAllowed("any-user"))
		assert.True(t, service.IsUserAllowed(""))
	})

	t.Run("restricted user list", func(t *testing.T) {
		cfg := testExternalLLMConfigForTest()
		cfg.AllowedUserIDs = []string{"user-1", "user-2"}

		service := NewExternalLLMService(cfg, zap.NewNop())
		require.NotNil(t, service)
		assert.True(t, service.IsUserAllowed("user-1"))
		assert.True(t, service.IsUserAllowed("user-2"))
		assert.False(t, service.IsUserAllowed("user-3"))
		assert.False(t, service.IsUserAllowed(""))
	})
}

func TestExternalLLMServiceQueryWithFallback(t *testing.T) {
	ctx := context.Background()

	t.Run("nil service falls back to FAQ", func(t *testing.T) {
		var service *ExternalLLMService

		answer, err := service.QueryWithFallback(ctx, "user-1", "Как изменить план?")
		require.NoError(t, err)
		assert.Equal(t, FindAnswer("Как изменить план?"), answer)
	})

	t.Run("nil client falls back to FAQ", func(t *testing.T) {
		service := NewExternalLLMService(testExternalLLMConfigForTest(), zap.NewNop())
		require.NotNil(t, service)

		answer, err := service.QueryWithFallback(ctx, "user-1", "Как изменить план?")
		require.NoError(t, err)
		assert.Equal(t, FindAnswer("Как изменить план?"), answer)
	})

	t.Run("user not allowed falls back to FAQ", func(t *testing.T) {
		cfg := testExternalLLMConfigForTest()
		cfg.AllowedUserIDs = []string{"user-1"}

		service := NewExternalLLMService(cfg, zap.NewNop())
		require.NotNil(t, service)

		client := &fakeExternalLLMSenderForTest{response: "external answer"}
		service.SetClient(client)

		answer, err := service.QueryWithFallback(ctx, "user-2", "Как изменить план?")
		require.NoError(t, err)
		assert.Equal(t, FindAnswer("Как изменить план?"), answer)
		assert.Zero(t, client.calls)
	})

	t.Run("client error falls back to FAQ", func(t *testing.T) {
		service := NewExternalLLMService(testExternalLLMConfigForTest(), zap.NewNop())
		require.NotNil(t, service)

		client := &fakeExternalLLMSenderForTest{err: errors.New("provider unavailable")}
		service.SetClient(client)

		answer, err := service.QueryWithFallback(ctx, "user-1", "хочу похудеть")
		require.NoError(t, err)
		assert.Equal(t, FindAnswer("хочу похудеть"), answer)
		assert.Equal(t, 1, client.calls)
	})

	t.Run("empty response falls back to FAQ", func(t *testing.T) {
		service := NewExternalLLMService(testExternalLLMConfigForTest(), zap.NewNop())
		require.NotNil(t, service)

		client := &fakeExternalLLMSenderForTest{response: ""}
		service.SetClient(client)

		answer, err := service.QueryWithFallback(ctx, "user-1", "Как изменить план?")
		require.NoError(t, err)
		assert.Equal(t, FindAnswer("Как изменить план?"), answer)
		assert.Equal(t, 1, client.calls)
	})

	t.Run("returns external response on success", func(t *testing.T) {
		service := NewExternalLLMService(testExternalLLMConfigForTest(), zap.NewNop())
		require.NotNil(t, service)

		client := &fakeExternalLLMSenderForTest{response: "external answer"}
		service.SetClient(client)

		answer, err := service.QueryWithFallback(ctx, "user-1", "любой вопрос")
		require.NoError(t, err)
		assert.Equal(t, "external answer", answer)
		assert.Equal(t, 1, client.calls)
		assert.Equal(t, "user-1", client.gotUserID)
		assert.Equal(t, "любой вопрос", client.gotMsg)
		assert.Equal(t, ctx, client.gotCtx)
	})

	t.Run("allowed user reaches the client", func(t *testing.T) {
		cfg := testExternalLLMConfigForTest()
		cfg.AllowedUserIDs = []string{"user-1"}

		service := NewExternalLLMService(cfg, zap.NewNop())
		require.NotNil(t, service)

		client := &fakeExternalLLMSenderForTest{response: "external answer"}
		service.SetClient(client)

		answer, err := service.QueryWithFallback(ctx, "user-1", "любой вопрос")
		require.NoError(t, err)
		assert.Equal(t, "external answer", answer)
		assert.Equal(t, 1, client.calls)
	})
}

func TestNewExternalLLMClientImpl(t *testing.T) {
	client := NewExternalLLMClientImpl("api-key", "gpt-test")
	require.NotNil(t, client)
	assert.Equal(t, "api-key", client.apiKey)
	assert.Equal(t, "gpt-test", client.model)
}

func TestExternalLLMClientImplSend(t *testing.T) {
	client := NewExternalLLMClientImpl("api-key", "gpt-test")

	response, err := client.Send(context.Background(), "user-1", "привет")
	require.NoError(t, err)
	assert.Equal(t, "[External LLM (gpt-test)] Response to: привет", response)
}
