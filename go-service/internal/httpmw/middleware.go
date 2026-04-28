package httpmw

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go-service/internal/tracing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type contextKey string

const requestIDKey contextKey = "request_id"

var httpTracer = otel.Tracer("go-service/http")

func GetRequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}

func WithTracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := tracing.ExtractContext(r.Context(), r.Header)
		spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)

		ctx, span := httpTracer.Start(ctx, spanName, oteltrace.WithSpanKind(oteltrace.SpanKindServer))
		defer span.End()

		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.target", r.URL.Path),
		)

		next.ServeHTTP(recorder, r.WithContext(ctx))

		span.SetAttributes(
			attribute.Int("http.status_code", recorder.statusCode),
		)

		if requestID := recorder.Header().Get("X-Request-ID"); requestID != "" {
			span.SetAttributes(attribute.String("request.id", requestID))
		}

		if recorder.statusCode >= 500 {
			span.SetStatus(codes.Error, http.StatusText(recorder.statusCode))
		}
	})
}

func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}

		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		w.Header().Set("X-Request-ID", requestID)

		if traceID := tracing.TraceIDFromContext(ctx); traceID != "" {
			w.Header().Set("X-Trace-ID", traceID)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func WithAccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		fields := []any{
			"event", "http_request_completed",
			"service", "go-service",
			"request_id", GetRequestID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
		}
		fields = append(fields, tracing.LogFieldsFromContext(r.Context())...)

		slog.Info("http_request_completed", fields...)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
