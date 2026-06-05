package main

import (
	"fmt"
	"time"

	"github.com/ZoneCNH/redisx/pkg/redisx"
)

func main() {
	cfg := redisx.Config{
		Name:    "redisx",
		Timeout: time.Second,
		Secret:  "example",
	}

	fmt.Println(cfg.Sanitize().Secret)
}
