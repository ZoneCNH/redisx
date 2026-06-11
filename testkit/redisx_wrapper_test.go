package testkit

import (
	"context"

	"github.com/ZoneCNH/redisx/pkg/redisx"
)

func redisxNew(ctx context.Context, cfg redisx.Config, opts ...redisx.Option) (*redisx.Client, error) {
	return redisx.New(ctx, cfg, opts...)
}
