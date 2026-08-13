package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

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

func TestGRPC_LogTraceFromContext_WithValidSpan(t *testing.T) {
	log := logger.New("test")

	tp := sdktrace.NewTracerProvider()
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	LogTraceFromContext(ctx, log)
}
