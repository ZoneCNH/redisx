package goredis

import (
	"context"
	"errors"
	"fmt"
	"net"
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
	return ttl, mapError(err)
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
			return fmt.Errorf("%w: %v", provider.ErrTimeout, err)
		}
		return fmt.Errorf("%w: %v", provider.ErrNetwork, err)
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "noauth"),
		strings.Contains(message, "wrongpass"),
		strings.Contains(message, "invalid username-password pair"),
		strings.Contains(message, "auth"):
		return fmt.Errorf("%w: %v", provider.ErrAuth, err)
	case strings.Contains(message, "readonly"):
		return fmt.Errorf("%w: %v", provider.ErrReadOnly, err)
	case strings.Contains(message, "misconf"),
		strings.Contains(message, "stop-writes-on-bgsave-error"):
		return fmt.Errorf("%w: %v", provider.ErrReadOnly, err)
	case strings.Contains(message, "loading"):
		return fmt.Errorf("%w: %v", provider.ErrLoading, err)
	case strings.Contains(message, "tryagain"):
		return fmt.Errorf("%w: %v", provider.ErrTryAgain, err)
	case strings.HasPrefix(message, "moved") || strings.Contains(message, " moved "):
		return fmt.Errorf("%w: %v", provider.ErrClusterMoved, err)
	case strings.HasPrefix(message, "ask") || strings.Contains(message, " ask "):
		return fmt.Errorf("%w: %v", provider.ErrClusterAsk, err)
	case strings.Contains(message, "timeout"),
		strings.Contains(message, "i/o timeout"),
		strings.Contains(message, "deadline exceeded"):
		return fmt.Errorf("%w: %v", provider.ErrTimeout, err)
	case strings.Contains(message, "dial tcp"),
		strings.Contains(message, "connection refused"),
		strings.Contains(message, "no route"),
		strings.Contains(message, "connection reset"),
		strings.Contains(message, "broken pipe"),
		strings.Contains(message, "network"):
		return fmt.Errorf("%w: %v", provider.ErrNetwork, err)
	default:
		return err
	}
}
