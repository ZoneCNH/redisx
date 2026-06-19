package redisx

import (
	"context"

	"github.com/ZoneCNH/redisx/internal/provider"
	redisprovider "github.com/ZoneCNH/redisx/internal/provider/goredis"
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
	metrics     Metrics
	provider    Provider
	providerSet bool
}

var newRedisProvider = func(cfg redisprovider.Config) (Provider, error) {
	return redisprovider.New(cfg)
}

func defaultOptions() options {
	return options{
		metrics: NoopMetrics{},
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

func (o options) providerForConfig(cfg Config) (Provider, error) {
	if o.providerSet {
		return o.provider, nil
	}
	if cfg.Redis.Enabled() {
		return newRedisProvider(redisprovider.Config{
			Addr:         cfg.Redis.Addr,
			Username:     cfg.Redis.Username,
			Password:     cfg.Redis.Password,
			DB:           cfg.Redis.DB,
			DialTimeout:  cfg.Redis.DialTimeout,
			ReadTimeout:  cfg.Redis.ReadTimeout,
			WriteTimeout: cfg.Redis.WriteTimeout,
			PoolSize:     cfg.Redis.PoolSize,
			MinIdleConns: cfg.Redis.MinIdleConns,
			MaxRetries:   cfg.Redis.MaxRetries,
		})
	}
	return provider.NewMemory(), nil
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
			o.providerSet = true
		}
	}
}
