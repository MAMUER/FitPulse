// Package telemetry provides OpenTelemetry tracing for gRPC and HTTP services.
package telemetry

import (
	"context"
	"fmt"
	"os"
	"sync"

	"go.opentelemetry.io/otel"
	otlptrace "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

var (
	shutdownOnce sync.Once
	tp           *sdktrace.TracerProvider
)

// InitTracer initializes OpenTelemetry tracing with OTLP export.
// It returns a shutdown function that should be deferred in main().
// If OTLP endpoint is not configured or initialization fails, it returns a no-op shutdown function.
func InitTracer() func(context.Context) error {
	return InitTracerWithContext(context.Background())
}

// InitTracerWithContext initializes OpenTelemetry tracing with OTLP export using the provided context.
// It returns a shutdown function that should be deferred in main().
// If OTLP endpoint is not configured or initialization fails, it returns a no-op shutdown function.
func InitTracerWithContext(ctx context.Context) func(context.Context) error {
	return initTracerWithFactories(ctx, otlptracegrpc.New, resource.New)
}

func initTracerWithFactories(
	ctx context.Context,
	newExporter func(context.Context, ...otlptracegrpc.Option) (*otlptrace.Exporter, error),
	newResource func(context.Context, ...resource.Option) (*resource.Resource, error),
) func(context.Context) error {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return noopShutdown
	}

	exp, err := newExporter(ctx, otlptracegrpc.WithEndpoint(endpoint))
	if err != nil {
		return noopShutdown
	}

	res, err := newResource(ctx, resource.WithAttributes(
		semconv.ServiceNameKey.String(serviceName()),
	))
	if err != nil {
		return noopShutdown
	}

	tp = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	return shutdown
}

func noopShutdown(context.Context) error {
	return nil
}

func shutdown(ctx context.Context) error {
	var shutdownErr error
	shutdownOnce.Do(func() {
		if tp != nil {
			shutdownErr = tp.Shutdown(ctx)
		}
	})
	if shutdownErr != nil {
		return fmt.Errorf("shutdown tracer: %w", shutdownErr)
	}
	return nil
}

func serviceName() string {
	if name := os.Getenv("OTEL_SERVICE_NAME"); name != "" {
		return name
	}
	if name := os.Getenv("SERVICE_NAME"); name != "" {
		return name
	}
	return "unknown-service"
}
