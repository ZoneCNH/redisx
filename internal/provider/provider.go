package provider

import (
	"context"
	"time"
)

// Value represents a string Redis value that may be absent in multi-key reads.
type Value struct {
	Value string `json:"value"`
	Found bool   `json:"found"`
}

// Provider is the storage boundary implemented by Redis adapters.
type Provider interface {
	Ping(context.Context) error
	Close(context.Context) error
	Get(context.Context, string) (string, error)
	Set(context.Context, string, string, time.Duration) error
	Del(context.Context, ...string) (int64, error)
	Exists(context.Context, ...string) (int64, error)
	Expire(context.Context, string, time.Duration) (bool, error)
	TTL(context.Context, string) (time.Duration, error)
	MGet(context.Context, ...string) ([]Value, error)
	MSet(context.Context, map[string]string) error
	Incr(context.Context, string) (int64, error)
	Decr(context.Context, string) (int64, error)
}
