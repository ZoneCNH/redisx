package redisx

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNewErrorFormatsKindOpAndMessage(t *testing.T) {
	err := NewError(ErrorKindValidation, "redisx.Test", "bad input", false)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Kind != ErrorKindValidation {
		t.Fatalf("expected validation kind, got %q", err.Kind)
	}
	if err.Retryable {
		t.Fatal("expected non-retryable error")
	}
	if got := err.Error(); !strings.Contains(got, "validation: redisx.Test: bad input") {
		t.Fatalf("unexpected error string: %q", got)
	}
}

func TestWrapErrorPreservesCauseAndKind(t *testing.T) {
	cause := context.DeadlineExceeded
	err := WrapError(ErrorKindTimeout, "redisx.Test", "", true, cause)

	if !IsKind(err, ErrorKindTimeout) {
		t.Fatalf("expected timeout kind, got %v", err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("expected wrapped cause, got %v", err)
	}
	if !err.Retryable {
		t.Fatal("expected retryable error")
	}
}

func TestErrorHandlesNilReceiverAndCauseOnlyMessage(t *testing.T) {
	var nilError *Error
	if got := nilError.Error(); got != "" {
		t.Fatalf("nil Error.Error() = %q; want empty string", got)
	}
	if got := nilError.Unwrap(); got != nil {
		t.Fatalf("nil Error.Unwrap() = %v; want nil", got)
	}

	cause := errors.New("root cause")
	err := &Error{Kind: ErrorKindInternal, Op: "redisx.Test", Cause: cause}
	if got := err.Error(); got != "internal: redisx.Test: root cause" {
		t.Fatalf("cause-only Error() = %q", got)
	}
}

func TestErrorKindFallbacksForPlainErrors(t *testing.T) {
	err := errors.New("plain")
	if IsKind(err, ErrorKindInternal) {
		t.Fatal("plain errors should not match redisx error kinds")
	}
	if got := errorKind(err); got != ErrorKindInternal {
		t.Fatalf("errorKind(plain) = %q; want %q", got, ErrorKindInternal)
	}
}

func TestContextErrorClassifiesDeadlineAsRetryableTimeout(t *testing.T) {
	err := contextError("redisx.Test", context.DeadlineExceeded)
	if !IsKind(err, ErrorKindTimeout) {
		t.Fatalf("expected timeout kind, got %v", err)
	}
	if !err.Retryable {
		t.Fatal("expected deadline errors to be retryable")
	}
}

func TestRedisErrorIDKindMapping(t *testing.T) {
	cases := map[RedisErrorID]ErrorKind{
		ErrNil:              ErrorKindNil,
		ErrTimeout:          ErrorKindTimeout,
		ErrCanceled:         ErrorKindCanceled,
		ErrNetwork:          ErrorKindConnection,
		ErrAuth:             ErrorKindAuth,
		ErrReadOnly:         ErrorKindConflict,
		ErrLoading:          ErrorKindUnavailable,
		ErrTryAgain:         ErrorKindUnavailable,
		ErrClusterMoved:     ErrorKindUnavailable,
		ErrClusterAsk:       ErrorKindUnavailable,
		ErrConnectionClosed: ErrorKindClosed,
		ErrInvalidConfig:    ErrorKindConfig,
		ErrProvider:         ErrorKindProvider,
	}

	for id, kind := range cases {
		if id.String() == "" {
			t.Fatalf("%v string identifier is empty", id)
		}
		if got := id.Kind(); got != kind {
			t.Fatalf("%s Kind() = %q; want %q", id, got, kind)
		}
	}
}
