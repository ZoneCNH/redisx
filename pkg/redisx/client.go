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
	resolvedProvider, err := options.providerForConfig(cfg)
	if err != nil {
		wrapped := newError(ErrorKindInvalidConfig, op, err.Error(), false, err)
		recordErrorMetric(options.metrics, "new", wrapped)
		return nil, wrapped
	}

	options.metrics.IncCounter(MetricClientCreatedTotal, map[string]string{"name": cfg.Name})
	return &Client{cfg: cfg, metrics: options.metrics, provider: resolvedProvider, initialized: true}, nil
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

func (c *Client) SetNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error) {
	const op = "redisx.SetNX"
	if err := validateKey(op, key); err != nil {
		return false, err
	}
	if ttl < 0 {
		return false, validationError(op, "ttl must not be negative", nil)
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, "setnx", err)
		return false, err
	}
	set, err := provider.SetNX(ctx, key, value, ttl)
	if err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, "setnx", wrapped)
		return false, wrapped
	}
	recordOperationMetric(metrics, "setnx")
	return set, nil
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

func (c *Client) HSet(ctx context.Context, key string, values map[string]string) (int64, error) {
	const op = "redisx.HSet"
	if err := validateKey(op, key); err != nil {
		return 0, err
	}
	if err := validateHashValues(op, values); err != nil {
		return 0, err
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, "hset", err)
		return 0, err
	}
	added, err := provider.HSet(ctx, key, values)
	if err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, "hset", wrapped)
		return 0, wrapped
	}
	recordOperationMetric(metrics, "hset")
	return added, nil
}

func (c *Client) HGet(ctx context.Context, key string, field string) (string, error) {
	const op = "redisx.HGet"
	if err := validateKey(op, key); err != nil {
		return "", err
	}
	if err := validateField(op, field); err != nil {
		return "", err
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, "hget", err)
		return "", err
	}
	value, err := provider.HGet(ctx, key, field)
	if err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, "hget", wrapped)
		return "", wrapped
	}
	recordOperationMetric(metrics, "hget")
	return value, nil
}

func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	const op = "redisx.HGetAll"
	if err := validateKey(op, key); err != nil {
		return nil, err
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, "hgetall", err)
		return nil, err
	}
	values, err := provider.HGetAll(ctx, key)
	if err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, "hgetall", wrapped)
		return nil, wrapped
	}
	recordOperationMetric(metrics, "hgetall")
	return values, nil
}

func (c *Client) HDel(ctx context.Context, key string, fields ...string) (int64, error) {
	const op = "redisx.HDel"
	if err := validateKey(op, key); err != nil {
		return 0, err
	}
	if err := validateFields(op, fields); err != nil {
		return 0, err
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, "hdel", err)
		return 0, err
	}
	deleted, err := provider.HDel(ctx, key, fields...)
	if err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, "hdel", wrapped)
		return 0, wrapped
	}
	recordOperationMetric(metrics, "hdel")
	return deleted, nil
}

func (c *Client) LPush(ctx context.Context, key string, values ...string) (int64, error) {
	return c.listPushOperation(ctx, "redisx.LPush", "lpush", key, values, func(provider Provider) (int64, error) {
		return provider.LPush(ctx, key, values...)
	})
}

func (c *Client) RPush(ctx context.Context, key string, values ...string) (int64, error) {
	return c.listPushOperation(ctx, "redisx.RPush", "rpush", key, values, func(provider Provider) (int64, error) {
		return provider.RPush(ctx, key, values...)
	})
}

func (c *Client) LRange(ctx context.Context, key string, start int64, stop int64) ([]string, error) {
	const op = "redisx.LRange"
	if err := validateKey(op, key); err != nil {
		return nil, err
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, "lrange", err)
		return nil, err
	}
	values, err := provider.LRange(ctx, key, start, stop)
	if err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, "lrange", wrapped)
		return nil, wrapped
	}
	recordOperationMetric(metrics, "lrange")
	return values, nil
}

func (c *Client) LLen(ctx context.Context, key string) (int64, error) {
	return c.intOperation(ctx, "redisx.LLen", "llen", key, func(provider Provider) (int64, error) {
		return provider.LLen(ctx, key)
	})
}

func (c *Client) LPop(ctx context.Context, key string) (string, error) {
	return c.listPopOperation(ctx, "redisx.LPop", "lpop", key, func(provider Provider) (string, error) {
		return provider.LPop(ctx, key)
	})
}

func (c *Client) RPop(ctx context.Context, key string) (string, error) {
	return c.listPopOperation(ctx, "redisx.RPop", "rpop", key, func(provider Provider) (string, error) {
		return provider.RPop(ctx, key)
	})
}

func (c *Client) Pipeline(ctx context.Context, first any, rest ...PipelineCommand) ([]PipelineResult, error) {
	const op = "redisx.Pipeline"
	commands, err := normalizePipelineCommands(op, first, rest)
	if err != nil {
		return nil, err
	}
	if err := validatePipelineCommands(op, commands); err != nil {
		return nil, err
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, "pipeline", err)
		return nil, err
	}
	results, err := provider.Pipeline(ctx, commands)
	if err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, "pipeline", wrapped)
		return nil, wrapped
	}
	recordOperationMetric(metrics, "pipeline")
	return results, nil
}

func (c *Client) AcquireLock(ctx context.Context, key string, token string, ttl time.Duration) (bool, error) {
	const op = "redisx.AcquireLock"
	if err := validateKey(op, key); err != nil {
		return false, err
	}
	if token == "" {
		return false, validationError(op, "token is required", nil)
	}
	if ttl <= 0 {
		return false, validationError(op, "ttl must be positive", nil)
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, "acquire_lock", err)
		return false, err
	}
	acquired, err := provider.AcquireLock(ctx, key, token, ttl)
	if err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, "acquire_lock", wrapped)
		return false, wrapped
	}
	recordOperationMetric(metrics, "acquire_lock")
	return acquired, nil
}

func (c *Client) ReleaseLock(ctx context.Context, key string, token string) (bool, error) {
	const op = "redisx.ReleaseLock"
	if err := validateKey(op, key); err != nil {
		return false, err
	}
	if token == "" {
		return false, validationError(op, "token is required", nil)
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, "release_lock", err)
		return false, err
	}
	released, err := provider.ReleaseLock(ctx, key, token)
	if err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, "release_lock", wrapped)
		return false, wrapped
	}
	recordOperationMetric(metrics, "release_lock")
	return released, nil
}

func (c *Client) FixedWindowRateLimit(ctx context.Context, key string, limit int64, window time.Duration) (RateLimitResult, error) {
	const op = "redisx.FixedWindowRateLimit"
	if err := validateKey(op, key); err != nil {
		return RateLimitResult{}, err
	}
	if limit <= 0 {
		return RateLimitResult{}, validationError(op, "limit must be positive", nil)
	}
	if window <= 0 {
		return RateLimitResult{}, validationError(op, "window must be positive", nil)
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, "fixed_window_rate_limit", err)
		return RateLimitResult{}, err
	}
	result, err := provider.FixedWindowRateLimit(ctx, key, limit, window)
	if err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, "fixed_window_rate_limit", wrapped)
		return RateLimitResult{}, wrapped
	}
	recordOperationMetric(metrics, "fixed_window_rate_limit")
	return result, nil
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

func (c *Client) listPushOperation(ctx context.Context, op string, metricOp string, key string, values []string, fn func(Provider) (int64, error)) (int64, error) {
	if err := validateKey(op, key); err != nil {
		return 0, err
	}
	if err := validateValues(op, values); err != nil {
		return 0, err
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, metricOp, err)
		return 0, err
	}
	length, err := fn(provider)
	if err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, metricOp, wrapped)
		return 0, wrapped
	}
	recordOperationMetric(metrics, metricOp)
	return length, nil
}

func (c *Client) listPopOperation(ctx context.Context, op string, metricOp string, key string, fn func(Provider) (string, error)) (string, error) {
	if err := validateKey(op, key); err != nil {
		return "", err
	}
	provider, metrics, err := c.providerForOperation(ctx, op)
	if err != nil {
		recordErrorMetric(metrics, metricOp, err)
		return "", err
	}
	value, err := fn(provider)
	if err != nil {
		wrapped := providerError(op, err)
		recordErrorMetric(metrics, metricOp, wrapped)
		return "", wrapped
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

func validateField(op string, field string) error {
	if field == "" {
		return validationError(op, "field is required", nil)
	}
	return nil
}

func validateFields(op string, fields []string) error {
	if len(fields) == 0 {
		return validationError(op, "at least one field is required", nil)
	}
	for _, field := range fields {
		if err := validateField(op, field); err != nil {
			return err
		}
	}
	return nil
}

func validateHashValues(op string, values map[string]string) error {
	if len(values) == 0 {
		return validationError(op, "at least one field is required", nil)
	}
	for field := range values {
		if err := validateField(op, field); err != nil {
			return err
		}
	}
	return nil
}

func validateValues(op string, values []string) error {
	if len(values) == 0 {
		return validationError(op, "at least one value is required", nil)
	}
	return nil
}

func normalizePipelineCommands(op string, first any, rest []PipelineCommand) ([]PipelineCommand, error) {
	switch commands := first.(type) {
	case []PipelineCommand:
		if len(rest) > 0 {
			return nil, validationError(op, "cannot mix pipeline command slice and variadic commands", nil)
		}
		return commands, nil
	case PipelineCommand:
		return append([]PipelineCommand{commands}, rest...), nil
	default:
		return nil, validationError(op, "commands must be pipeline commands", nil)
	}
}

func validatePipelineCommands(op string, commands []PipelineCommand) error {
	if len(commands) == 0 {
		return validationError(op, "at least one command is required", nil)
	}
	for _, command := range commands {
		commandType := internalprovider.PipelineCommandKind(command)
		switch commandType {
		case PipelineSet:
			if err := validateKey(op, command.Key); err != nil {
				return err
			}
			if command.TTL < 0 {
				return validationError(op, "ttl must not be negative", nil)
			}
		case PipelineMSet:
			if len(command.Values) == 0 {
				return validationError(op, "at least one key is required", nil)
			}
			for key := range command.Values {
				if err := validateKey(op, key); err != nil {
					return err
				}
			}
		case PipelineHSet:
			if err := validateKey(op, command.Key); err != nil {
				return err
			}
			if err := validateHashValues(op, command.Values); err != nil {
				return err
			}
		case PipelineGet:
			if err := validateKey(op, command.Key); err != nil {
				return err
			}
		case PipelineRPush:
			if err := validateKey(op, command.Key); err != nil {
				return err
			}
			if err := validateValues(op, internalprovider.PipelineCommandListValues(command)); err != nil {
				return err
			}
		case PipelineHGet:
			if err := validateKey(op, command.Key); err != nil {
				return err
			}
			if err := validateField(op, command.Field); err != nil {
				return err
			}
		case PipelineLRange:
			if err := validateKey(op, command.Key); err != nil {
				return err
			}
		case PipelineIncr:
			if err := validateKey(op, command.Key); err != nil {
				return err
			}
		default:
			return validationError(op, "unsupported pipeline command", nil)
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
	if errors.Is(cause, internalprovider.ErrTimeout) {
		return newError(ErrorKindTimeout, op, cause.Error(), true, cause)
	}
	if errors.Is(cause, internalprovider.ErrNetwork) {
		return newError(ErrorKindNetwork, op, cause.Error(), true, cause)
	}
	if errors.Is(cause, internalprovider.ErrAuth) {
		return newError(ErrorKindAuth, op, cause.Error(), false, cause)
	}
	if errors.Is(cause, internalprovider.ErrReadOnly) {
		return newError(ErrorKindReadOnly, op, cause.Error(), true, cause)
	}
	if errors.Is(cause, internalprovider.ErrLoading) {
		return newError(ErrorKindLoading, op, cause.Error(), true, cause)
	}
	if errors.Is(cause, internalprovider.ErrTryAgain) {
		return newError(ErrorKindTryAgain, op, cause.Error(), true, cause)
	}
	if errors.Is(cause, internalprovider.ErrClusterMoved) {
		return newError(ErrorKindClusterMoved, op, cause.Error(), true, cause)
	}
	if errors.Is(cause, internalprovider.ErrClusterAsk) {
		return newError(ErrorKindClusterAsk, op, cause.Error(), true, cause)
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
	if errors.Is(cause, internalprovider.ErrInvalidInt) {
		return newError(ErrorKindValidation, op, "value is not an integer", false, cause)
	}
	if errors.Is(cause, internalprovider.ErrWrongType) {
		return newError(ErrorKindValidation, op, "redis key contains a different data type", false, cause)
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
