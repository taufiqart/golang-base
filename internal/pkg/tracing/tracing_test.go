package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestSetupDisabledLeavesProviderAlone(t *testing.T) {
	previous := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	shutdown, err := Setup(context.Background(), Config{Enabled: false, ServiceName: "svc"})
	require.NoError(t, err)
	require.NotNil(t, shutdown, "the returned shutdown must always be callable")
	assert.NoError(t, shutdown(context.Background()))

	assert.Same(t, previous, otel.GetTracerProvider(),
		"a disabled setup must not replace the provider, so callers stay on the cheap no-op")
}

func TestSetupInstallsProviderAndPropagators(t *testing.T) {
	previous := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	cfg := Config{
		Enabled:     true,
		ServiceName: "tracing-test",
		Version:     "v9.9.9",
		Environment: "test",
		SampleRatio: 1,
	}

	shutdown, err := Setup(context.Background(), cfg)
	require.NoError(t, err)

	provider, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	require.True(t, ok, "an enabled setup must install the SDK provider, got %T", otel.GetTracerProvider())

	scoped := provider.Tracer(InstrumentationScope)
	assert.NotNil(t, scoped)

	// Shutdown flushes and stops the batch processor; it must be idempotent-safe.
	assert.NoError(t, shutdown(context.Background()))
}

func TestSetupRegistersW3CPropagator(t *testing.T) {
	previous := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	_, err := Setup(context.Background(), Config{Enabled: false})
	require.NoError(t, err)

	// Even a disabled setup must wire propagation, so an inbound traceparent is
	// still forwarded downstream instead of being dropped.
	assert.NotEmpty(t, otel.GetTextMapPropagator().Fields(),
		"propagators must be installed so trace context is not silently lost")
}

func TestSampleRatioClamping(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  float64
	}{
		{name: "zero becomes full", input: 0, want: 1},
		{name: "negative becomes full", input: -0.5, want: 1},
		{name: "above one becomes full", input: 1.5, want: 1},
		{name: "half stays", input: 0.5, want: 0.5},
		{name: "tiny stays", input: 0.001, want: 0.001},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.InDelta(t, tc.want, sampleRatio(tc.input), 1e-9)
		})
	}
}

func TestBuildResourceCarriesServiceIdentity(t *testing.T) {
	res, err := buildResource(context.Background(), Config{
		ServiceName: "identity-svc",
		Version:     "v1.2.3",
		Environment: "staging",
	})
	require.NoError(t, err)

	got := map[string]string{}
	for _, attr := range res.Attributes() {
		got[string(attr.Key)] = attr.Value.Emit()
	}

	assert.Equal(t, "identity-svc", got["service.name"])
	assert.Equal(t, "v1.2.3", got["service.version"])
	assert.Equal(t, "staging", got["deployment.environment.name"])
}

func TestBuildResourceOmitsEmptyOptionalAttributes(t *testing.T) {
	res, err := buildResource(context.Background(), Config{ServiceName: "minimal"})
	require.NoError(t, err)

	got := map[string]string{}
	for _, attr := range res.Attributes() {
		got[string(attr.Key)] = attr.Value.Emit()
	}

	assert.Equal(t, "minimal", got["service.name"])
	assert.NotContains(t, got, "service.version", "an unset version must not publish an empty label")
	assert.NotContains(t, got, "deployment.environment.name")
}
