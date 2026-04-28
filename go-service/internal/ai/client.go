package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"go-service/internal/httpmw"
	"go-service/internal/metrics"
	"go-service/internal/tracing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Reply string `json:"reply"`
	Model string `json:"model"`
}

var pythonAITracer = otel.Tracer("go-service/python-ai")

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Chat(ctx context.Context, message string) (*ChatResponse, error) {
	ctx, span := pythonAITracer.Start(ctx, "python-ai chat", oteltrace.WithSpanKind(oteltrace.SpanKindClient))
	defer span.End()

	start := time.Now()
	result := "success"
	requestID := httpmw.GetRequestID(ctx)

	span.SetAttributes(
		attribute.String("http.method", http.MethodPost),
		attribute.String("http.url", c.baseURL+"/chat"),
		attribute.String("ai.operation", "chat"),
	)

	if requestID != "" {
		span.SetAttributes(attribute.String("request.id", requestID))
	}

	defer func() {
		metrics.PythonAIRequestsTotal.WithLabelValues("chat", result).Inc()
		metrics.PythonAIRequestDurationSeconds.WithLabelValues("chat").Observe(time.Since(start).Seconds())

		fields := []any{
			"event", "python_ai_call_completed",
			"service", "go-service",
			"request_id", requestID,
			"operation", "chat",
			"result", result,
			"duration_ms", time.Since(start).Milliseconds(),
		}
		fields = append(fields, tracing.LogFieldsFromContext(ctx)...)

		slog.Info("python_ai_call_completed", fields...)
	}()

	payload, err := json.Marshal(ChatRequest{Message: message})
	if err != nil {
		result = "marshal_error"
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat", bytes.NewReader(payload))
	if err != nil {
		result = "request_create_error"
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}

	tracing.InjectHeaders(ctx, req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		result = "request_error"
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("call python-ai: %w", err)
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		result = fmt.Sprintf("status_%d", resp.StatusCode)
		span.SetStatus(codes.Error, fmt.Sprintf("python-ai status %d", resp.StatusCode))
		return nil, fmt.Errorf("python-ai returned status %d", resp.StatusCode)
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		result = "decode_error"
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("decode python-ai response: %w", err)
	}

	return &chatResp, nil
}
