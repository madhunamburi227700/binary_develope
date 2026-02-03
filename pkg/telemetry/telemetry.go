package telemetry

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// Config holds OpenTelemetry configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	Enabled        bool
}

// Provider manages OpenTelemetry resources
type Provider struct {
	meterProvider *metric.MeterProvider
	registry      *prometheus.Registry
	enabled       bool
}

var globalProvider *Provider

// Initialize sets up OpenTelemetry with Prometheus exporter
func Initialize(cfg Config) (*Provider, error) {
	if !cfg.Enabled {
		log.Info().Msg("Telemetry disabled")
		return &Provider{enabled: false}, nil
	}

	// Create resource (avoid merging with default to prevent schema version conflicts)
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion(cfg.ServiceVersion),
	)

	// Create Prometheus registry and exporter
	registry := prometheus.NewRegistry()
	exporter, err := promexporter.New(promexporter.WithRegisterer(registry))
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create and set meter provider
	mp := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(exporter),
	)
	otel.SetMeterProvider(mp)

	globalProvider = &Provider{
		meterProvider: mp,
		registry:      registry,
		enabled:       true,
	}

	log.Info().Str("service", cfg.ServiceName).Msg("Telemetry initialized")
	return globalProvider, nil
}

// MetricsHandler returns HTTP handler for /metrics endpoint
func MetricsHandler() http.Handler {
	if globalProvider == nil || globalProvider.registry == nil {
		return promhttp.Handler()
	}
	return promhttp.HandlerFor(globalProvider.registry, promhttp.HandlerOpts{})
}

// Shutdown gracefully shuts down the provider
func (p *Provider) Shutdown(ctx context.Context) error {
	if !p.enabled || p.meterProvider == nil {
		return nil
	}
	return p.meterProvider.Shutdown(ctx)
}
