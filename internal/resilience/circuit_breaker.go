// Package resilience provides circuit breaker and retry utilities for external service calls.
package resilience

import (
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

// CircuitBreakerState represents the state of a circuit breaker.
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	}
	return "unknown"
}

// CircuitBreaker implements the circuit breaker pattern for external service calls.
type CircuitBreaker struct {
	mu sync.RWMutex

	name            string
	state           CircuitBreakerState
	failureCount    int
	successCount    int
	lastFailureTime time.Time

	config Config
}

// Config holds circuit breaker configuration.
type Config struct {
	FailureThreshold      int
	SuccessThreshold      int
	Timeout               time.Duration
	MaxRequestsInHalfOpen int
}

// DefaultConfig returns sensible defaults for circuit breaker.
func DefaultConfig() Config {
	return Config{
		FailureThreshold:      5,
		SuccessThreshold:      3,
		Timeout:               30 * time.Second,
		MaxRequestsInHalfOpen: 1,
	}
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(name string, config Config) *CircuitBreaker {
	return &CircuitBreaker{
		name:   name,
		state:  StateClosed,
		config: config,
	}
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Execute executes the given function if the circuit breaker allows it.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.allowRequest() {
		return ErrCircuitOpen
	}

	err := fn()
	cb.recordResult(err)
	return err
}

func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := cb.state

	switch state {
	case StateClosed:
		return true
	case StateOpen:
		if cb.isTimeoutExpiredUnlocked() {
			cb.setState(StateHalfOpen)
			return true
		}
		return false
	case StateHalfOpen:
		return cb.successCount < cb.config.SuccessThreshold
	}
	return false
}

func (cb *CircuitBreaker) isTimeoutExpired() bool {
	cb.mu.RLock()
	lastFailure := cb.lastFailureTime
	timeout := cb.config.Timeout
	cb.mu.RUnlock()

	return time.Since(lastFailure) >= timeout
}

func (cb *CircuitBreaker) isTimeoutExpiredUnlocked() bool {
	return time.Since(cb.lastFailureTime) >= cb.config.Timeout
}

func (cb *CircuitBreaker) recordResult(err error) {
	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
}

func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.state == StateHalfOpen {
		cb.setState(StateOpen)
		return
	}

	if cb.failureCount >= cb.config.FailureThreshold {
		cb.setState(StateOpen)
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.setState(StateClosed)
		}
		return
	}

	cb.failureCount = 0
}

func (cb *CircuitBreaker) setState(state CircuitBreakerState) {
	cb.state = state
	if state == StateClosed {
		cb.failureCount = 0
		cb.successCount = 0
	} else if state == StateOpen {
		cb.successCount = 0
	}
}

// ErrCircuitOpen is returned when the circuit breaker is open.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// WithCircuitBreaker wraps an external service client with circuit breaker protection.
func WithCircuitBreaker(name string, config Config, fn func() error) error {
	cb := NewCircuitBreaker(name, config)
	return cb.Execute(fn)
}

// Logger is a minimal logger interface for circuit breaker events.
type Logger interface {
	Warn(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
}

// LoggingCircuitBreaker wraps a CircuitBreaker with logging.
type LoggingCircuitBreaker struct {
	cb  *CircuitBreaker
	log Logger
}

// NewLoggingCircuitBreaker creates a circuit breaker with logging.
func NewLoggingCircuitBreaker(name string, config Config, log Logger) *LoggingCircuitBreaker {
	return &LoggingCircuitBreaker{
		cb:  NewCircuitBreaker(name, config),
		log: log,
	}
}

// Execute executes the function with circuit breaker protection and logs state changes.
func (lcb *LoggingCircuitBreaker) Execute(fn func() error) error {
	state := lcb.cb.State()
	if state == StateOpen {
		lcb.log.Warn("circuit breaker open, request rejected",
			zap.String("name", lcb.cb.name),
			zap.String("state", state.String()),
		)
		return ErrCircuitOpen
	}

	err := lcb.cb.Execute(fn)
	if err != nil {
		lcb.log.Warn("circuit breaker recorded failure",
			zap.String("name", lcb.cb.name),
			zap.Error(err),
			zap.String("state", lcb.cb.State().String()),
		)
	} else if lcb.cb.State() == StateClosed && state == StateHalfOpen {
		lcb.log.Info("circuit breaker closed",
			zap.String("name", lcb.cb.name),
		)
	}

	return err
}

// State returns the current state.
func (lcb *LoggingCircuitBreaker) State() CircuitBreakerState {
	return lcb.cb.State()
}
