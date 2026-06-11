package testkit

import (
	"context"

	"github.com/ZoneCNH/redisx/internal/provider"
	"github.com/ZoneCNH/redisx/pkg/redisx"
)

// NewFakeRedis returns an isolated in-memory Redis provider for tests.
// It does not read Redis environment variables and never opens a network
// connection, so generated tests cannot accidentally depend on a real Redis.
func NewFakeRedis() redisx.Provider {
	return provider.NewMemory()
}

// NewClientWithFakeRedis constructs a redisx client backed by NewFakeRedis.
func NewClientWithFakeRedis(ctx context.Context, cfg redisx.Config, opts ...redisx.Option) (*redisx.Client, error) {
	clientOptions := append([]redisx.Option{}, opts...)
	clientOptions = append(clientOptions, redisx.WithProvider(NewFakeRedis()))
	return redisx.New(ctx, cfg, clientOptions...)
}
