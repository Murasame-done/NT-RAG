package main

import (
	"context"
	"encoding/json"
	"go-service/internal/ai"
	"go-service/internal/config"
	"go-service/internal/httpmw"
	"go-service/internal/metrics"
	"go-service/internal/repository"
	"go-service/internal/tracing"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type chatRequest struct {
	Message string `json:"message"`
}

type shutdowner interface {
	Shutdown(context.Context) error
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := config.Load()

	var tp shutdowner
	var err error
	if cfg.DisableOTelTracing {
		slog.Info("tracing_disabled",
			"event", "tracing_disabled",
			"service", "go-service",
		)
	} else {
		tp, err = tracing.InitProvider(context.Background(), cfg.OTelServiceName, cfg.OTelExporterOTLPEndpoint)
		if err != nil {
			slog.Error("tracing_init_failed",
				"event", "tracing_init_failed",
				"service", "go-service",
				"error", err,
			)
			os.Exit(1)
		}
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := tp.Shutdown(shutdownCtx); err != nil {
				slog.Error("tracing_shutdown_failed",
					"event", "tracing_shutdown_failed",
					"service", "go-service",
					"error", err,
				)
			}
		}()
	}

	db, err := repository.NewMySQL(cfg.DBDSN)
	if err != nil {
		slog.Error("database_connect_failed",
			"event", "database_connect_failed",
			"service", "go-service",
			"error", err,
		)
		os.Exit(1)
	}
	defer db.Close()

	aiClient := ai.NewClient(cfg.PythonAIURL, cfg.AITimeout())

	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/health", metrics.InstrumentHandler("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	mux.HandleFunc("/api/ai/chat", metrics.InstrumentHandler("/api/ai/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.Message == "" {
			http.Error(w, "message is required", http.StatusBadRequest)
			return
		}

		resp, err := aiClient.Chat(r.Context(), req.Message)
		if err != nil {
			fields := []any{
				"event", "python_ai_request_failed",
				"service", "go-service",
				"request_id", httpmw.GetRequestID(r.Context()),
				"error", err,
			}
			fields = append(fields, tracing.LogFieldsFromContext(r.Context())...)

			slog.Error("python_ai_request_failed", fields...)
			http.Error(w, "python-ai request failed: "+err.Error(), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			fields := []any{
				"event", "response_encode_failed",
				"service", "go-service",
				"request_id", httpmw.GetRequestID(r.Context()),
				"error", err,
			}
			fields = append(fields, tracing.LogFieldsFromContext(r.Context())...)

			slog.Error("response_encode_failed", fields...)
		}
	}))

	handler := httpmw.WithTracing(httpmw.WithRequestID(httpmw.WithAccessLog(mux)))

	slog.Info("server_starting",
		"event", "server_starting",
		"service", "go-service",
		"port", cfg.Port,
		"python_ai_url", cfg.PythonAIURL,
		"ai_timeout_seconds", cfg.AITimeoutSeconds,
		"otel_exporter_otlp_endpoint", cfg.OTelExporterOTLPEndpoint,
		"disable_otel_tracing", cfg.DisableOTelTracing,
	)

	if err := http.ListenAndServe(cfg.Port, handler); err != nil {
		slog.Error("server_failed_to_start",
			"event", "server_failed_to_start",
			"service", "go-service",
			"error", err,
		)
	}
}
