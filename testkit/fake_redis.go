package testkit

import (
	internalprovider "github.com/ZoneCNH/redisx/internal/provider"
	"github.com/ZoneCNH/redisx/pkg/redisx"
)

func NewFakeRedis() redisx.Provider {
	return internalprovider.NewMemory()
}
