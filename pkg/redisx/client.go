package redisx

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	internalprovider "github.com/ZoneCNH/redisx/internal/provider"
)

type Client struct {
	cfg         Config
	metrics     Metrics
	provider    Provider
	mu          sync.Mutex
	initialized bool
	closed      bool
}

func New(ctx context.Context, cfg Config, opts ...Option) (*Client, error) {
	const op = "redisx.New"
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	if ctx == nil {
		err := validationError(op, "context is required", nil)
		recordErrorMetric(options.metrics, "new", err)
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		wrapped := contextError(op, err)
		recordErrorMetric(options.metrics, "new", wrapped)
		return nil, wrapped
	}
	if err := cfg.Validate(); err != nil {
		recordErrorMetric(options.metrics, "new", err)
		return nil, err
	}

	options.metrics.IncCounter(MetricClientCreatedTotal, map[string]string{"name": cfg.Name})
	return &Client{cfg: cfg, metrics: options.metrics, provider: options.provider, initialized: true}, nil
}

func (c *Client) Close(ctx context.Context) error {
	const op = "redisx.Close"
	if c == nil {
		return validationError(op, "client is nil", nil)
	}
	if ctx == nil {
		err := validationError(op, "context is required", nil)
		recordErrorMetric(c.metrics, "close", err)
		return err
	}
	if err := ctx.Err(); err != nil {
		wrapped := contextError(op, err)
		recordErrorMetric(c.metrics, "close", wrapped)
		return wrapped
	}

	c.mu.Lock()
	if !c.initialized {
		c.mu.Unlock()
		err := validationError(op, "client is not initialized", nil)
		recordErrorMetric(c.metrics, "close", err)
		return err
	}
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	name := c.cfg.Name
	metrics := c.metrics
	provider := c.provider
	c.mu.Unlock()

	if provider != nil {
		if err := provider.Close(ctx); err != nil {
			wrapped := providerError(op, err)
			recordErrorMetric(metrics, "close", wrapped)
			return wrapped
		}
	}
	if metrics != nil {
		metrics.IncCounter(MetricClientClosedTotal, map[string]string{"name": name})
	}
	return nil
}

func recordErrorMetric(metrics Metrics, op string, err error) {
	if metrics == nil {
		return
	}
	labels := map[string]string{
		"op":   op,
		"kind": string(errorKind(err)),
	}
	metrics.IncCounter(MetricClientErrorsTotal, labels)
	metrics.IncCounter(MetricRedisErrorsTotal, labels)
}

func (c *Client) Ping(ctx context.Context) error {
	const op = "redisx.Ping"
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, "ping", err)
		return err
	}
	if err := provider.Ping(ctx); err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, "ping", wrapped)
		return wrapped
	}
	recordOperationMetric(metrics, "ping")
	return nil
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	const op = "redisx.Get"
	if err := validateKey(op, key); err != nil {
		return "", err
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, "get", err)
		return "", err
	}
	value, err := provider.Get(ctx, key)
	if err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, "get", wrapped)
		return "", wrapped
	}
	recordOperationMetric(metrics, "get")
	return value, nil
}

func (c *Client) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	const op = "redisx.Set"
	if err := validateKey(op, key); err != nil {
		return err
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, "set", err)
		return err
	}
	if err := provider.Set(ctx, key, value, ttl); err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, "set", wrapped)
		return wrapped
	}
	recordOperationMetric(metrics, "set")
	return nil
}

func (c *Client) Del(ctx context.Context, keys ...string) (int64, error) {
	return c.keyCountOperation(ctx, "redisx.Del", "del", keys, func(provider Provider) (int64, error) { return provider.Del(ctx, keys...) })
}

func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.keyCountOperation(ctx, "redisx.Exists", "exists", keys, func(provider Provider) (int64, error) { return provider.Exists(ctx, keys...) })
}

func (c *Client) Expire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	const op = "redisx.Expire"
	if err := validateKey(op, key); err != nil {
		return false, err
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, "expire", err)
		return false, err
	}
	updated, err := provider.Expire(ctx, key, ttl)
	if err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, "expire", wrapped)
		return false, wrapped
	}
	recordOperationMetric(metrics, "expire")
	return updated, nil
}

func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	const op = "redisx.TTL"
	if err := validateKey(op, key); err != nil {
		return 0, err
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, "ttl", err)
		return 0, err
	}
	ttl, err := provider.TTL(ctx, key)
	if err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, "ttl", wrapped)
		return 0, wrapped
	}
	recordOperationMetric(metrics, "ttl")
	return ttl, nil
}

func (c *Client) MGet(ctx context.Context, keys ...string) ([]Value, error) {
	const op = "redisx.MGet"
	if err := validateKeys(op, keys); err != nil {
		return nil, err
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, "mget", err)
		return nil, err
	}
	values, err := provider.MGet(ctx, keys...)
	if err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, "mget", wrapped)
		return nil, wrapped
	}
	recordOperationMetric(metrics, "mget")
	return values, nil
}

func (c *Client) MSet(ctx context.Context, values map[string]string) error {
	const op = "redisx.MSet"
	if len(values) == 0 {
		return validationError(op, "at least one key is required", nil)
	}
	for key := range values {
		if err := validateKey(op, key); err != nil {
			return err
		}
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, "mset", err)
		return err
	}
	if err := provider.MSet(ctx, values); err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, "mset", wrapped)
		return wrapped
	}
	recordOperationMetric(metrics, "mset")
	return nil
}

func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	const op = "redisx.Incr"
	return c.intOperation(ctx, op, "incr", key, func(provider Provider) (int64, error) { return provider.Incr(ctx, key) })
}

func (c *Client) Decr(ctx context.Context, key string) (int64, error) {
	const op = "redisx.Decr"
	return c.intOperation(ctx, op, "decr", key, func(provider Provider) (int64, error) { return provider.Decr(ctx, key) })
}

func (c *Client) keyCountOperation(ctx context.Context, op string, metricOp string, keys []string, fn func(Provider) (int64, error)) (int64, error) {
	if err := validateKeys(op, keys); err != nil {
		return 0, err
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, metricOp, err)
		return 0, err
	}
	count, err := fn(provider)
	if err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, metricOp, wrapped)
		return 0, wrapped
	}
	recordOperationMetric(metrics, metricOp)
	return count, nil
}

func (c *Client) intOperation(ctx context.Context, op string, metricOp string, key string, fn func(Provider) (int64, error)) (int64, error) {
	if err := validateKey(op, key); err != nil {
		return 0, err
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, metricOp, err)
		return 0, err
	}
	value, err := fn(provider)
	if err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, metricOp, wrapped)
		return 0, wrapped
	}
	recordOperationMetric(metrics, metricOp)
	return value, nil
}

func (c *Client) providerForOperation(ctx context.Context, op string) (Provider, Metrics, error) {
	if c == nil {
		return nil, nil, validationError(op, "client is nil", nil)
	}
	metrics := c.metrics
	if ctx == nil {
		return nil, metrics, validationError(op, "context is required", nil)
	}
	if err := ctx.Err(); err != nil {
		return nil, metrics, contextError(op, err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.initialized {
		return nil, metrics, validationError(op, "client is not initialized", nil)
	}
	if c.closed {
		return nil, metrics, newError(ErrorKindClosed, op, "client is closed", false, nil)
	}
	if c.provider == nil {
		return nil, metrics, newError(ErrorKindProvider, op, "provider is required", false, nil)
	}
	return c.provider, metrics, nil
}

func validateKey(op string, key string) error {
	if key == "" {
		return validationError(op, "key is required", nil)
	}
	return nil
}

func validateKeys(op string, keys []string) error {
	if len(keys) == 0 {
		return validationError(op, "at least one key is required", nil)
	}
	for _, key := range keys {
		if err := validateKey(op, key); err != nil {
			return err
		}
	}
	return nil
}

func providerError(op string, cause error) *Error {
	if errors.Is(cause, internalprovider.ErrNil) || errors.Is(cause, ErrNil) {
		return newError(ErrorKindNil, op, cause.Error(), false, cause)
	}
	if errors.Is(cause, internalprovider.ErrClosed) || errors.Is(cause, ErrConnectionClosed) {
		return newError(ErrorKindClosed, op, cause.Error(), false, cause)
	}
	if errors.Is(cause, context.DeadlineExceeded) || errors.Is(cause, context.Canceled) {
		return contextError(op, cause)
	}
	if errors.Is(cause, ErrTimeout) {
		return newError(ErrorKindTimeout, op, cause.Error(), true, cause)
	}
	if errors.Is(cause, ErrCanceled) {
		return newError(ErrorKindCanceled, op, cause.Error(), false, cause)
	}
	if errors.Is(cause, ErrNetwork) {
		return newError(ErrorKindNetwork, op, cause.Error(), true, cause)
	}
	if errors.Is(cause, ErrAuth) {
		return newError(ErrorKindAuth, op, cause.Error(), false, cause)
	}
	if errors.Is(cause, ErrReadOnly) {
		return newError(ErrorKindReadOnly, op, cause.Error(), true, cause)
	}
	if errors.Is(cause, ErrLoading) {
		return newError(ErrorKindLoading, op, cause.Error(), true, cause)
	}
	if errors.Is(cause, ErrTryAgain) {
		return newError(ErrorKindTryAgain, op, cause.Error(), true, cause)
	}
	if errors.Is(cause, ErrClusterMoved) {
		return newError(ErrorKindClusterMoved, op, cause.Error(), true, cause)
	}
	if errors.Is(cause, ErrClusterAsk) {
		return newError(ErrorKindClusterAsk, op, cause.Error(), true, cause)
	}
	if errors.Is(cause, ErrInvalidConfig) {
		return newError(ErrorKindInvalidConfig, op, cause.Error(), false, cause)
	}
	if errors.Is(cause, ErrProvider) {
		return newError(ErrorKindProvider, op, cause.Error(), false, cause)
	}
	var parseErr *strconv.NumError
	if errors.As(cause, &parseErr) {
		return newError(ErrorKindValidation, op, "value is not an integer", false, cause)
	}
	return newError(ErrorKindProvider, op, cause.Error(), false, cause)
}

func recordOperationMetric(metrics Metrics, op string) {
	if metrics == nil {
		return
	}
	metrics.IncCounter(MetricRedisOperationsTotal, map[string]string{"op": op})
}
