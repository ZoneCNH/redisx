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
	ErrInvalidInt   = errors.New("redis value is not an integer")
	ErrWrongType    = errors.New("redis wrong type")
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

// PipelineCommandType identifies a supported pipeline command.
type PipelineCommandType string

const (
	PipelineSet    PipelineCommandType = "set"
	PipelineMSet   PipelineCommandType = "mset"
	PipelineGet    PipelineCommandType = "get"
	PipelineHSet   PipelineCommandType = "hset"
	PipelineHGet   PipelineCommandType = "hget"
	PipelineRPush  PipelineCommandType = "rpush"
	PipelineLRange PipelineCommandType = "lrange"
	PipelineIncr   PipelineCommandType = "incr"
)

// PipelineCommand describes one operation in a Redis pipeline.
type PipelineCommand struct {
	Type       PipelineCommandType `json:"type"`
	Op         PipelineCommandType `json:"op,omitempty"`
	Kind       PipelineCommandType `json:"kind,omitempty"`
	Key        string              `json:"key,omitempty"`
	Field      string              `json:"field,omitempty"`
	Value      string              `json:"value,omitempty"`
	Values     map[string]string   `json:"values,omitempty"`
	ListValues []string            `json:"list_values,omitempty"`
	List       []string            `json:"list,omitempty"`
	Items      []string            `json:"items,omitempty"`
	Start      int64               `json:"start,omitempty"`
	Stop       int64               `json:"stop,omitempty"`
	TTL        time.Duration       `json:"ttl,omitempty"`
}

// PipelineCommandKind returns the canonical command type, accepting older field names.
func PipelineCommandKind(command PipelineCommand) PipelineCommandType {
	if command.Type != "" {
		return command.Type
	}
	if command.Op != "" {
		return command.Op
	}
	return command.Kind
}

// PipelineCommandListValues returns the canonical list payload, accepting older field names.
func PipelineCommandListValues(command PipelineCommand) []string {
	if len(command.ListValues) > 0 {
		return command.ListValues
	}
	if len(command.List) > 0 {
		return command.List
	}
	return command.Items
}

// PipelineResult captures the result for a pipelined operation.
type PipelineResult struct {
	Type    PipelineCommandType `json:"type"`
	Key     string              `json:"key,omitempty"`
	Int     int64               `json:"int,omitempty"`
	Bool    bool                `json:"bool,omitempty"`
	Values  []Value             `json:"values,omitempty"`
	Found   bool                `json:"found,omitempty"`
	Value   string              `json:"value,omitempty"`
	Count   int64               `json:"count,omitempty"`
	Strings []string            `json:"strings,omitempty"`
}

// RateLimitResult describes the fixed-window rate limit decision.
type RateLimitResult struct {
	Allowed    bool          `json:"allowed"`
	Limit      int64         `json:"limit"`
	Remaining  int64         `json:"remaining"`
	ResetAfter time.Duration `json:"reset_after"`
	Count      int64         `json:"count"`
}

// Provider is the storage boundary implemented by Redis adapters.
type Provider interface {
	Ping(context.Context) error
	Close(context.Context) error
	Get(context.Context, string) (string, error)
	Set(context.Context, string, string, time.Duration) error
	SetNX(context.Context, string, string, time.Duration) (bool, error)
	Del(context.Context, ...string) (int64, error)
	Exists(context.Context, ...string) (int64, error)
	Expire(context.Context, string, time.Duration) (bool, error)
	TTL(context.Context, string) (time.Duration, error)
	MGet(context.Context, ...string) ([]Value, error)
	MSet(context.Context, map[string]string) error
	Incr(context.Context, string) (int64, error)
	Decr(context.Context, string) (int64, error)
	HSet(context.Context, string, map[string]string) (int64, error)
	HGet(context.Context, string, string) (string, error)
	HGetAll(context.Context, string) (map[string]string, error)
	HDel(context.Context, string, ...string) (int64, error)
	LPush(context.Context, string, ...string) (int64, error)
	RPush(context.Context, string, ...string) (int64, error)
	LRange(context.Context, string, int64, int64) ([]string, error)
	LLen(context.Context, string) (int64, error)
	LPop(context.Context, string) (string, error)
	RPop(context.Context, string) (string, error)
	Pipeline(context.Context, []PipelineCommand) ([]PipelineResult, error)
	AcquireLock(context.Context, string, string, time.Duration) (bool, error)
	ReleaseLock(context.Context, string, string) (bool, error)
	FixedWindowRateLimit(context.Context, string, int64, time.Duration) (RateLimitResult, error)
}
