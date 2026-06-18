package redisx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Cache is a small typed cache helper backed by Client string operations.
type Cache[T any] struct {
	Client *Client
	Codec  Codec[T]
	TTL    time.Duration
}

func (c Cache[T]) Get(ctx context.Context, key string) (T, bool, error) {
	const op = "redisx.Cache.Get"
	var zero T
	if c.Client == nil {
		return zero, false, validationError(op, "client is required", nil)
	}
	codec := c.codec()
	raw, err := c.Client.Get(ctx, key)
	if IsKind(err, ErrorKindNil) {
		return zero, false, nil
	}
	if err != nil {
		return zero, false, err
	}
	value, err := codec.Unmarshal(raw)
	if err != nil {
		return zero, false, validationError(op, "cached value is invalid", err)
	}
	return value, true, nil
}

func (c Cache[T]) Set(ctx context.Context, key string, value T) error {
	const op = "redisx.Cache.Set"
	if c.Client == nil {
		return validationError(op, "client is required", nil)
	}
	raw, err := c.codec().Marshal(value)
	if err != nil {
		return validationError(op, "value cannot be encoded", err)
	}
	return c.Client.Set(ctx, key, raw, c.TTL)
}

func (c Cache[T]) GetOrLoad(ctx context.Context, key string, load func(context.Context) (T, error)) (T, error) {
	const op = "redisx.Cache.GetOrLoad"
	var zero T
	if load == nil {
		return zero, validationError(op, "load function is required", nil)
	}
	value, found, err := c.Get(ctx, key)
	if err != nil || found {
		return value, err
	}
	value, err = load(ctx)
	if err != nil {
		return zero, err
	}
	if err := c.Set(ctx, key, value); err != nil {
		return zero, err
	}
	return value, nil
}

func (c Cache[T]) codec() Codec[T] {
	if c.Codec != nil {
		return c.Codec
	}
	return JSONCodec[T]{}
}

// Lock is a Redis lock acquired with SET NX and released by token comparison.
type Lock struct {
	client *Client
	key    string
	token  string
}

var readLockTokenRandom = rand.Read

func (c *Client) NewLock(ctx context.Context, key string, ttl time.Duration) (*Lock, bool, error) {
	const op = "redisx.NewLock"
	if ttl <= 0 {
		return nil, false, validationError(op, "ttl must be positive", nil)
	}
	token, err := newLockToken()
	if err != nil {
		return nil, false, newError(ErrorKindInternal, op, "lock token cannot be generated", true, err)
	}
	acquired, err := c.AcquireLock(ctx, key, token, ttl)
	if err != nil || !acquired {
		return nil, acquired, err
	}
	return &Lock{client: c, key: key, token: token}, true, nil
}

func (l *Lock) Release(ctx context.Context) (bool, error) {
	const op = "redisx.Lock.Release"
	if l == nil || l.client == nil {
		return false, validationError(op, "lock is required", nil)
	}
	return l.client.ReleaseLock(ctx, l.key, l.token)
}

func (l *Lock) Token() string {
	if l == nil {
		return ""
	}
	return l.token
}

func newLockToken() (string, error) {
	var bytes [16]byte
	if _, err := readLockTokenRandom(bytes[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes[:]), nil
}

func fixedWindowResetAfter(window time.Duration, now time.Time, bucket int64) time.Duration {
	resetAfter := window - now.Sub(time.Unix(0, bucket*int64(window)))
	if resetAfter < 0 {
		return 0
	}
	return resetAfter
}

type FixedWindowRateLimiter struct {
	Client *Client
	Prefix string
	Limit  int64
	Window time.Duration
	Clock  func() time.Time
}

func (r FixedWindowRateLimiter) Allow(ctx context.Context, subject string) (RateLimitResult, error) {
	const op = "redisx.FixedWindowRateLimiter.Allow"
	if r.Client == nil {
		return RateLimitResult{}, validationError(op, "client is required", nil)
	}
	if subject == "" {
		return RateLimitResult{}, validationError(op, "subject is required", nil)
	}
	if r.Limit <= 0 {
		return RateLimitResult{}, validationError(op, "limit must be positive", nil)
	}
	if r.Window <= 0 {
		return RateLimitResult{}, validationError(op, "window must be positive", nil)
	}
	clock := r.Clock
	if clock == nil {
		clock = time.Now
	}
	now := clock()
	bucket := now.UnixNano() / int64(r.Window)
	prefix := r.Prefix
	if prefix == "" {
		prefix = "redisx:rate"
	}
	key := NewKeyBuilder(prefix).Build(subject, time.Unix(0, bucket*int64(r.Window)).UTC().Format("20060102150405"))
	count, err := r.Client.Incr(ctx, key)
	if err != nil {
		return RateLimitResult{}, err
	}
	if count == 1 {
		if _, err := r.Client.Expire(ctx, key, r.Window); err != nil {
			return RateLimitResult{}, err
		}
	}
	remaining := r.Limit - count
	if remaining < 0 {
		remaining = 0
	}
	resetAfter := fixedWindowResetAfter(r.Window, now, bucket)
	return RateLimitResult{Allowed: count <= r.Limit, Limit: r.Limit, Remaining: remaining, ResetAfter: resetAfter, Count: count}, nil
}
