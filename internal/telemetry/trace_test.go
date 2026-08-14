package telemetry

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace"
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

	shutdownFn := InitTracer()
	require.NotNil(t, shutdownFn)

	err := shutdownFn(context.Background())
	assert.NoError(t, err)
}

func TestTraceInitTracer_InvalidEndpoint(t *testing.T) {
	original := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "invalid:endpoint:123")
	defer func() { _ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", original) }()

	shutdownFn := InitTracer()
	require.NotNil(t, shutdownFn)

	err := shutdownFn(context.Background())
	assert.NoError(t, err)
}

func TestTraceNoopShutdown(t *testing.T) {
	err := noopShutdown(context.Background())
	assert.NoError(t, err)
}

func TestTraceShutdown_NoopWhenNotInitialized(t *testing.T) {
	shutdownFn := InitTracer()
	require.NotNil(t, shutdownFn)

	err := shutdownFn(context.Background())
	assert.NoError(t, err)
}

func TestTraceShutdown_MultipleCalls(t *testing.T) {
	original := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	defer func() { _ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", original) }()

	shutdownFn := InitTracer()
	require.NotNil(t, shutdownFn)

	err1 := shutdownFn(context.Background())
	err2 := shutdownFn(context.Background())

	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

func TestTraceInitTracer_ReturnsNoopOnError(t *testing.T) {
	original := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "bad-endpoint")
	defer func() { _ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", original) }()

	shutdownFn := InitTracer()
	require.NotNil(t, shutdownFn)

	err := shutdownFn(context.Background())
	assert.NoError(t, err)
}

func TestTraceInitTracer_ReturnsNoopOnResourceError(t *testing.T) {
	original := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://[::1]:4317")
	defer func() { _ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", original) }()

	shutdownFn := InitTracer()
	require.NotNil(t, shutdownFn)

	err := shutdownFn(context.Background())
	assert.NoError(t, err)
}

func TestTraceInitTracerWithContext_ReturnsNoopOnExporterError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	original := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")
	defer func() { _ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", original) }()

	shutdownFn := InitTracerWithContext(ctx)
	require.NotNil(t, shutdownFn)

	err := shutdownFn(context.Background())
	assert.NoError(t, err)
}

func TestTraceInitTracerWithContext_ReturnsNoopOnResourceError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	original := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")
	defer func() { _ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", original) }()

	shutdownFn := InitTracerWithContext(ctx)
	require.NotNil(t, shutdownFn)

	err := shutdownFn(context.Background())
	assert.NoError(t, err)
}

type failingExporter struct{}

func (e *failingExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	return nil
}

func (e *failingExporter) Shutdown(ctx context.Context) error {
	return errors.New("forced shutdown failure")
}

func resetShutdownState() {
	shutdownOnce = sync.Once{}
	tp = nil
}

func TestTraceShutdown_ReturnsErrorWhenTracerFails(t *testing.T) {
	resetShutdownState()
	tp = trace.NewTracerProvider(trace.WithBatcher(&failingExporter{}))
	defer func() { tp = nil }()

	err := shutdown(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "shutdown tracer")
}
