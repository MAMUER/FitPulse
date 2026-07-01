package telemetry

import (
	"context"

	grpctrace "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func ServerHandlerOption() grpc.ServerOption {
	return grpc.StatsHandler(grpctrace.NewServerHandler())
}

func ClientHandlerOption() grpc.DialOption {
	return grpc.WithStatsHandler(grpctrace.NewClientHandler())
}

func LogTraceFromContext(ctx context.Context, log *zap.Logger, label string) {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		log.Info("gRPC tracing",
			zap.String(label, span.SpanContext().TraceID().String()),
		)
	}
}
