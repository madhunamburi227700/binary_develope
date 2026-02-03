package telemetry

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter           = otel.Meter("ai-guardian-api")
	requestsTotal   metric.Int64Counter
	requestDuration metric.Float64Histogram
	metricsReady    bool
)

func init() {
	var err error
	requestsTotal, err = meter.Int64Counter("http_requests_total",
		metric.WithDescription("Total HTTP requests"))
	if err != nil {
		return
	}
	requestDuration, err = meter.Float64Histogram("http_request_duration_seconds",
		metric.WithDescription("HTTP request duration"))
	if err != nil {
		return
	}
	metricsReady = true
}

// HTTPMiddleware provides metrics instrumentation
func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !metricsReady || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		route := r.URL.Path
		if rt := mux.CurrentRoute(r); rt != nil {
			if tpl, err := rt.GetPathTemplate(); err == nil {
				route = tpl
			}
		}

		attrs := metric.WithAttributes(
			attribute.String("method", r.Method),
			attribute.String("route", route),
			attribute.String("status", strconv.Itoa(rw.status)),
		)
		requestsTotal.Add(r.Context(), 1, attrs)
		requestDuration.Record(r.Context(), time.Since(start).Seconds(), attrs)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
