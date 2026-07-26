// Package telemetry provides OpenTelemetry tracing for gRPC and HTTP services.
package telemetry

import (
	"context"

	grpctrace "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/MAMUER/project/internal/logger"
)

// ServerHandlerOption returns a gRPC server option that enables OpenTelemetry tracing.
func ServerHandlerOption() grpc.ServerOption {
	return grpc.StatsHandler(grpctrace.NewServerHandler())
}

// ClientHandlerOption returns a gRPC dial option that enables OpenTelemetry tracing.
func ClientHandlerOption() grpc.DialOption {
	return grpc.WithStatsHandler(grpctrace.NewClientHandler())
}

// LogTraceFromContext logs the trace ID from the current span if valid.
func LogTraceFromContext(ctx context.Context, log *logger.Logger) {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		log.Info("gRPC tracing", zap.String("trace_id", span.SpanContext().TraceID().String()))
	}
}
