package telemetry

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceServiceName_FromOTEL_SERVICE_NAME(t *testing.T) {
	original := os.Getenv("OTEL_SERVICE_NAME")
	_ = os.Setenv("OTEL_SERVICE_NAME", "my-service")
	defer func() { _ = os.Setenv("OTEL_SERVICE_NAME", original) }()

	name := serviceName()
	assert.Equal(t, "my-service", name)
}

func TestTraceServiceName_FromSERVICE_NAME(t *testing.T) {
	originalOTEL := os.Getenv("OTEL_SERVICE_NAME")
	originalSERVICE := os.Getenv("SERVICE_NAME")
	_ = os.Unsetenv("OTEL_SERVICE_NAME")
	_ = os.Setenv("SERVICE_NAME", "fallback-service")
	defer func() {
		_ = os.Setenv("OTEL_SERVICE_NAME", originalOTEL)
		_ = os.Setenv("SERVICE_NAME", originalSERVICE)
	}()

	name := serviceName()
	assert.Equal(t, "fallback-service", name)
}

func TestTraceServiceName_Default(t *testing.T) {
	originalOTEL := os.Getenv("OTEL_SERVICE_NAME")
	originalSERVICE := os.Getenv("SERVICE_NAME")
	_ = os.Unsetenv("OTEL_SERVICE_NAME")
	_ = os.Unsetenv("SERVICE_NAME")
	defer func() {
		_ = os.Setenv("OTEL_SERVICE_NAME", originalOTEL)
		_ = os.Setenv("SERVICE_NAME", originalSERVICE)
	}()

	name := serviceName()
	assert.Equal(t, "unknown-service", name)
}

func TestTraceInitTracer_EmptyEndpoint(t *testing.T) {
	original := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	defer func() { _ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", original) }()

	shutdown := InitTracer()
	require.NotNil(t, shutdown)

	err := shutdown(context.Background())
	assert.NoError(t, err)
}

func TestTraceInitTracer_InvalidEndpoint(t *testing.T) {
	original := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "invalid:endpoint:123")
	defer func() { _ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", original) }()

	shutdown := InitTracer()
	require.NotNil(t, shutdown)

	err := shutdown(context.Background())
	assert.NoError(t, err)
}

func TestTraceNoopShutdown(t *testing.T) {
	err := noopShutdown(context.Background())
	assert.NoError(t, err)
}

func TestTraceShutdown_NoopWhenNotInitialized(t *testing.T) {
	shutdown := InitTracer()
	require.NotNil(t, shutdown)

	err := shutdown(context.Background())
	assert.NoError(t, err)
}

func TestTraceShutdown_MultipleCalls(t *testing.T) {
	original := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	defer func() { _ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", original) }()

	shutdown := InitTracer()
	require.NotNil(t, shutdown)

	err1 := shutdown(context.Background())
	err2 := shutdown(context.Background())

	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

func TestTraceInitTracer_ReturnsNoopOnError(t *testing.T) {
	original := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "bad-endpoint")
	defer func() { _ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", original) }()

	shutdown := InitTracer()
	require.NotNil(t, shutdown)

	err := shutdown(context.Background())
	assert.NoError(t, err)
}
