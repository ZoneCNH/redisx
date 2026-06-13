package goredis

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/ZoneCNH/redisx/internal/provider"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Addr         string
	Username     string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
}

type Provider struct {
	client *redis.Client
}

var _ provider.Provider = (*Provider)(nil)

const (
	redisTTLNoExpire = -time.Nanosecond
	redisTTLMissing  = -2 * time.Nanosecond
)

var (
	releaseLockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)
	fixedWindowRateLimitScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
	redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
local ttl = redis.call("PTTL", KEYS[1])
return {current, ttl}
`)
)

func New(cfg Config) (*Provider, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Provider{client: redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Username:     cfg.Username,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
	})}, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Addr) == "" {
		return errors.New("redis addr is required")
	}
	checks := []struct {
		name string
		bad  bool
	}{
		{name: "redis db must not be negative", bad: c.DB < 0},
		{name: "redis dial timeout must not be negative", bad: c.DialTimeout < 0},
		{name: "redis read timeout must not be negative", bad: c.ReadTimeout < 0},
		{name: "redis write timeout must not be negative", bad: c.WriteTimeout < 0},
		{name: "redis pool size must not be negative", bad: c.PoolSize < 0},
		{name: "redis min idle conns must not be negative", bad: c.MinIdleConns < 0},
		{name: "redis max retries must not be negative", bad: c.MaxRetries < 0},
	}
	for _, check := range checks {
		if check.bad {
			return errors.New(check.name)
		}
	}
	return nil
}

func (p *Provider) Ping(ctx context.Context) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	client, err := p.redisClient()
	if err != nil {
		return err
	}
	if err := client.Ping(ctx).Err(); err != nil {
		return mapError(err)
	}
	return nil
}

func (p *Provider) Close(ctx context.Context) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	client, err := p.redisClient()
	if err != nil {
		return err
	}
	if err := mapError(client.Close()); err != nil {
		return err
	}
	p.client = nil
	return nil
}

func (p *Provider) Get(ctx context.Context, key string) (string, error) {
	if err := contextError(ctx); err != nil {
		return "", err
	}
	client, err := p.redisClient()
	if err != nil {
		return "", err
	}
	value, err := client.Get(ctx, key).Result()
	return value, mapError(err)
}

func (p *Provider) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	client, err := p.redisClient()
	if err != nil {
		return err
	}
	return mapError(client.Set(ctx, key, value, ttl).Err())
}

func (p *Provider) SetNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error) {
	if err := contextError(ctx); err != nil {
		return false, err
	}
	client, err := p.redisClient()
	if err != nil {
		return false, err
	}
	set, err := client.SetNX(ctx, key, value, ttl).Result()
	return set, mapError(err)
}

func (p *Provider) Del(ctx context.Context, keys ...string) (int64, error) {
	if err := contextError(ctx); err != nil {
		return 0, err
	}
	client, err := p.redisClient()
	if err != nil {
		return 0, err
	}
	count, err := client.Del(ctx, keys...).Result()
	return count, mapError(err)
}

func (p *Provider) Exists(ctx context.Context, keys ...string) (int64, error) {
	if err := contextError(ctx); err != nil {
		return 0, err
	}
	client, err := p.redisClient()
	if err != nil {
		return 0, err
	}
	count, err := client.Exists(ctx, keys...).Result()
	return count, mapError(err)
}

func (p *Provider) Expire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if err := contextError(ctx); err != nil {
		return false, err
	}
	client, err := p.redisClient()
	if err != nil {
		return false, err
	}
	updated, err := client.Expire(ctx, key, ttl).Result()
	return updated, mapError(err)
}

func (p *Provider) TTL(ctx context.Context, key string) (time.Duration, error) {
	if err := contextError(ctx); err != nil {
		return 0, err
	}
	client, err := p.redisClient()
	if err != nil {
		return 0, err
	}
	ttl, err := client.TTL(ctx, key).Result()
	if err := mapError(err); err != nil {
		return 0, err
	}
	return normalizeTTL(ttl), nil
}

func (p *Provider) MGet(ctx context.Context, keys ...string) ([]provider.Value, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	client, err := p.redisClient()
	if err != nil {
		return nil, err
	}
	rawValues, err := client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, mapError(err)
	}
	values := make([]provider.Value, len(rawValues))
	for i, raw := range rawValues {
		if raw == nil {
			continue
		}
		switch value := raw.(type) {
		case string:
			values[i] = provider.Value{Value: value, Found: true}
		case []byte:
			values[i] = provider.Value{Value: string(value), Found: true}
		default:
			values[i] = provider.Value{Value: fmt.Sprint(value), Found: true}
		}
	}
	return values, nil
}

func (p *Provider) MSet(ctx context.Context, values map[string]string) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	client, err := p.redisClient()
	if err != nil {
		return err
	}
	args := make([]any, 0, len(values)*2)
	for key, value := range values {
		args = append(args, key, value)
	}
	return mapError(client.MSet(ctx, args...).Err())
}

func (p *Provider) Incr(ctx context.Context, key string) (int64, error) {
	if err := contextError(ctx); err != nil {
		return 0, err
	}
	client, err := p.redisClient()
	if err != nil {
		return 0, err
	}
	value, err := client.Incr(ctx, key).Result()
	return value, mapError(err)
}

func (p *Provider) Decr(ctx context.Context, key string) (int64, error) {
	if err := contextError(ctx); err != nil {
		return 0, err
	}
	client, err := p.redisClient()
	if err != nil {
		return 0, err
	}
	value, err := client.Decr(ctx, key).Result()
	return value, mapError(err)
}

func (p *Provider) HSet(ctx context.Context, key string, values map[string]string) (int64, error) {
	if err := contextError(ctx); err != nil {
		return 0, err
	}
	client, err := p.redisClient()
	if err != nil {
		return 0, err
	}
	count, err := client.HSet(ctx, key, stringMapToAny(values)...).Result()
	return count, mapError(err)
}

func (p *Provider) HGet(ctx context.Context, key string, field string) (string, error) {
	if err := contextError(ctx); err != nil {
		return "", err
	}
	client, err := p.redisClient()
	if err != nil {
		return "", err
	}
	value, err := client.HGet(ctx, key, field).Result()
	return value, mapError(err)
}

func (p *Provider) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	client, err := p.redisClient()
	if err != nil {
		return nil, err
	}
	values, err := client.HGetAll(ctx, key).Result()
	return values, mapError(err)
}

func (p *Provider) HDel(ctx context.Context, key string, fields ...string) (int64, error) {
	if err := contextError(ctx); err != nil {
		return 0, err
	}
	client, err := p.redisClient()
	if err != nil {
		return 0, err
	}
	count, err := client.HDel(ctx, key, fields...).Result()
	return count, mapError(err)
}

func (p *Provider) LPush(ctx context.Context, key string, values ...string) (int64, error) {
	if err := contextError(ctx); err != nil {
		return 0, err
	}
	client, err := p.redisClient()
	if err != nil {
		return 0, err
	}
	count, err := client.LPush(ctx, key, stringArgs(values)...).Result()
	return count, mapError(err)
}

func (p *Provider) RPush(ctx context.Context, key string, values ...string) (int64, error) {
	if err := contextError(ctx); err != nil {
		return 0, err
	}
	client, err := p.redisClient()
	if err != nil {
		return 0, err
	}
	count, err := client.RPush(ctx, key, stringArgs(values)...).Result()
	return count, mapError(err)
}

func (p *Provider) LRange(ctx context.Context, key string, start int64, stop int64) ([]string, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	client, err := p.redisClient()
	if err != nil {
		return nil, err
	}
	values, err := client.LRange(ctx, key, start, stop).Result()
	return values, mapError(err)
}

func (p *Provider) LLen(ctx context.Context, key string) (int64, error) {
	if err := contextError(ctx); err != nil {
		return 0, err
	}
	client, err := p.redisClient()
	if err != nil {
		return 0, err
	}
	length, err := client.LLen(ctx, key).Result()
	return length, mapError(err)
}

func (p *Provider) LPop(ctx context.Context, key string) (string, error) {
	if err := contextError(ctx); err != nil {
		return "", err
	}
	client, err := p.redisClient()
	if err != nil {
		return "", err
	}
	value, err := client.LPop(ctx, key).Result()
	return value, mapError(err)
}

func (p *Provider) RPop(ctx context.Context, key string) (string, error) {
	if err := contextError(ctx); err != nil {
		return "", err
	}
	client, err := p.redisClient()
	if err != nil {
		return "", err
	}
	value, err := client.RPop(ctx, key).Result()
	return value, mapError(err)
}

func (p *Provider) Pipeline(ctx context.Context, commands []provider.PipelineCommand) ([]provider.PipelineResult, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	client, err := p.redisClient()
	if err != nil {
		return nil, err
	}

	pipe := client.Pipeline()
	results := make([]provider.PipelineResult, len(commands))
	queued := make([]redis.Cmder, len(commands))
	for i, command := range commands {
		commandType := provider.PipelineCommandKind(command)
		results[i] = provider.PipelineResult{Type: commandType, Key: command.Key}
		switch commandType {
		case provider.PipelineSet:
			queued[i] = pipe.Set(ctx, command.Key, command.Value, command.TTL)
		case provider.PipelineMSet:
			queued[i] = pipe.MSet(ctx, mapArgs(command.Values)...)
			results[i].Bool = true
		case provider.PipelineGet:
			queued[i] = pipe.Get(ctx, command.Key)
		case provider.PipelineHSet:
			queued[i] = pipe.HSet(ctx, command.Key, stringMapToAny(command.Values)...)
		case provider.PipelineHGet:
			queued[i] = pipe.HGet(ctx, command.Key, command.Field)
		case provider.PipelineRPush:
			queued[i] = pipe.RPush(ctx, command.Key, stringArgs(provider.PipelineCommandListValues(command))...)
		case provider.PipelineLRange:
			queued[i] = pipe.LRange(ctx, command.Key, command.Start, command.Stop)
		case provider.PipelineIncr:
			queued[i] = pipe.Incr(ctx, command.Key)
		default:
			return nil, fmt.Errorf("unsupported pipeline command type %q", commandType)
		}
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, mapError(err)
	}
	for i, command := range commands {
		commandType := provider.PipelineCommandKind(command)
		switch commandType {
		case provider.PipelineSet:
			if err := mapError(queued[i].Err()); err != nil {
				return nil, err
			}
			results[i].Bool = true
		case provider.PipelineMSet:
			if err := mapError(queued[i].Err()); err != nil {
				return nil, err
			}
			results[i].Bool = true
		case provider.PipelineGet, provider.PipelineHGet:
			stringCmd, ok := queued[i].(*redis.StringCmd)
			if !ok {
				return nil, fmt.Errorf("unexpected pipeline result type for %q", commandType)
			}
			value, err := stringCmd.Result()
			if errors.Is(err, redis.Nil) {
				continue
			}
			if err != nil {
				return nil, mapError(err)
			}
			results[i].Found = true
			results[i].Value = value
			if commandType == provider.PipelineGet {
				results[i].Values = []provider.Value{{Value: value, Found: true}}
			}
		case provider.PipelineHSet, provider.PipelineRPush, provider.PipelineIncr:
			intCmd, ok := queued[i].(*redis.IntCmd)
			if !ok {
				return nil, fmt.Errorf("unexpected pipeline result type for %q", commandType)
			}
			value, err := intCmd.Result()
			if err != nil {
				return nil, mapError(err)
			}
			results[i].Int = value
			if commandType == provider.PipelineHSet || commandType == provider.PipelineRPush {
				results[i].Count = value
			}
		case provider.PipelineLRange:
			stringSliceCmd, ok := queued[i].(*redis.StringSliceCmd)
			if !ok {
				return nil, fmt.Errorf("unexpected pipeline result type for %q", commandType)
			}
			values, err := stringSliceCmd.Result()
			if err != nil {
				return nil, mapError(err)
			}
			results[i].Strings = values
		}
	}
	return results, nil
}

func (p *Provider) AcquireLock(ctx context.Context, key string, token string, ttl time.Duration) (bool, error) {
	if err := contextError(ctx); err != nil {
		return false, err
	}
	client, err := p.redisClient()
	if err != nil {
		return false, err
	}
	acquired, err := client.SetNX(ctx, key, token, ttl).Result()
	return acquired, mapError(err)
}

func (p *Provider) ReleaseLock(ctx context.Context, key string, token string) (bool, error) {
	if err := contextError(ctx); err != nil {
		return false, err
	}
	client, err := p.redisClient()
	if err != nil {
		return false, err
	}
	raw, err := releaseLockScript.Run(ctx, client, []string{key}, token).Result()
	if err != nil {
		return false, mapError(err)
	}
	deleted, err := int64Result(raw)
	if err != nil {
		return false, err
	}
	return deleted == 1, nil
}

func (p *Provider) FixedWindowRateLimit(ctx context.Context, key string, limit int64, window time.Duration) (provider.RateLimitResult, error) {
	result := provider.RateLimitResult{Limit: limit}
	if err := contextError(ctx); err != nil {
		return result, err
	}
	client, err := p.redisClient()
	if err != nil {
		return result, err
	}
	windowMS := window.Milliseconds()
	if windowMS < 1 {
		windowMS = 1
	}
	raw, err := fixedWindowRateLimitScript.Run(ctx, client, []string{key}, windowMS).Result()
	if err != nil {
		return result, mapError(err)
	}
	values, ok := raw.([]any)
	if !ok || len(values) != 2 {
		return result, fmt.Errorf("unexpected rate limit script result %T", raw)
	}
	count, err := int64Result(values[0])
	if err != nil {
		return result, err
	}
	ttlMS, err := int64Result(values[1])
	if err != nil {
		return result, err
	}
	if ttlMS < 0 {
		ttlMS = 0
	}
	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}
	result.Allowed = count <= limit
	result.Count = count
	result.Remaining = remaining
	result.ResetAfter = time.Duration(ttlMS) * time.Millisecond
	return result, nil
}

func stringMapToAny(values map[string]string) []any {
	args := make([]any, 0, len(values)*2)
	for field, value := range values {
		args = append(args, field, value)
	}
	return args
}

func (p *Provider) redisClient() (*redis.Client, error) {
	if p == nil || p.client == nil {
		return nil, provider.ErrClosed
	}
	return p.client, nil
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	return ctx.Err()
}

func normalizeTTL(ttl time.Duration) time.Duration {
	switch ttl {
	case redisTTLNoExpire:
		return -time.Second
	case redisTTLMissing:
		return -2 * time.Second
	default:
		return ttl
	}
}

func stringArgs(values []string) []any {
	args := make([]any, len(values))
	for i, value := range values {
		args[i] = value
	}
	return args
}

func mapArgs(values map[string]string) []any {
	args := make([]any, 0, len(values)*2)
	for key, value := range values {
		args = append(args, key, value)
	}
	return args
}

func int64Result(value any) (int64, error) {
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case int:
		return int64(typed), nil
	case string:
		return strconv.ParseInt(typed, 10, 64)
	case []byte:
		return strconv.ParseInt(string(typed), 10, 64)
	default:
		return 0, fmt.Errorf("unexpected integer result type %T", value)
	}
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, redis.Nil) {
		return provider.ErrNil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return fmt.Errorf("%w: %w", provider.ErrTimeout, err)
		}
		return fmt.Errorf("%w: %w", provider.ErrNetwork, err)
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "wrongtype"):
		return fmt.Errorf("%w: %w", provider.ErrWrongType, err)
	case strings.Contains(message, "noauth"),
		strings.Contains(message, "wrongpass"),
		strings.Contains(message, "invalid username-password pair"),
		strings.Contains(message, "auth"):
		return fmt.Errorf("%w: %w", provider.ErrAuth, err)
	case strings.Contains(message, "wrongtype"):
		return fmt.Errorf("%w: %w", provider.ErrWrongType, err)
	case strings.Contains(message, "not an integer"),
		strings.Contains(message, "out of range"):
		return fmt.Errorf("%w: %w", provider.ErrInvalidInt, err)
	case strings.Contains(message, "readonly"):
		return fmt.Errorf("%w: %w", provider.ErrReadOnly, err)
	case strings.Contains(message, "misconf"),
		strings.Contains(message, "stop-writes-on-bgsave-error"):
		return fmt.Errorf("%w: %w", provider.ErrReadOnly, err)
	case strings.Contains(message, "loading"):
		return fmt.Errorf("%w: %w", provider.ErrLoading, err)
	case strings.Contains(message, "tryagain"):
		return fmt.Errorf("%w: %w", provider.ErrTryAgain, err)
	case strings.HasPrefix(message, "moved") || strings.Contains(message, " moved "):
		return fmt.Errorf("%w: %w", provider.ErrClusterMoved, err)
	case strings.HasPrefix(message, "ask") || strings.Contains(message, " ask "):
		return fmt.Errorf("%w: %w", provider.ErrClusterAsk, err)
	case strings.Contains(message, "timeout"),
		strings.Contains(message, "i/o timeout"),
		strings.Contains(message, "deadline exceeded"):
		return fmt.Errorf("%w: %w", provider.ErrTimeout, err)
	case strings.Contains(message, "dial tcp"),
		strings.Contains(message, "connection refused"),
		strings.Contains(message, "no route"),
		strings.Contains(message, "connection reset"),
		strings.Contains(message, "broken pipe"),
		strings.Contains(message, "network"):
		return fmt.Errorf("%w: %w", provider.ErrNetwork, err)
	default:
		return err
	}
}
