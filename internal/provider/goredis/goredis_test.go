package goredis

import (
	"errors"
	"testing"
	"time"

	"github.com/ZoneCNH/redisx/internal/provider"
)

func TestMapErrorClassifiesMisconfAsReadOnly(t *testing.T) {
	err := mapError(errors.New("MISCONF Redis is configured to save RDB snapshots, stop-writes-on-bgsave-error option"))
	if !errors.Is(err, provider.ErrReadOnly) {
		t.Fatalf("mapError(MISCONF) = %v, want provider.ErrReadOnly", err)
	}
}

func TestMapErrorClassifiesInvalidInteger(t *testing.T) {
	tests := []string{
		"ERR value is not an integer or out of range",
		"ERR increment or decrement would overflow and exceed out of range",
	}

	for _, message := range tests {
		t.Run(message, func(t *testing.T) {
			err := mapError(errors.New(message))
			if !errors.Is(err, provider.ErrInvalidInt) {
				t.Fatalf("mapError(%q) = %v, want provider.ErrInvalidInt", message, err)
			}
		})
	}
}

func TestNormalizeTTLUsesProviderSentinelSemantics(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
		want time.Duration
	}{
		{name: "no expiration", ttl: -time.Nanosecond, want: -time.Second},
		{name: "missing", ttl: -2 * time.Nanosecond, want: -2 * time.Second},
		{name: "positive", ttl: 30 * time.Second, want: 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeTTL(tt.ttl)
			if got != tt.want {
				t.Fatalf("normalizeTTL(%s) = %s, want %s", tt.ttl, got, tt.want)
			}
		})
	}
}
