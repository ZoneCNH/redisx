package redisx

import (
	"context"
	"testing"
	"time"
)

type recordingMetrics struct {
	counters map[string]map[string]string
}

func (m *recordingMetrics) IncCounter(name string, labels map[string]string) {
	if m.counters == nil {
		m.counters = make(map[string]map[string]string)
	}
	m.counters[name] = labels
}

func (m *recordingMetrics) ObserveHistogram(string, float64, map[string]string) {}
func (m *recordingMetrics) SetGauge(string, float64, map[string]string)          {}

func TestOptionsValidateAndBindToConfig(t *testing.T) {
	opts := Options{
		Name:           "cache",
		Address:        "redis.example:6379",
		Username:       "default",
		Password:       "secret",
		DB:             2,
		TLS:            true,
		ConnectTimeout: 500 * time.Millisecond,
		ReadTimeout:    time.Second,
		WriteTimeout:   time.Second,
		PoolSize:       8,
	}
	if err := opts.Validate(); err != nil {
		t.Fatalf("Options.Validate() unexpected error: %v", err)
	}
	cfg := opts.ToConfig()
	if cfg.Name != "cache" || cfg.Timeout != 500*time.Millisecond || cfg.Secret != "secret" {
		t.Fatalf("Options.ToConfig() = %#v", cfg)
	}
	if got := opts.Sanitize().Password; got == "secret" || got == "" {
		t.Fatalf("Options.Sanitize().Password = %q; want redacted non-empty secret", got)
	}
}

func TestOptionsValidateRejectsNegativeFields(t *testing.T) {
	cases := []Options{
		{DB: -1},
		{ConnectTimeout: -time.Nanosecond},
		{ReadTimeout: -time.Nanosecond},
		{WriteTimeout: -time.Nanosecond},
		{PoolSize: -1},
	}
	for _, opts := range cases {
		if err := opts.Validate(); err == nil {
			t.Fatalf("Options.Validate(%#v) expected error", opts)
		}
	}
}

func TestDefaultOptionsUseInMemoryProviderAndDoNotDialRedis(t *testing.T) {
	ctx := context.Background()
	client, err := New(ctx, Config{Name: "default-memory"})
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	defer client.Close(ctx)

	if err := client.Set(ctx, "k", "v", 0); err != nil {
		t.Fatalf("Set() unexpected error: %v", err)
	}
	got, err := client.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}
	if got != "v" {
		t.Fatalf("Get() = %q; want v", got)
	}
}

func TestFunctionalOptionsIgnoreNilAndUseInjectedValues(t *testing.T) {
	ctx := context.Background()
	metrics := &recordingMetrics{}
	provider := defaultOptions().provider
	client, err := New(ctx, Config{Name: "injected"}, WithMetrics(nil), WithProvider(nil), WithMetrics(metrics), WithProvider(provider))
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	defer client.Close(ctx)

	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping() unexpected error: %v", err)
	}
	if labels := metrics.counters[MetricRedisOperationsTotal]; labels["op"] != "ping" {
		t.Fatalf("injected metrics did not record redis operation labels: %#v", labels)
	}
}
