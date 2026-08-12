package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MAMUER/project/internal/logger"
)

func TestGRPC_ServerHandlerOption_ReturnsOption(t *testing.T) {
	opt := ServerHandlerOption()
	assert.NotNil(t, opt)
}

func TestGRPC_ClientHandlerOption_ReturnsOption(t *testing.T) {
	opt := ClientHandlerOption()
	assert.NotNil(t, opt)
}

func TestGRPC_LogTraceFromContext_WithoutSpan(t *testing.T) {
	log := logger.New("test")
	LogTraceFromContext(context.Background(), log)
}

func TestGRPC_LogTraceFromContext_WithValidLogger(t *testing.T) {
	log := logger.New("test")
	LogTraceFromContext(context.Background(), log)
	assert.NotNil(t, log)
}
