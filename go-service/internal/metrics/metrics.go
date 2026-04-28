package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ntrag_http_requests_total",
			Help: "Total number of HTTP requests handled by go-service.",
		},
		[]string{"method", "route", "status"},
	)

	HTTPRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ntrag_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	PythonAIRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ntrag_python_ai_requests_total",
			Help: "Total number of requests sent from go-service to python-ai.",
		},
		[]string{"operation", "result"},
	)

	PythonAIRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ntrag_python_ai_request_duration_seconds",
			Help:    "Duration of requests from go-service to python-ai in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)
)

func init() {
	prometheus.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestDurationSeconds,
		PythonAIRequestsTotal,
		PythonAIRequestDurationSeconds,
	)
}

func InstrumentHandler(route string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next(recorder, r)

		status := strconv.Itoa(recorder.statusCode)
		HTTPRequestsTotal.WithLabelValues(r.Method, route, status).Inc()
		HTTPRequestDurationSeconds.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
