package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DBDSN                    string
	Port                     string
	PythonAIURL              string
	AITimeoutSeconds         int
	OTelExporterOTLPEndpoint string
	OTelServiceName          string
	DisableOTelTracing       bool
}

func Load() *Config {
	return &Config{
		DBDSN:                    getEnv("DB_DSN", "ntrag_user:ntrag_password@tcp(127.0.0.1:3307)/ntrag_db?parseTime=true"),
		Port:                     getAnyEnv([]string{"PORT", "Port"}, ":8080"),
		PythonAIURL:              getEnv("PYTHON_AI_URL", "http://127.0.0.1:8001"),
		AITimeoutSeconds:         getEnvAsInt("AI_TIMEOUT_SECONDS", 2),
		OTelExporterOTLPEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://tempo:4317"),
		OTelServiceName:          getEnv("OTEL_SERVICE_NAME", "go-service"),
		DisableOTelTracing:       getEnvAsBool("DISABLE_OTEL_TRACING", false),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getAnyEnv(keys []string, fallback string) string {
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			return value
		}
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvAsBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func (c *Config) AITimeout() time.Duration {
	return time.Duration(c.AITimeoutSeconds) * time.Second
}
