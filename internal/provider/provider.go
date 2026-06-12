package provider

import (
	"context"
	"errors"
	"time"
)

var (
	ErrClosed       = errors.New("provider is closed")
	ErrNil          = errors.New("redis nil")
	ErrTimeout      = errors.New("redis timeout")
	ErrNetwork      = errors.New("redis network")
	ErrAuth         = errors.New("redis auth")
	ErrReadOnly     = errors.New("redis read only")
	ErrLoading      = errors.New("redis loading")
	ErrTryAgain     = errors.New("redis try again")
	ErrClusterMoved = errors.New("redis cluster moved")
	ErrClusterAsk   = errors.New("redis cluster ask")
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
