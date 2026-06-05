package redisx

import (
	"context"
	"errors"
)

type ErrorKind string

type ErrorIdentifier string

const (
	ErrorKindConfig        ErrorKind = "config"
	ErrorKindValidation    ErrorKind = "validation"
	ErrorKindConnection    ErrorKind = "connection"
	ErrorKindUnavailable   ErrorKind = "unavailable"
	ErrorKindTimeout       ErrorKind = "timeout"
	ErrorKindAuth          ErrorKind = "auth"
	ErrorKindNetwork       ErrorKind = "network"
	ErrorKindReadOnly      ErrorKind = "read_only"
	ErrorKindLoading       ErrorKind = "loading"
	ErrorKindTryAgain      ErrorKind = "try_again"
	ErrorKindClusterMoved  ErrorKind = "cluster_moved"
	ErrorKindClusterAsk    ErrorKind = "cluster_ask"
	ErrorKindConflict      ErrorKind = "conflict"
	ErrorKindRateLimit     ErrorKind = "rate_limit"
	ErrorKindInternal      ErrorKind = "internal"
	ErrorKindCanceled      ErrorKind = "canceled"
	ErrorKindNil           ErrorKind = "nil"
	ErrorKindClosed        ErrorKind = "closed"
	ErrorKindInvalidConfig ErrorKind = "invalid_config"
	ErrorKindProvider      ErrorKind = "provider"
)

const (
	ErrNil              ErrorIdentifier = "ErrNil"
	ErrTimeout          ErrorIdentifier = "ErrTimeout"
	ErrCanceled         ErrorIdentifier = "ErrCanceled"
	ErrNetwork          ErrorIdentifier = "ErrNetwork"
	ErrAuth             ErrorIdentifier = "ErrAuth"
	ErrReadOnly         ErrorIdentifier = "ErrReadOnly"
	ErrLoading          ErrorIdentifier = "ErrLoading"
	ErrTryAgain         ErrorIdentifier = "ErrTryAgain"
	ErrClusterMoved     ErrorIdentifier = "ErrClusterMoved"
	ErrClusterAsk       ErrorIdentifier = "ErrClusterAsk"
	ErrConnectionClosed ErrorIdentifier = "ErrConnectionClosed"
	ErrInvalidConfig    ErrorIdentifier = "ErrInvalidConfig"
	ErrProvider         ErrorIdentifier = "ErrProvider"
)


// RedisErrorID is the Redis-specific error taxonomy identifier exposed for
// contract documents and downstream adapters. It is additive to ErrorKind so
// existing generic error classification remains stable.
type RedisErrorID string

const (
	ErrNil              RedisErrorID = "redis.nil"
	ErrTimeout          RedisErrorID = "redis.timeout"
	ErrCanceled         RedisErrorID = "redis.canceled"
	ErrNetwork          RedisErrorID = "redis.network"
	ErrAuth             RedisErrorID = "redis.auth"
	ErrReadOnly         RedisErrorID = "redis.read_only"
	ErrLoading          RedisErrorID = "redis.loading"
	ErrTryAgain         RedisErrorID = "redis.try_again"
	ErrClusterMoved     RedisErrorID = "redis.cluster_moved"
	ErrClusterAsk       RedisErrorID = "redis.cluster_ask"
	ErrConnectionClosed RedisErrorID = "redis.connection_closed"
	ErrInvalidConfig    RedisErrorID = "redis.invalid_config"
	ErrProvider         RedisErrorID = "redis.provider"
)

func (id RedisErrorID) String() string {
	return string(id)
}

func (id RedisErrorID) Kind() ErrorKind {
	switch id {
	case ErrNil:
		return ErrorKindNil
	case ErrTimeout:
		return ErrorKindTimeout
	case ErrCanceled:
		return ErrorKindCanceled
	case ErrNetwork:
		return ErrorKindConnection
	case ErrAuth:
		return ErrorKindAuth
	case ErrReadOnly:
		return ErrorKindConflict
	case ErrLoading, ErrTryAgain, ErrClusterMoved, ErrClusterAsk:
		return ErrorKindUnavailable
	case ErrConnectionClosed:
		return ErrorKindClosed
	case ErrInvalidConfig:
		return ErrorKindConfig
	case ErrProvider:
		return ErrorKindProvider
	default:
		return ErrorKindInternal
	}
}

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

func (e *Error) Is(target error) bool {
	if e == nil {
		return false
	}
	switch target {
	case ErrNil:
		return e.Kind == ErrorKindNil
	case ErrTimeout:
		return e.Kind == ErrorKindTimeout
	case ErrCanceled:
		return e.Kind == ErrorKindCanceled
	case ErrNetwork:
		return e.Kind == ErrorKindNetwork
	case ErrAuth:
		return e.Kind == ErrorKindAuth
	case ErrReadOnly:
		return e.Kind == ErrorKindReadOnly
	case ErrLoading:
		return e.Kind == ErrorKindLoading
	case ErrTryAgain:
		return e.Kind == ErrorKindTryAgain
	case ErrClusterMoved:
		return e.Kind == ErrorKindClusterMoved
	case ErrClusterAsk:
		return e.Kind == ErrorKindClusterAsk
	case ErrConnectionClosed:
		return e.Kind == ErrorKindClosed
	case ErrInvalidConfig:
		return e.Kind == ErrorKindInvalidConfig
	case ErrProvider:
		return e.Kind == ErrorKindProvider
	default:
		return false
	}
}

func IsKind(err error, kind ErrorKind) bool {
	var target *Error
	if errors.As(err, &target) {
		return target.Kind == kind
	}
	return false
}

func ErrorIdentifierForKind(kind ErrorKind) ErrorIdentifier {
	switch kind {
	case ErrorKindConfig, ErrorKindValidation:
		return ErrInvalidConfig
	case ErrorKindTimeout:
		return ErrTimeout
	case ErrorKindCanceled:
		return ErrCanceled
	case ErrorKindAuth:
		return ErrAuth
	case ErrorKindConnection:
		return ErrConnectionClosed
	case ErrorKindNil:
		return ErrNil
	case ErrorKindClosed:
		return ErrConnectionClosed
	case ErrorKindProvider, ErrorKindUnavailable, ErrorKindConflict, ErrorKindRateLimit, ErrorKindInternal:
		return ErrProvider
	default:
		return ErrProvider
	}
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
