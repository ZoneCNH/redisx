package redisx

import (
	"context"

	"github.com/ZoneCNH/redisx/internal/provider"
)

type Option func(*options)

// Options is the public redisx client options contract used by schema and
// binder integrations. It keeps the validated Config together with optional
// Metrics and Provider overrides without changing New's functional-option API.
type Options struct {
	Config   Config
	Metrics  Metrics
	Provider Provider
}

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

// Validate checks the bound public configuration.
func (o Options) Validate() error {
	return o.Config.Validate()
}

// NewWithOptions constructs a client from the public options binder.
func NewWithOptions(ctx context.Context, opts Options) (*Client, error) {
	return New(ctx, opts.Config, opts.clientOptions()...)
}

func (o Options) clientOptions() []Option {
	clientOptions := make([]Option, 0, 2)
	if o.Metrics != nil {
		clientOptions = append(clientOptions, WithMetrics(o.Metrics))
	}
	if o.Provider != nil {
		clientOptions = append(clientOptions, WithProvider(o.Provider))
	}
	return clientOptions
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
