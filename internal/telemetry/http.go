package telemetry

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/logger"
)

func HTTPMiddleware(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if span := trace.SpanFromContext(r.Context()); span.SpanContext().IsValid() {
					traceID := span.SpanContext().TraceID().String()
					log.Info("request tracing",
						zap.String("trace_id", traceID),
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
					)
					w.Header().Set("X-Trace-ID", traceID)
				}
				next.ServeHTTP(w, r)
			}),
			"HTTP",
		)
	}
}
