package resilience

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestCircuitBreakerStateString(t *testing.T) {
	assert.Equal(t, "closed", StateClosed.String())
	assert.Equal(t, "open", StateOpen.String())
	assert.Equal(t, "half-open", StateHalfOpen.String())
	assert.Equal(t, "unknown", CircuitBreakerState(42).String())
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, 5, config.FailureThreshold)
	assert.Equal(t, 3, config.SuccessThreshold)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 1, config.MaxRequestsInHalfOpen)
}

func TestCircuitBreakerSuccessResetsFailureCount(t *testing.T) {
	config := Config{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          time.Second,
	}
	cb := NewCircuitBreaker("test", config)

	require.Error(t, cb.Execute(func() error { return errors.New("service error") }))
	require.Error(t, cb.Execute(func() error { return errors.New("service error") }))
	require.NoError(t, cb.Execute(func() error { return nil }))

	assert.Equal(t, StateClosed, cb.State())

	cb.mu.RLock()
	assert.Equal(t, 0, cb.failureCount)
	cb.mu.RUnlock()

	for i := 0; i < 2; i++ {
		require.Error(t, cb.Execute(func() error { return errors.New("service error") }))
	}
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreakerReopensAfterFailureInHalfOpen(t *testing.T) {
	config := Config{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          10 * time.Millisecond,
	}
	cb := NewCircuitBreaker("test", config)

	for i := 0; i < 2; i++ {
		require.Error(t, cb.Execute(func() error { return errors.New("service error") }))
	}
	require.Equal(t, StateOpen, cb.State())

	time.Sleep(20 * time.Millisecond)

	require.NoError(t, cb.Execute(func() error { return nil }))
	require.Equal(t, StateHalfOpen, cb.State())

	require.Error(t, cb.Execute(func() error { return errors.New("service error") }))
	assert.Equal(t, StateOpen, cb.State())

	require.ErrorIs(t, cb.Execute(func() error { return nil }), ErrCircuitOpen)
}

func TestCircuitBreakerRejectsRequestsWhileOpen(t *testing.T) {
	config := Config{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          time.Hour,
	}
	cb := NewCircuitBreaker("test", config)

	require.Error(t, cb.Execute(func() error { return errors.New("service error") }))
	require.Equal(t, StateOpen, cb.State())

	err := cb.Execute(func() error { return nil })
	assert.ErrorIs(t, err, ErrCircuitOpen)
}

func TestCircuitBreakerRejectsRequestsWhenHalfOpenBudgetExhausted(t *testing.T) {
	config := Config{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          10 * time.Millisecond,
	}
	cb := NewCircuitBreaker("test", config)

	for i := 0; i < 2; i++ {
		require.Error(t, cb.Execute(func() error { return errors.New("service error") }))
	}
	require.Equal(t, StateOpen, cb.State())

	cb.mu.Lock()
	cb.state = StateHalfOpen
	cb.successCount = config.SuccessThreshold
	cb.mu.Unlock()

	err := cb.Execute(func() error { return nil })
	assert.ErrorIs(t, err, ErrCircuitOpen)
}

func TestCircuitBreakerRejectsUnknownState(t *testing.T) {
	cb := NewCircuitBreaker("test", DefaultConfig())

	cb.mu.Lock()
	cb.state = CircuitBreakerState(99)
	cb.mu.Unlock()

	err := cb.Execute(func() error { return nil })
	assert.ErrorIs(t, err, ErrCircuitOpen)
}

func TestCircuitBreakerConcurrentExecute(t *testing.T) {
	cb := NewCircuitBreaker("concurrent", DefaultConfig())

	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			_ = cb.Execute(func() error {
				if index%2 == 0 {
					return errors.New("service error")
				}
				return nil
			})

			_ = cb.State()
		}(i)
	}
	wg.Wait()

	assert.NotEqual(t, CircuitBreakerState(42), cb.State())
}

func TestWithCircuitBreaker(t *testing.T) {
	t.Run("returns function result on success", func(t *testing.T) {
		called := false
		err := WithCircuitBreaker("test", DefaultConfig(), func() error {
			called = true
			return nil
		})
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("returns function error on failure", func(t *testing.T) {
		wantErr := errors.New("service error")
		err := WithCircuitBreaker("test", DefaultConfig(), func() error {
			return wantErr
		})
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("runs function in closed state", func(t *testing.T) {
		called := false
		err := WithCircuitBreaker("test", Config{
			FailureThreshold: 1,
			SuccessThreshold: 1,
			Timeout:          time.Hour,
		}, func() error {
			called = true
			return errors.New("service error")
		})
		require.Error(t, err)
		assert.True(t, called)
	})
}

func TestLoggingCircuitBreaker(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	log := zap.New(core)

	config := Config{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          time.Hour,
	}
	lcb := NewLoggingCircuitBreaker("svc", config, log)

	assert.Equal(t, StateClosed, lcb.State())
	require.NoError(t, lcb.Execute(func() error { return nil }))
	assert.Zero(t, logs.Len())

	require.Error(t, lcb.Execute(func() error { return errors.New("service error") }))
	assert.Equal(t, StateOpen, lcb.State())
	assert.Equal(t, 1, logs.FilterMessage("circuit breaker recorded failure").Len())

	err := lcb.Execute(func() error { return nil })
	assert.ErrorIs(t, err, ErrCircuitOpen)
	assert.Equal(t, 1, logs.FilterMessage("circuit breaker open, request rejected").Len())

	lcb.cb.mu.Lock()
	lcb.cb.state = StateHalfOpen
	lcb.cb.mu.Unlock()

	require.NoError(t, lcb.Execute(func() error { return nil }))
	assert.Equal(t, StateClosed, lcb.State())
	assert.Equal(t, 1, logs.FilterMessage("circuit breaker closed").Len())
}
