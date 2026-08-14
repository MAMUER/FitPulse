package telemetry

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func TestServiceName_FromOTEL_SERVICE_NAME(t *testing.T) {
	original := os.Getenv("OTEL_SERVICE_NAME")
	_ = os.Setenv("OTEL_SERVICE_NAME", "my-service")
	defer func() { _ = os.Setenv("OTEL_SERVICE_NAME", original) }()

	name := serviceName()
	assert.Equal(t, "my-service", name)
}

func TestServiceName_FromSERVICE_NAME(t *testing.T) {
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

func TestServiceName_Default(t *testing.T) {
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

func TestInitTracer_EmptyEndpoint(t *testing.T) {
	original := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	defer func() { _ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", original) }()

	shutdownFn := InitTracer()
	require.NotNil(t, shutdownFn)

	err := shutdownFn(context.Background())
	assert.NoError(t, err)
}

func TestInitTracer_InvalidEndpoint(t *testing.T) {
	original := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "invalid:endpoint:123")
	defer func() { _ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", original) }()

	shutdownFn := InitTracer()
	require.NotNil(t, shutdownFn)

	err := shutdownFn(context.Background())
	assert.NoError(t, err)
}

func TestShutdown_NoopWhenNotInitialized(t *testing.T) {
	err := noopShutdown(context.Background())
	assert.NoError(t, err)
}

func TestShutdown_MultipleCalls(t *testing.T) {
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

func TestResource_Attributes(t *testing.T) {
	ctx := context.Background()
	res, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceNameKey.String("test-service"),
	))
	require.NoError(t, err)
	assert.NotNil(t, res)
}

func TestServerHandlerOption_ReturnsOption(t *testing.T) {
	opt := ServerHandlerOption()
	assert.NotNil(t, opt)
}

func TestClientHandlerOption_ReturnsOption(t *testing.T) {
	opt := ClientHandlerOption()
	assert.NotNil(t, opt)
}

func TestInitTracer_ReturnsNoopOnError(t *testing.T) {
	original := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "bad-endpoint")
	defer func() { _ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", original) }()

	shutdownFn := InitTracer()
	require.NotNil(t, shutdownFn)

	err := shutdownFn(context.Background())
	assert.NoError(t, err)
}
