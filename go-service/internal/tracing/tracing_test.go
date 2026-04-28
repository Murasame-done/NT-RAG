package tracing

import (
	"context"
	"testing"

	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestNormalizeEndpoint(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "http with port", input: "http://tempo:4317", want: "tempo:4317"},
		{name: "https with slash", input: "https://tempo:4317/", want: "tempo:4317"},
		{name: "plain endpoint", input: "tempo:4317", want: "tempo:4317"},
		{name: "trim spaces", input: "  http://tempo:4317/  ", want: "tempo:4317"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeEndpoint(tt.input)
			if got != tt.want {
				t.Fatalf("normalizeEndpoint(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTraceHelpersFromContext(t *testing.T) {
	traceID, err := oteltrace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("Parse trace id : %v", err)
	}
	spanID, err := oteltrace.SpanIDFromHex("0123456789abcdef")
	if err != nil {
		t.Fatalf("parse span id: %v", err)
	}
	spanCtx := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: oteltrace.FlagsSampled,
		Remote:     true,
	})

	ctx := oteltrace.ContextWithSpanContext(context.Background(), spanCtx)

	if got := TraceIDFromContext(ctx); got != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("TraceIDFromContext() = %q", got)
	}

	if got := SpanIDFromContext(ctx); got != "0123456789abcdef" {
		t.Fatalf("SpanIDFromContext() = %q", got)
	}

	fields := LogFieldsFromContext(ctx)
	if len(fields) != 4 {
		t.Fatalf("LogFieldsFromContext() length = %d, want 4", len(fields))
	}
}
