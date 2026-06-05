package testkit

import (
	"github.com/ZoneCNH/redisx/internal/provider"
	"github.com/ZoneCNH/redisx/pkg/redisx"
)

func NewFakeRedis() redisx.Provider {
	return provider.NewMemory()
}
