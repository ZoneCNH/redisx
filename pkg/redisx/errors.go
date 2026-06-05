package redisx

import (
	"context"
	"errors"
)

type ErrorKind string

const (
	ErrorKindConfig      ErrorKind = "config"
	ErrorKindValidation  ErrorKind = "validation"
	ErrorKindConnection  ErrorKind = "connection"
	ErrorKindUnavailable ErrorKind = "unavailable"
	ErrorKindTimeout     ErrorKind = "timeout"
	ErrorKindAuth        ErrorKind = "auth"
	ErrorKindNetwork     ErrorKind = "network"
	ErrorKindReadOnly    ErrorKind = "read_only"
	ErrorKindLoading     ErrorKind = "loading"
	ErrorKindTryAgain    ErrorKind = "try_again"
	ErrorKindClusterMoved ErrorKind = "cluster_moved"
	ErrorKindClusterAsk  ErrorKind = "cluster_ask"
	ErrorKindConflict    ErrorKind = "conflict"
	ErrorKindRateLimit   ErrorKind = "rate_limit"
	ErrorKindInternal    ErrorKind = "internal"
	ErrorKindCanceled    ErrorKind = "canceled"
	ErrorKindNil         ErrorKind = "nil"
	ErrorKindClosed      ErrorKind = "closed"
	ErrorKindInvalidConfig ErrorKind = "invalid_config"
	ErrorKindProvider    ErrorKind = "provider"
)

var (
	ErrNil              = errors.New("redis nil")
	ErrTimeout          = errors.New("redis timeout")
	ErrCanceled         = errors.New("redis canceled")
	ErrNetwork          = errors.New("redis network")
	ErrAuth             = errors.New("redis auth")
	ErrReadOnly         = errors.New("redis read only")
	ErrLoading          = errors.New("redis loading")
	ErrTryAgain         = errors.New("redis try again")
	ErrClusterMoved     = errors.New("redis cluster moved")
	ErrClusterAsk       = errors.New("redis cluster ask")
	ErrConnectionClosed = errors.New("redis connection closed")
	ErrInvalidConfig    = errors.New("redis invalid config")
	ErrProvider         = errors.New("redis provider")
)

type Error struct {
	Kind      ErrorKind
	Op        string
	Message   string
	Cause     error
	Retryable bool
}

func NewError(kind ErrorKind, op string, message string, retryable bool) *Error {
	return newError(kind, op, message, retryable, nil)
}

func WrapError(kind ErrorKind, op string, message string, retryable bool, cause error) *Error {
	return newError(kind, op, message, retryable, cause)
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	message := string(e.Kind)
	if e.Op != "" {
		message += ": " + e.Op
	}
	if e.Message != "" {
		message += ": " + e.Message
	}
	if e.Message == "" && e.Cause != nil {
		message += ": " + e.Cause.Error()
	}
	return message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func IsKind(err error, kind ErrorKind) bool {
	var target *Error
	if errors.As(err, &target) {
		return target.Kind == kind
	}
	return false
}

func newError(kind ErrorKind, op string, message string, retryable bool, cause error) *Error {
	if message == "" && cause != nil {
		message = cause.Error()
	}
	return &Error{
		Kind:      kind,
		Op:        op,
		Message:   message,
		Cause:     cause,
		Retryable: retryable,
	}
}

func validationError(op string, message string, cause error) *Error {
	return newError(ErrorKindValidation, op, message, false, cause)
}

func contextError(op string, cause error) *Error {
	kind := ErrorKindUnavailable
	retryable := false
	if errors.Is(cause, context.Canceled) {
		kind = ErrorKindCanceled
	} else if errors.Is(cause, context.DeadlineExceeded) {
		kind = ErrorKindTimeout
		retryable = true
	}
	return newError(kind, op, "", retryable, cause)
}

func errorKind(err error) ErrorKind {
	var target *Error
	if errors.As(err, &target) {
		return target.Kind
	}
	return ErrorKindInternal
}
