package telemetry

import (
	"context"
	"os"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

var (
	shutdownOnce sync.Once
	tp           *sdktrace.TracerProvider
)

func InitTracer() func(context.Context) error {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317"
	}

	ctx := context.Background()

	exp, err := otlptracegrpc.New(ctx)
	if err != nil {
		return func(context.Context) error { return nil }
	}

	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceNameKey.String(serviceName())))
	if err != nil {
		return func(context.Context) error { return nil }
	}

	tp = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	shutdownFn := func(ctx context.Context) error {
		var shutdownErr error
		shutdownOnce.Do(func() {
			if tp != nil {
				shutdownErr = tp.Shutdown(ctx)
			}
		})
		return shutdownErr
	}

	return shutdownFn
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
