package redisx

import (
	"context"
	"strings"
	"time"
)

// CacheClient provides cache-aside helpers on top of Redis string values.
type CacheClient[T any] struct {
	client *Client
	codec  Codec[T]
}

func NewCacheClient[T any](client *Client, codec Codec[T]) (*CacheClient[T], error) {
	const op = "redisx.NewCacheClient"
	if client == nil {
		return nil, validationError(op, "client is nil", nil)
	}
	if codec == nil {
		return nil, validationError(op, "codec is required", nil)
	}
	return &CacheClient[T]{client: client, codec: codec}, nil
}

func (c *CacheClient[T]) Get(ctx context.Context, key string) (T, error) {
	const op = "redisx.CacheClient.Get"
	var zero T
	if c == nil {
		return zero, validationError(op, "cache client is nil", nil)
	}
	raw, err := c.client.Get(ctx, key)
	if err != nil {
		return zero, err
	}
	value, err := c.codec.Unmarshal(raw)
	if err != nil {
		return zero, validationError(op, "decode cached value failed", err)
	}
	return value, nil
}

func (c *CacheClient[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	const op = "redisx.CacheClient.Set"
	if c == nil {
		return validationError(op, "cache client is nil", nil)
	}
	raw, err := c.codec.Marshal(value)
	if err != nil {
		return validationError(op, "encode cached value failed", err)
	}
	return c.client.Set(ctx, key, raw, ttl)
}

func (c *CacheClient[T]) GetOrLoad(ctx context.Context, key string, ttl time.Duration, loader func(context.Context) (T, error)) (T, error) {
	const op = "redisx.CacheClient.GetOrLoad"
	var zero T
	if c == nil {
		return zero, validationError(op, "cache client is nil", nil)
	}
	if loader == nil {
		return zero, validationError(op, "loader is required", nil)
	}
	value, err := c.Get(ctx, key)
	if err == nil {
		return value, nil
	}
	if !IsKind(err, ErrorKindNil) {
		return zero, err
	}
	loaded, err := loader(ctx)
	if err != nil {
		return zero, err
	}
	if err := c.Set(ctx, key, loaded, ttl); err != nil {
		return zero, err
	}
	return loaded, nil
}

// KeyBuilder builds Redis keys with an optional prefix and separator.
type KeyBuilder struct {
	Prefix    string
	Separator string
}

func NewKeyBuilder(prefix string) KeyBuilder {
	return KeyBuilder{
		Prefix:    strings.Trim(prefix, ":"),
		Separator: ":",
	}
}

func (b KeyBuilder) Build(parts ...string) string {
	separator := b.Separator
	if separator == "" {
		separator = ":"
	}

	clean := make([]string, 0, len(parts)+1)
	if prefix := strings.Trim(b.Prefix, ":"); prefix != "" {
		clean = append(clean, prefix)
	}
	for _, part := range parts {
		trimmed := strings.Trim(part, ":")
		if trimmed == "" {
			continue
		}
		clean = append(clean, trimmed)
	}
	return strings.Join(clean, separator)
}
