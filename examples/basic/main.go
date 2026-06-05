package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/ZoneCNH/redisx/pkg/redisx"
)

func main() {
	run(os.Stdout, os.Stderr, redisx.Config{Name: "redisx"})
}

func run(stdout, stderr io.Writer, cfg redisx.Config) {
	client, err := redisx.New(context.Background(), cfg)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "create client: %v\n", err)
		return
	}
	defer func() {
		_ = client.Close(context.Background())
	}()

	_, _ = fmt.Fprintln(stdout, redisx.ModuleName)
}
