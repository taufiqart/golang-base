// Package tracing wires OpenTelemetry distributed tracing for the service.
//
// The OTLP exporter is configured through the standard OTEL_EXPORTER_OTLP_*
// environment variables (endpoint, headers, TLS), so this package only decides
// whether tracing is on and how spans are sampled.
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

// InstrumentationScope is the tracer name shown in trace backends.
const InstrumentationScope = "golang-base"

// Config controls whether a real trace provider replaces the default no-op one.
type Config struct {
	Enabled bool
	// ServiceName becomes the resource service.name attribute.
	ServiceName string
	// Version becomes service.version, so releases are distinguishable in traces.
	Version string
	// Environment becomes deployment.environment.name.
	Environment string
	// SampleRatio is the head-sampling rate in (0, 1]. Out-of-range values
	// become 1 so a typo cannot silently drop every span.
	SampleRatio float64
}

// Setup installs the global tracer provider and W3C propagators.
//
// It returns a shutdown function that flushes buffered spans; the returned
// function is always safe to call. When cfg.Enabled is false no provider is
// installed, and every tracer call site falls back to OTel's zero-cost no-op,
// so callers never need to check whether tracing is on.
func Setup(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create otlp trace exporter: %w", err)
	}

	res, err := buildResource(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// ParentBased keeps a sampling decision made upstream, so one request stays
	// either fully traced or fully dropped across services.
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRatio(cfg.SampleRatio)))

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)
	otel.SetTracerProvider(provider)

	return provider.Shutdown, nil
}

func sampleRatio(ratio float64) float64 {
	if ratio <= 0 || ratio > 1 {
		return 1
	}
	return ratio
}

func buildResource(ctx context.Context, cfg Config) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{semconv.ServiceName(cfg.ServiceName)}
	if cfg.Version != "" {
		attrs = append(attrs, semconv.ServiceVersion(cfg.Version))
	}
	if cfg.Environment != "" {
		// Written as a literal key: the semconv package renames this helper
		// between versions, and the attribute name itself is stable.
		attrs = append(attrs, attribute.String("deployment.environment.name", cfg.Environment))
	}

	// Merge with default detection so host and process attributes survive.
	return resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL, attrs...,
	))
}
