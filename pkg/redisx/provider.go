package redisx

import "github.com/ZoneCNH/redisx/internal/provider"

// Value represents a string Redis value that may be absent in multi-key reads.
type Value = provider.Value

// PipelineCommandType identifies a supported command in Client.Pipeline.
type PipelineCommandType = provider.PipelineCommandType

const (
	PipelineSet    = provider.PipelineSet
	PipelineMSet   = provider.PipelineMSet
	PipelineGet    = provider.PipelineGet
	PipelineHSet   = provider.PipelineHSet
	PipelineHGet   = provider.PipelineHGet
	PipelineRPush  = provider.PipelineRPush
	PipelineLRange = provider.PipelineLRange
	PipelineIncr   = provider.PipelineIncr
)

// PipelineCommand describes one supported Redis command enqueued by Client.Pipeline.
type PipelineCommand = provider.PipelineCommand

// PipelineOperation is kept for compatibility with earlier v1 prerelease tests.
type PipelineOperation = provider.PipelineCommand

// PipelineResult captures the result for one pipelined command.
type PipelineResult = provider.PipelineResult

// RateLimitResult describes a fixed-window rate-limit decision.
type RateLimitResult = provider.RateLimitResult

// Provider is the storage boundary used by Client. Implementations must not
// expose provider-specific types through this package API.
type Provider = provider.Provider
