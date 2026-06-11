package testkit

import (
	"time"

	"github.com/ZoneCNH/redisx/pkg/redisx"
)

func Config(name string) redisx.Config {
	return redisx.Config{
		Name:    name,
		Timeout: time.Second,
	}
}
