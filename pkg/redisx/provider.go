package redisx

import "github.com/ZoneCNH/redisx/internal/provider"

// Value represents a string Redis value that may be absent in multi-key reads.
type Value = provider.Value

// Provider is the storage boundary used by Client. Implementations must not
// expose provider-specific types through this package API.
type Provider = provider.Provider
