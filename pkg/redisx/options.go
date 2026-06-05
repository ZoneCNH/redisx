package redisx

import "github.com/ZoneCNH/redisx/internal/provider"

type Option func(*options)

type options struct {
	metrics  Metrics
	provider Provider
}

func defaultOptions() options {
	return options{
		metrics:  NoopMetrics{},
		provider: provider.NewMemory(),
	}
}

func WithMetrics(metrics Metrics) Option {
	return func(o *options) {
		if metrics != nil {
			o.metrics = metrics
		}
	}
}

func WithProvider(provider Provider) Option {
	return func(o *options) {
		if provider != nil {
			o.provider = provider
		}
	}
}
