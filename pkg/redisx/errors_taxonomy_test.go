package redisx

import (
	"errors"
	"testing"
)

func TestRedisErrorIdentifiersArePublicAndStable(t *testing.T) {
	for name, err := range map[string]error{
		"ErrNil":              ErrNil,
		"ErrTimeout":          ErrTimeout,
		"ErrCanceled":         ErrCanceled,
		"ErrNetwork":          ErrNetwork,
		"ErrAuth":             ErrAuth,
		"ErrReadOnly":         ErrReadOnly,
		"ErrLoading":          ErrLoading,
		"ErrTryAgain":         ErrTryAgain,
		"ErrClusterMoved":     ErrClusterMoved,
		"ErrClusterAsk":       ErrClusterAsk,
		"ErrConnectionClosed": ErrConnectionClosed,
		"ErrInvalidConfig":    ErrInvalidConfig,
		"ErrProvider":         ErrProvider,
	} {
		if err == nil || err.Error() == "" {
			t.Fatalf("%s must be a non-empty public sentinel", name)
		}
	}
}

func TestProviderErrorsMapToRedisTaxonomy(t *testing.T) {
	cases := []struct {
		cause     error
		kind      ErrorKind
		retryable bool
	}{
		{ErrNil, ErrorKindNil, false},
		{ErrTimeout, ErrorKindTimeout, true},
		{ErrCanceled, ErrorKindCanceled, false},
		{ErrNetwork, ErrorKindNetwork, true},
		{ErrAuth, ErrorKindAuth, false},
		{ErrReadOnly, ErrorKindReadOnly, true},
		{ErrLoading, ErrorKindLoading, true},
		{ErrTryAgain, ErrorKindTryAgain, true},
		{ErrClusterMoved, ErrorKindClusterMoved, true},
		{ErrClusterAsk, ErrorKindClusterAsk, true},
		{ErrConnectionClosed, ErrorKindClosed, false},
		{ErrInvalidConfig, ErrorKindInvalidConfig, false},
		{ErrProvider, ErrorKindProvider, false},
	}
	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			err := providerError("redisx.Test", tc.cause)
			if !IsKind(err, tc.kind) || err.Retryable != tc.retryable || !errors.Is(err, tc.cause) {
				t.Fatalf("providerError(%v) = %#v", tc.cause, err)
			}
		})
	}
}
