package resilience

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreakerClosedState(t *testing.T) {
	config := Config{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}
	cb := NewCircuitBreaker("test", config)

	assert.Equal(t, StateClosed, cb.State())

	err := cb.Execute(func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreakerOpensAfterFailures(t *testing.T) {
	config := Config{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}
	cb := NewCircuitBreaker("test", config)

	for i := 0; i < 3; i++ {
		err := cb.Execute(func() error {
			return errors.New("service error")
		})
		assert.Error(t, err)
	}

	assert.Equal(t, StateOpen, cb.State())

	err := cb.Execute(func() error {
		return nil
	})
	assert.ErrorIs(t, err, ErrCircuitOpen)
}

func TestCircuitBreakerHalfOpenAfterTimeout(t *testing.T) {
	config := Config{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	}
	cb := NewCircuitBreaker("test", config)

	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return errors.New("service error")
		})
	}

	assert.Equal(t, StateOpen, cb.State())
	time.Sleep(60 * time.Millisecond)

	err := cb.Execute(func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateHalfOpen, cb.State())
}

func TestCircuitBreakerClosesAfterSuccessInHalfOpen(t *testing.T) {
	config := Config{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	}
	cb := NewCircuitBreaker("test", config)

	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return errors.New("service error")
		})
	}

	time.Sleep(60 * time.Millisecond)

	for i := 0; i < 2; i++ {
		err := cb.Execute(func() error {
			return nil
		})
		assert.NoError(t, err)
	}

	assert.Equal(t, StateClosed, cb.State())
}
