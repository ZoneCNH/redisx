package goredis

import (
	"errors"
	"testing"

	"github.com/ZoneCNH/redisx/internal/provider"
)

func TestMapErrorClassifiesMisconfAsReadOnly(t *testing.T) {
	err := mapError(errors.New("MISCONF Redis is configured to save RDB snapshots, stop-writes-on-bgsave-error option"))
	if !errors.Is(err, provider.ErrReadOnly) {
		t.Fatalf("mapError(MISCONF) = %v, want provider.ErrReadOnly", err)
	}
}
