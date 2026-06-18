package redisx

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	internalprovider "github.com/ZoneCNH/redisx/internal/provider"
	redisprovider "github.com/ZoneCNH/redisx/internal/provider/goredis"
)

type coverageProvider struct {
	err              error
	closeErr         error
	expireErr        error
	getValue         string
	setNX            bool
	llen             int64
	incrValue        int64
	acquireLockSet   bool
	acquireLock      bool
	pipelineCommands []PipelineCommand
	rateLimit        RateLimitResult
	closed           bool
}

func (p *coverageProvider) Ping(context.Context) error { return p.err }

func (p *coverageProvider) Close(context.Context) error {
	p.closed = true
	if p.closeErr != nil {
		return p.closeErr
	}
	return p.err
}

func (p *coverageProvider) Get(context.Context, string) (string, error) {
	if p.err != nil {
		return "", p.err
	}
	return p.getValue, nil
}

func (p *coverageProvider) Set(context.Context, string, string, time.Duration) error {
	return p.err
}

func (p *coverageProvider) SetNX(context.Context, string, string, time.Duration) (bool, error) {
	if p.err != nil {
		return false, p.err
	}
	return p.setNX, nil
}

func (p *coverageProvider) Del(context.Context, ...string) (int64, error) {
	if p.err != nil {
		return 0, p.err
	}
	return 1, nil
}

func (p *coverageProvider) Exists(context.Context, ...string) (int64, error) {
	if p.err != nil {
		return 0, p.err
	}
	return 1, nil
}

func (p *coverageProvider) Expire(context.Context, string, time.Duration) (bool, error) {
	if p.expireErr != nil {
		return false, p.expireErr
	}
	if p.err != nil {
		return false, p.err
	}
	return true, nil
}

func (p *coverageProvider) TTL(context.Context, string) (time.Duration, error) {
	if p.err != nil {
		return 0, p.err
	}
	return time.Minute, nil
}

func (p *coverageProvider) MGet(context.Context, ...string) ([]Value, error) {
	if p.err != nil {
		return nil, p.err
	}
	return []Value{{Value: "value", Found: true}}, nil
}

func (p *coverageProvider) MSet(context.Context, map[string]string) error { return p.err }

func (p *coverageProvider) Incr(context.Context, string) (int64, error) {
	if p.err != nil {
		return 0, p.err
	}
	if p.incrValue != 0 {
		return p.incrValue, nil
	}
	return 1, nil
}

func (p *coverageProvider) Decr(context.Context, string) (int64, error) {
	if p.err != nil {
		return 0, p.err
	}
	return -1, nil
}

func (p *coverageProvider) HSet(context.Context, string, map[string]string) (int64, error) {
	if p.err != nil {
		return 0, p.err
	}
	return 1, nil
}

func (p *coverageProvider) HGet(context.Context, string, string) (string, error) {
	if p.err != nil {
		return "", p.err
	}
	return "hash-value", nil
}

func (p *coverageProvider) HGetAll(context.Context, string) (map[string]string, error) {
	if p.err != nil {
		return nil, p.err
	}
	return map[string]string{"field": "value"}, nil
}

func (p *coverageProvider) HDel(context.Context, string, ...string) (int64, error) {
	if p.err != nil {
		return 0, p.err
	}
	return 1, nil
}

func (p *coverageProvider) LPush(context.Context, string, ...string) (int64, error) {
	if p.err != nil {
		return 0, p.err
	}
	return 1, nil
}

func (p *coverageProvider) RPush(context.Context, string, ...string) (int64, error) {
	if p.err != nil {
		return 0, p.err
	}
	return 1, nil
}

func (p *coverageProvider) LRange(context.Context, string, int64, int64) ([]string, error) {
	if p.err != nil {
		return nil, p.err
	}
	return []string{"a", "b"}, nil
}

func (p *coverageProvider) LLen(context.Context, string) (int64, error) {
	if p.err != nil {
		return 0, p.err
	}
	return p.llen, nil
}

func (p *coverageProvider) LPop(context.Context, string) (string, error) {
	if p.err != nil {
		return "", p.err
	}
	return "left", nil
}

func (p *coverageProvider) RPop(context.Context, string) (string, error) {
	if p.err != nil {
		return "", p.err
	}
	return "right", nil
}

func (p *coverageProvider) Pipeline(_ context.Context, commands []PipelineCommand) ([]PipelineResult, error) {
	p.pipelineCommands = append([]PipelineCommand(nil), commands...)
	if p.err != nil {
		return nil, p.err
	}
	return []PipelineResult{{Type: PipelineGet, Key: "pipe", Found: true, Value: "ok"}}, nil
}

func (p *coverageProvider) AcquireLock(context.Context, string, string, time.Duration) (bool, error) {
	if p.err != nil {
		return false, p.err
	}
	if p.acquireLockSet {
		return p.acquireLock, nil
	}
	return true, nil
}

func (p *coverageProvider) ReleaseLock(context.Context, string, string) (bool, error) {
	if p.err != nil {
		return false, p.err
	}
	return true, nil
}

func (p *coverageProvider) FixedWindowRateLimit(context.Context, string, int64, time.Duration) (RateLimitResult, error) {
	if p.err != nil {
		return RateLimitResult{}, p.err
	}
	if p.rateLimit.Limit != 0 {
		return p.rateLimit, nil
	}
	return RateLimitResult{Allowed: true, Limit: 2, Remaining: 1, Count: 1, ResetAfter: time.Minute}, nil
}

type coverageStringCodec struct {
	marshalErr   error
	unmarshalErr error
}

func (c coverageStringCodec) Marshal(value string) (string, error) {
	if c.marshalErr != nil {
		return "", c.marshalErr
	}
	return "encoded:" + value, nil
}

func (c coverageStringCodec) Unmarshal(raw string) (string, error) {
	if c.unmarshalErr != nil {
		return "", c.unmarshalErr
	}
	return raw, nil
}

func TestCoverageBOptionsAndProviderOverrides(t *testing.T) {
	ctx := context.Background()
	metrics := &recordingMetrics{}
	provider := &coverageProvider{getValue: "from-provider"}
	opts := Options{
		Config:   Config{Name: "redisx-options-b"},
		Metrics:  metrics,
		Provider: provider,
	}

	clientOptions := opts.clientOptions()
	if len(clientOptions) != 2 {
		t.Fatalf("expected metrics and provider options, got %d", len(clientOptions))
	}
	if err := opts.Validate(); err != nil {
		t.Fatalf("valid options rejected: %v", err)
	}
	client, err := NewWithOptions(ctx, opts)
	if err != nil {
		t.Fatalf("NewWithOptions returned error: %v", err)
	}
	value, err := client.Get(ctx, "custom-key")
	if err != nil {
		t.Fatalf("custom provider Get returned error: %v", err)
	}
	if value != "from-provider" {
		t.Fatalf("Get value = %q, want custom provider value", value)
	}
	if !metrics.hasCounter(MetricClientCreatedTotal) || !metrics.hasCounter(MetricRedisOperationsTotal) {
		t.Fatalf("expected creation and operation metrics, got %#v", metrics)
	}

	defaults := defaultOptions()
	WithMetrics(nil)(&defaults)
	WithProvider(nil)(&defaults)
	if defaults.metrics == nil {
		t.Fatal("WithMetrics(nil) removed default metrics")
	}
	if defaults.providerSet || defaults.provider != nil {
		t.Fatalf("WithProvider(nil) should not set a provider: %#v", defaults)
	}
}

func TestCoverageBClientPublicOperationEdges(t *testing.T) {
	ctx := context.Background()
	client, err := New(ctx, Config{Name: "redisx-operation-edges-b"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	ok, err := client.SetNX(ctx, "nx-b", "one", time.Minute)
	if err != nil || !ok {
		t.Fatalf("first SetNX = %v, %v; want true nil", ok, err)
	}
	ok, err = client.SetNX(ctx, "nx-b", "two", time.Minute)
	if err != nil || ok {
		t.Fatalf("second SetNX = %v, %v; want false nil", ok, err)
	}
	if _, err := client.SetNX(ctx, "nx-negative-b", "value", -time.Second); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("SetNX negative ttl error = %v, want validation", err)
	}

	if _, err := client.RPush(ctx, "list-b", "a", "b"); err != nil {
		t.Fatalf("RPush returned error: %v", err)
	}
	length, err := client.LLen(ctx, "list-b")
	if err != nil || length != 2 {
		t.Fatalf("LLen = %d, %v; want 2 nil", length, err)
	}
	if _, err := client.LLen(ctx, ""); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("LLen empty key error = %v, want validation", err)
	}

	if _, err := client.AcquireLock(ctx, "lock-b", "", time.Second); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("AcquireLock empty token error = %v, want validation", err)
	}
	if _, err := client.AcquireLock(ctx, "lock-b", "token", 0); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("AcquireLock zero ttl error = %v, want validation", err)
	}
	if _, err := client.ReleaseLock(ctx, "lock-b", ""); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("ReleaseLock empty token error = %v, want validation", err)
	}
	if _, err := client.FixedWindowRateLimit(ctx, "rate-b", 0, time.Minute); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("rate limit zero limit error = %v, want validation", err)
	}
	if _, err := client.FixedWindowRateLimit(ctx, "rate-b", 1, 0); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("rate limit zero window error = %v, want validation", err)
	}
	result, err := client.FixedWindowRateLimit(ctx, "rate-b", 2, time.Minute)
	if err != nil || result.Limit != 2 || !result.Allowed {
		t.Fatalf("FixedWindowRateLimit = %#v, %v; want allowed limit=2", result, err)
	}
}

func TestCoverageBPipelineValidationAndVariadicForms(t *testing.T) {
	ctx := context.Background()
	client, err := New(ctx, Config{Name: "redisx-pipeline-b"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	results, err := client.Pipeline(ctx,
		PipelineCommand{Type: PipelineSet, Key: "pipeline-b", Value: "value"},
		PipelineCommand{Op: PipelineGet, Key: "pipeline-b"},
		PipelineCommand{Kind: PipelineRPush, Key: "pipeline-list-b", Items: []string{"a", "b"}},
	)
	if err != nil {
		t.Fatalf("variadic Pipeline returned error: %v", err)
	}
	if len(results) != 3 || !results[1].Found || results[1].Value != "value" {
		t.Fatalf("unexpected pipeline results: %#v", results)
	}

	cases := []struct {
		name string
		run  func() error
	}{
		{"mix slice and rest", func() error {
			_, err := client.Pipeline(ctx, []PipelineCommand{{Type: PipelineGet, Key: "k"}}, PipelineCommand{Type: PipelineGet, Key: "k"})
			return err
		}},
		{"wrong first type", func() error {
			_, err := client.Pipeline(ctx, "not-commands")
			return err
		}},
		{"empty slice", func() error {
			_, err := client.Pipeline(ctx, []PipelineCommand{})
			return err
		}},
		{"set empty key", func() error {
			_, err := client.Pipeline(ctx, []PipelineCommand{{Type: PipelineSet, Value: "value"}})
			return err
		}},
		{"set negative ttl", func() error {
			_, err := client.Pipeline(ctx, []PipelineCommand{{Type: PipelineSet, Key: "k", Value: "value", TTL: -time.Second}})
			return err
		}},
		{"get empty key", func() error {
			_, err := client.Pipeline(ctx, []PipelineCommand{{Type: PipelineGet}})
			return err
		}},
		{"mset empty", func() error {
			_, err := client.Pipeline(ctx, []PipelineCommand{{Type: PipelineMSet}})
			return err
		}},
		{"mset empty key", func() error {
			_, err := client.Pipeline(ctx, []PipelineCommand{{Type: PipelineMSet, Values: map[string]string{"": "value"}}})
			return err
		}},
		{"hset empty key", func() error {
			_, err := client.Pipeline(ctx, []PipelineCommand{{Type: PipelineHSet, Values: map[string]string{"field": "value"}}})
			return err
		}},
		{"hset empty values", func() error {
			_, err := client.Pipeline(ctx, []PipelineCommand{{Type: PipelineHSet, Key: "hash"}})
			return err
		}},
		{"hset empty field", func() error {
			_, err := client.Pipeline(ctx, []PipelineCommand{{Type: PipelineHSet, Key: "hash", Values: map[string]string{"": "value"}}})
			return err
		}},
		{"rpush no values", func() error {
			_, err := client.Pipeline(ctx, []PipelineCommand{{Type: PipelineRPush, Key: "list"}})
			return err
		}},
		{"rpush empty key", func() error {
			_, err := client.Pipeline(ctx, []PipelineCommand{{Type: PipelineRPush, Items: []string{"value"}}})
			return err
		}},
		{"hget empty key", func() error {
			_, err := client.Pipeline(ctx, []PipelineCommand{{Type: PipelineHGet, Field: "field"}})
			return err
		}},
		{"hget empty field", func() error {
			_, err := client.Pipeline(ctx, []PipelineCommand{{Type: PipelineHGet, Key: "hash"}})
			return err
		}},
		{"lrange empty key", func() error {
			_, err := client.Pipeline(ctx, []PipelineCommand{{Type: PipelineLRange}})
			return err
		}},
		{"incr empty key", func() error {
			_, err := client.Pipeline(ctx, []PipelineCommand{{Type: PipelineIncr}})
			return err
		}},
		{"unsupported", func() error {
			_, err := client.Pipeline(ctx, []PipelineCommand{{Type: PipelineCommandType("unknown"), Key: "k"}})
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); !IsKind(err, ErrorKindValidation) {
				t.Fatalf("error = %v, want validation", err)
			}
		})
	}
}

func TestCoverageBProviderStateAndOperationErrors(t *testing.T) {
	ctx := context.Background()
	var nilCtx context.Context
	metrics := &recordingMetrics{}

	var nilClient *Client
	if err := nilClient.Ping(ctx); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("nil client Ping error = %v, want validation", err)
	}
	if err := (&Client{metrics: metrics}).Ping(ctx); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("uninitialized Ping error = %v, want validation", err)
	}
	if err := (&Client{initialized: true, metrics: metrics}).Ping(ctx); !IsKind(err, ErrorKindProvider) {
		t.Fatalf("nil provider Ping error = %v, want provider", err)
	}
	if err := (&Client{initialized: true, metrics: metrics, provider: &coverageProvider{}}).Ping(nilCtx); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("nil context Ping error = %v, want validation", err)
	}
	if !metrics.counterWithLabel(MetricClientErrorsTotal, "kind", string(ErrorKindProvider)) {
		t.Fatalf("expected provider error metric, got %#v", metrics)
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	client := &Client{initialized: true, metrics: metrics, provider: &coverageProvider{}}
	if err := client.Ping(canceled); !IsKind(err, ErrorKindCanceled) {
		t.Fatalf("canceled Ping error = %v, want canceled", err)
	}
	client.closed = true
	if err := client.Ping(ctx); !IsKind(err, ErrorKindClosed) {
		t.Fatalf("closed Ping error = %v, want closed", err)
	}

	for _, tc := range []struct {
		name string
		run  func(*Client) error
	}{
		{"set", func(c *Client) error { return c.Set(ctx, "k", "v", 0) }},
		{"setnx", func(c *Client) error {
			_, err := c.SetNX(ctx, "k", "v", 0)
			return err
		}},
		{"hdel", func(c *Client) error {
			_, err := c.HDel(ctx, "h", "f")
			return err
		}},
		{"llen", func(c *Client) error {
			_, err := c.LLen(ctx, "list")
			return err
		}},
		{"pipeline", func(c *Client) error {
			_, err := c.Pipeline(ctx, []PipelineCommand{{Type: PipelineGet, Key: "k"}})
			return err
		}},
		{"rate-limit", func(c *Client) error {
			_, err := c.FixedWindowRateLimit(ctx, "rate", 1, time.Minute)
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &Client{initialized: true, metrics: &recordingMetrics{}, provider: &coverageProvider{err: internalprovider.ErrNetwork}}
			if err := tc.run(c); !IsKind(err, ErrorKindNetwork) {
				t.Fatalf("wrapped provider error = %v, want network", err)
			}
		})
	}

	closeClient := &Client{
		cfg:         Config{Name: "close-error-b"},
		metrics:     &recordingMetrics{},
		provider:    &coverageProvider{closeErr: internalprovider.ErrClosed},
		initialized: true,
	}
	if err := closeClient.Close(ctx); !IsKind(err, ErrorKindClosed) {
		t.Fatalf("Close provider error = %v, want closed", err)
	}
	if !closeClient.provider.(*coverageProvider).closed {
		t.Fatal("Close did not call provider")
	}
}

func TestCoverageBProviderErrorTaxonomy(t *testing.T) {
	_, parseErr := strconv.Atoi("not-int")
	cases := []struct {
		name      string
		cause     error
		kind      ErrorKind
		retryable bool
	}{
		{"internal nil", internalprovider.ErrNil, ErrorKindNil, false},
		{"public nil", ErrNil, ErrorKindNil, false},
		{"internal closed", internalprovider.ErrClosed, ErrorKindClosed, false},
		{"public closed", ErrConnectionClosed, ErrorKindClosed, false},
		{"deadline", context.DeadlineExceeded, ErrorKindTimeout, true},
		{"canceled", context.Canceled, ErrorKindCanceled, false},
		{"internal timeout", internalprovider.ErrTimeout, ErrorKindTimeout, true},
		{"internal network", internalprovider.ErrNetwork, ErrorKindNetwork, true},
		{"internal auth", internalprovider.ErrAuth, ErrorKindAuth, false},
		{"internal read only", internalprovider.ErrReadOnly, ErrorKindReadOnly, true},
		{"internal loading", internalprovider.ErrLoading, ErrorKindLoading, true},
		{"internal try again", internalprovider.ErrTryAgain, ErrorKindTryAgain, true},
		{"internal moved", internalprovider.ErrClusterMoved, ErrorKindClusterMoved, true},
		{"internal ask", internalprovider.ErrClusterAsk, ErrorKindClusterAsk, true},
		{"public timeout", ErrTimeout, ErrorKindTimeout, true},
		{"public canceled", ErrCanceled, ErrorKindCanceled, false},
		{"public network", ErrNetwork, ErrorKindNetwork, true},
		{"public auth", ErrAuth, ErrorKindAuth, false},
		{"internal invalid int", internalprovider.ErrInvalidInt, ErrorKindValidation, false},
		{"internal wrong type", internalprovider.ErrWrongType, ErrorKindValidation, false},
		{"public read only", ErrReadOnly, ErrorKindReadOnly, true},
		{"public loading", ErrLoading, ErrorKindLoading, true},
		{"public try again", ErrTryAgain, ErrorKindTryAgain, true},
		{"public moved", ErrClusterMoved, ErrorKindClusterMoved, true},
		{"public ask", ErrClusterAsk, ErrorKindClusterAsk, true},
		{"public invalid config", ErrInvalidConfig, ErrorKindInvalidConfig, false},
		{"public provider", ErrProvider, ErrorKindProvider, false},
		{"parse", parseErr, ErrorKindValidation, false},
		{"default", errors.New("plain provider failure"), ErrorKindProvider, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := providerError("redisx.test", tc.cause)
			if err.Kind != tc.kind || err.Retryable != tc.retryable {
				t.Fatalf("providerError(%v) = kind %s retryable %v, want %s %v", tc.cause, err.Kind, err.Retryable, tc.kind, tc.retryable)
			}
			if !errors.Is(err, tc.cause) {
				t.Fatalf("providerError(%v) does not unwrap original cause: %v", tc.cause, err)
			}
		})
	}
}

func TestCoverageBErrorIdentifierSwitches(t *testing.T) {
	idKinds := map[RedisErrorID]ErrorKind{
		ErrNil:              ErrorKindNil,
		ErrTimeout:          ErrorKindTimeout,
		ErrCanceled:         ErrorKindCanceled,
		ErrNetwork:          ErrorKindNetwork,
		ErrAuth:             ErrorKindAuth,
		ErrReadOnly:         ErrorKindReadOnly,
		ErrLoading:          ErrorKindLoading,
		ErrTryAgain:         ErrorKindTryAgain,
		ErrClusterMoved:     ErrorKindClusterMoved,
		ErrClusterAsk:       ErrorKindClusterAsk,
		ErrConnectionClosed: ErrorKindClosed,
		ErrInvalidConfig:    ErrorKindInvalidConfig,
		ErrProvider:         ErrorKindProvider,
		RedisErrorID("x"):   ErrorKindInternal,
	}
	for id, kind := range idKinds {
		if got := id.Kind(); got != kind {
			t.Fatalf("%s.Kind() = %s, want %s", id, got, kind)
		}
		if id != RedisErrorID("x") && !errors.Is(WrapError(kind, "op", "", false, id), id) {
			t.Fatalf("wrapped %s is not errors.Is(%s)", kind, id)
		}
	}

	kindIDs := map[ErrorKind]RedisErrorID{
		ErrorKindConfig:        ErrInvalidConfig,
		ErrorKindValidation:    ErrInvalidConfig,
		ErrorKindInvalidConfig: ErrInvalidConfig,
		ErrorKindTimeout:       ErrTimeout,
		ErrorKindCanceled:      ErrCanceled,
		ErrorKindNetwork:       ErrNetwork,
		ErrorKindAuth:          ErrAuth,
		ErrorKindReadOnly:      ErrReadOnly,
		ErrorKindLoading:       ErrLoading,
		ErrorKindTryAgain:      ErrTryAgain,
		ErrorKindClusterMoved:  ErrClusterMoved,
		ErrorKindClusterAsk:    ErrClusterAsk,
		ErrorKindConnection:    ErrConnectionClosed,
		ErrorKindClosed:        ErrConnectionClosed,
		ErrorKindNil:           ErrNil,
		ErrorKindProvider:      ErrProvider,
		ErrorKindUnavailable:   ErrProvider,
		ErrorKindConflict:      ErrProvider,
		ErrorKindRateLimit:     ErrProvider,
		ErrorKindInternal:      ErrProvider,
		ErrorKind("unknown"):   ErrProvider,
	}
	for kind, id := range kindIDs {
		if got := ErrorIdentifierForKind(kind); got != id {
			t.Fatalf("ErrorIdentifierForKind(%s) = %s, want %s", kind, got, id)
		}
	}

	var nilErr *Error
	if nilErr.Error() != "" || nilErr.Unwrap() != nil || nilErr.Is(ErrProvider) {
		t.Fatal("nil *Error methods returned non-zero values")
	}
	if NewError(ErrorKindProvider, "op", "", false).Is(errors.New("not an id")) {
		t.Fatal("Error.Is matched a non RedisErrorID target")
	}
	if NewError(ErrorKindProvider, "op", "", false).Is(RedisErrorID("unknown")) {
		t.Fatal("Error.Is matched an unknown RedisErrorID")
	}
}

func TestCoverageBNoopMetricsMethods(t *testing.T) {
	metrics := NoopMetrics{}
	metrics.IncCounter("counter", map[string]string{"k": "v"})
	metrics.ObserveHistogram("histogram", 1.5, map[string]string{"k": "v"})
	metrics.SetGauge("gauge", 2.5, nil)
}

func TestCoverageBNewValidationEdges(t *testing.T) {
	ctx := context.Background()
	var nilCtx context.Context
	if _, err := New(nilCtx, Config{Name: "new-nil-context-b"}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("New nil context error = %v, want validation", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := New(canceled, Config{Name: "new-canceled-b"}); !IsKind(err, ErrorKindCanceled) {
		t.Fatalf("New canceled context error = %v, want canceled", err)
	}
	if _, err := New(ctx, Config{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("New missing name error = %v, want validation", err)
	}
	if _, err := New(ctx, Config{Name: "new-invalid-redis-b", Redis: RedisConfig{DB: -1}}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("New invalid redis config error = %v, want validation", err)
	}
}

func TestCoverageBClientOperationSuccessValidationAndProviderErrors(t *testing.T) {
	ctx := context.Background()
	successProvider := &coverageProvider{
		getValue:  "value",
		setNX:     true,
		llen:      2,
		rateLimit: RateLimitResult{Allowed: false, Limit: 1, Remaining: 0, Count: 2, ResetAfter: time.Second},
	}
	success := &Client{cfg: Config{Name: "ops-success-b"}, initialized: true, metrics: &recordingMetrics{}, provider: successProvider}

	successOps := []struct {
		name string
		run  func() error
	}{
		{"ping", func() error { return success.Ping(ctx) }},
		{"get", func() error {
			value, err := success.Get(ctx, "key")
			if value != "value" {
				t.Fatalf("Get value = %q, want value", value)
			}
			return err
		}},
		{"set", func() error { return success.Set(ctx, "key", "value", 0) }},
		{"setnx", func() error {
			ok, err := success.SetNX(ctx, "key", "value", time.Second)
			if !ok {
				t.Fatal("SetNX returned false, want true")
			}
			return err
		}},
		{"del", func() error {
			_, err := success.Del(ctx, "key")
			return err
		}},
		{"exists", func() error {
			_, err := success.Exists(ctx, "key")
			return err
		}},
		{"expire", func() error {
			ok, err := success.Expire(ctx, "key", time.Second)
			if !ok {
				t.Fatal("Expire returned false, want true")
			}
			return err
		}},
		{"ttl", func() error {
			_, err := success.TTL(ctx, "key")
			return err
		}},
		{"mget", func() error {
			_, err := success.MGet(ctx, "key")
			return err
		}},
		{"mset", func() error { return success.MSet(ctx, map[string]string{"key": "value"}) }},
		{"incr", func() error {
			_, err := success.Incr(ctx, "key")
			return err
		}},
		{"decr", func() error {
			_, err := success.Decr(ctx, "key")
			return err
		}},
		{"hset", func() error {
			_, err := success.HSet(ctx, "hash", map[string]string{"field": "value"})
			return err
		}},
		{"hget", func() error {
			_, err := success.HGet(ctx, "hash", "field")
			return err
		}},
		{"hgetall", func() error {
			_, err := success.HGetAll(ctx, "hash")
			return err
		}},
		{"hdel", func() error {
			_, err := success.HDel(ctx, "hash", "field")
			return err
		}},
		{"lpush", func() error {
			_, err := success.LPush(ctx, "list", "a")
			return err
		}},
		{"rpush", func() error {
			_, err := success.RPush(ctx, "list", "b")
			return err
		}},
		{"lrange", func() error {
			_, err := success.LRange(ctx, "list", 0, -1)
			return err
		}},
		{"llen", func() error {
			length, err := success.LLen(ctx, "list")
			if length != 2 {
				t.Fatalf("LLen = %d, want 2", length)
			}
			return err
		}},
		{"lpop", func() error {
			_, err := success.LPop(ctx, "list")
			return err
		}},
		{"rpop", func() error {
			_, err := success.RPop(ctx, "list")
			return err
		}},
		{"pipeline", func() error {
			results, err := success.Pipeline(ctx, []PipelineCommand{{Type: PipelineGet, Key: "pipe"}})
			if len(results) != 1 {
				t.Fatalf("Pipeline returned %d results, want 1", len(results))
			}
			return err
		}},
		{"acquire-lock", func() error {
			ok, err := success.AcquireLock(ctx, "lock", "token", time.Second)
			if !ok {
				t.Fatal("AcquireLock returned false, want true")
			}
			return err
		}},
		{"release-lock", func() error {
			ok, err := success.ReleaseLock(ctx, "lock", "token")
			if !ok {
				t.Fatal("ReleaseLock returned false, want true")
			}
			return err
		}},
		{"fixed-window", func() error {
			result, err := success.FixedWindowRateLimit(ctx, "subject", 1, time.Second)
			if result.Allowed {
				t.Fatal("FixedWindowRateLimit returned allowed, want provider result")
			}
			return err
		}},
	}
	for _, tc := range successOps {
		t.Run("success-"+tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("operation error = %v", err)
			}
		})
	}
	if len(successProvider.pipelineCommands) != 1 {
		t.Fatalf("provider saw %d pipeline commands, want 1", len(successProvider.pipelineCommands))
	}

	noMetrics := &Client{initialized: true, provider: &coverageProvider{}}
	if err := noMetrics.Ping(ctx); err != nil {
		t.Fatalf("Ping without metrics returned error: %v", err)
	}
	var nilSetNXClient *Client
	if _, err := nilSetNXClient.SetNX(ctx, "key", "value", time.Second); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("nil SetNX error = %v, want validation", err)
	}

	errorOps := []struct {
		name string
		run  func(*Client) error
	}{
		{"ping", func(c *Client) error { return c.Ping(ctx) }},
		{"get", func(c *Client) error {
			_, err := c.Get(ctx, "key")
			return err
		}},
		{"set", func(c *Client) error { return c.Set(ctx, "key", "value", 0) }},
		{"setnx", func(c *Client) error {
			_, err := c.SetNX(ctx, "key", "value", time.Second)
			return err
		}},
		{"del", func(c *Client) error {
			_, err := c.Del(ctx, "key")
			return err
		}},
		{"exists", func(c *Client) error {
			_, err := c.Exists(ctx, "key")
			return err
		}},
		{"expire", func(c *Client) error {
			_, err := c.Expire(ctx, "key", time.Second)
			return err
		}},
		{"ttl", func(c *Client) error {
			_, err := c.TTL(ctx, "key")
			return err
		}},
		{"mget", func(c *Client) error {
			_, err := c.MGet(ctx, "key")
			return err
		}},
		{"mset", func(c *Client) error { return c.MSet(ctx, map[string]string{"key": "value"}) }},
		{"incr", func(c *Client) error {
			_, err := c.Incr(ctx, "key")
			return err
		}},
		{"decr", func(c *Client) error {
			_, err := c.Decr(ctx, "key")
			return err
		}},
		{"hset", func(c *Client) error {
			_, err := c.HSet(ctx, "hash", map[string]string{"field": "value"})
			return err
		}},
		{"hget", func(c *Client) error {
			_, err := c.HGet(ctx, "hash", "field")
			return err
		}},
		{"hgetall", func(c *Client) error {
			_, err := c.HGetAll(ctx, "hash")
			return err
		}},
		{"hdel", func(c *Client) error {
			_, err := c.HDel(ctx, "hash", "field")
			return err
		}},
		{"lpush", func(c *Client) error {
			_, err := c.LPush(ctx, "list", "a")
			return err
		}},
		{"rpush", func(c *Client) error {
			_, err := c.RPush(ctx, "list", "a")
			return err
		}},
		{"lrange", func(c *Client) error {
			_, err := c.LRange(ctx, "list", 0, -1)
			return err
		}},
		{"llen", func(c *Client) error {
			_, err := c.LLen(ctx, "list")
			return err
		}},
		{"lpop", func(c *Client) error {
			_, err := c.LPop(ctx, "list")
			return err
		}},
		{"rpop", func(c *Client) error {
			_, err := c.RPop(ctx, "list")
			return err
		}},
		{"pipeline", func(c *Client) error {
			_, err := c.Pipeline(ctx, []PipelineCommand{{Type: PipelineGet, Key: "key"}})
			return err
		}},
		{"acquire-lock", func(c *Client) error {
			_, err := c.AcquireLock(ctx, "lock", "token", time.Second)
			return err
		}},
		{"release-lock", func(c *Client) error {
			_, err := c.ReleaseLock(ctx, "lock", "token")
			return err
		}},
		{"fixed-window", func(c *Client) error {
			_, err := c.FixedWindowRateLimit(ctx, "subject", 1, time.Second)
			return err
		}},
	}
	for _, tc := range errorOps {
		t.Run("provider-error-"+tc.name, func(t *testing.T) {
			c := &Client{initialized: true, metrics: &recordingMetrics{}, provider: &coverageProvider{err: internalprovider.ErrNetwork}}
			if err := tc.run(c); !IsKind(err, ErrorKindNetwork) {
				t.Fatalf("operation error = %v, want network", err)
			}
		})
	}

	validationOps := []struct {
		name string
		run  func() error
	}{
		{"get empty", func() error {
			_, err := success.Get(ctx, "")
			return err
		}},
		{"set empty", func() error { return success.Set(ctx, "", "value", 0) }},
		{"setnx empty", func() error {
			_, err := success.SetNX(ctx, "", "value", time.Second)
			return err
		}},
		{"del empty variadic", func() error {
			_, err := success.Del(ctx)
			return err
		}},
		{"del empty key", func() error {
			_, err := success.Del(ctx, "key", "")
			return err
		}},
		{"exists empty variadic", func() error {
			_, err := success.Exists(ctx)
			return err
		}},
		{"exists empty key", func() error {
			_, err := success.Exists(ctx, "")
			return err
		}},
		{"expire empty key", func() error {
			_, err := success.Expire(ctx, "", time.Second)
			return err
		}},
		{"ttl empty key", func() error {
			_, err := success.TTL(ctx, "")
			return err
		}},
		{"mget empty variadic", func() error {
			_, err := success.MGet(ctx)
			return err
		}},
		{"mget empty key", func() error {
			_, err := success.MGet(ctx, "key", "")
			return err
		}},
		{"mset empty", func() error { return success.MSet(ctx, map[string]string{}) }},
		{"mset empty key", func() error { return success.MSet(ctx, map[string]string{"": "value"}) }},
		{"incr empty", func() error {
			_, err := success.Incr(ctx, "")
			return err
		}},
		{"decr empty", func() error {
			_, err := success.Decr(ctx, "")
			return err
		}},
		{"hset empty key", func() error {
			_, err := success.HSet(ctx, "", map[string]string{"field": "value"})
			return err
		}},
		{"hset empty values", func() error {
			_, err := success.HSet(ctx, "hash", nil)
			return err
		}},
		{"hset empty field", func() error {
			_, err := success.HSet(ctx, "hash", map[string]string{"": "value"})
			return err
		}},
		{"hget empty key", func() error {
			_, err := success.HGet(ctx, "", "field")
			return err
		}},
		{"hget empty field", func() error {
			_, err := success.HGet(ctx, "hash", "")
			return err
		}},
		{"hgetall empty key", func() error {
			_, err := success.HGetAll(ctx, "")
			return err
		}},
		{"hdel empty key", func() error {
			_, err := success.HDel(ctx, "", "field")
			return err
		}},
		{"hdel no fields", func() error {
			_, err := success.HDel(ctx, "hash")
			return err
		}},
		{"hdel empty field", func() error {
			_, err := success.HDel(ctx, "hash", "")
			return err
		}},
		{"lpush empty key", func() error {
			_, err := success.LPush(ctx, "", "value")
			return err
		}},
		{"lpush no values", func() error {
			_, err := success.LPush(ctx, "list")
			return err
		}},
		{"rpush empty key", func() error {
			_, err := success.RPush(ctx, "", "value")
			return err
		}},
		{"rpush no values", func() error {
			_, err := success.RPush(ctx, "list")
			return err
		}},
		{"lrange empty key", func() error {
			_, err := success.LRange(ctx, "", 0, -1)
			return err
		}},
		{"lpop empty key", func() error {
			_, err := success.LPop(ctx, "")
			return err
		}},
		{"rpop empty key", func() error {
			_, err := success.RPop(ctx, "")
			return err
		}},
		{"acquire empty key", func() error {
			_, err := success.AcquireLock(ctx, "", "token", time.Second)
			return err
		}},
		{"release empty key", func() error {
			_, err := success.ReleaseLock(ctx, "", "token")
			return err
		}},
		{"fixed-window empty subject", func() error {
			_, err := success.FixedWindowRateLimit(ctx, "", 1, time.Second)
			return err
		}},
	}
	for _, tc := range validationOps {
		t.Run("validation-"+tc.name, func(t *testing.T) {
			if err := tc.run(); !IsKind(err, ErrorKindValidation) {
				t.Fatalf("operation error = %v, want validation", err)
			}
		})
	}
}

func TestCoverageBCacheClientEdges(t *testing.T) {
	ctx := context.Background()
	client, err := New(ctx, Config{Name: "redisx-cache-b"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if _, err := NewCacheClient[string](nil, coverageStringCodec{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("NewCacheClient nil client error = %v, want validation", err)
	}
	if _, err := NewCacheClient[string](client, nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("NewCacheClient nil codec error = %v, want validation", err)
	}

	var nilCache *CacheClient[string]
	if _, err := nilCache.Get(ctx, "key"); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("nil cache Get error = %v, want validation", err)
	}
	if err := nilCache.Set(ctx, "key", "value", time.Minute); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("nil cache Set error = %v, want validation", err)
	}
	if _, err := nilCache.GetOrLoad(ctx, "key", time.Minute, func(context.Context) (string, error) { return "value", nil }); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("nil cache GetOrLoad error = %v, want validation", err)
	}

	cache, err := NewCacheClient[string](client, coverageStringCodec{})
	if err != nil {
		t.Fatalf("NewCacheClient returned error: %v", err)
	}
	if _, err := cache.GetOrLoad(ctx, "key", time.Minute, nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("GetOrLoad nil loader error = %v, want validation", err)
	}
	if _, err := cache.GetOrLoad(ctx, "", time.Minute, func(context.Context) (string, error) { return "value", nil }); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("GetOrLoad invalid key error = %v, want validation", err)
	}

	loadErr := errors.New("load failed")
	if _, err := cache.GetOrLoad(ctx, "cache-load-error-b", time.Minute, func(context.Context) (string, error) { return "", loadErr }); !errors.Is(err, loadErr) {
		t.Fatalf("GetOrLoad loader error = %v, want %v", err, loadErr)
	}

	calls := 0
	value, err := cache.GetOrLoad(ctx, "cache-load-b", time.Minute, func(context.Context) (string, error) {
		calls++
		return "loaded", nil
	})
	if err != nil || value != "loaded" {
		t.Fatalf("GetOrLoad miss = %q, %v; want loaded nil", value, err)
	}
	value, err = cache.GetOrLoad(ctx, "cache-load-b", time.Minute, func(context.Context) (string, error) {
		calls++
		return "should-not-run", nil
	})
	if err != nil || value != "encoded:loaded" || calls != 1 {
		t.Fatalf("GetOrLoad hit = %q, %v calls=%d; want encoded:loaded nil calls=1", value, err, calls)
	}

	encodeErr := errors.New("encode failed")
	badEncode, err := NewCacheClient[string](client, coverageStringCodec{marshalErr: encodeErr})
	if err != nil {
		t.Fatalf("NewCacheClient returned error: %v", err)
	}
	if err := badEncode.Set(ctx, "cache-encode-b", "value", time.Minute); !IsKind(err, ErrorKindValidation) || !errors.Is(err, encodeErr) {
		t.Fatalf("Set encode error = %v, want validation wrapping %v", err, encodeErr)
	}
	if _, err := badEncode.GetOrLoad(ctx, "cache-set-error-b", time.Minute, func(context.Context) (string, error) { return "value", nil }); !IsKind(err, ErrorKindValidation) || !errors.Is(err, encodeErr) {
		t.Fatalf("GetOrLoad set error = %v, want validation wrapping %v", err, encodeErr)
	}

	decodeErr := errors.New("decode failed")
	badDecode, err := NewCacheClient[string](client, coverageStringCodec{unmarshalErr: decodeErr})
	if err != nil {
		t.Fatalf("NewCacheClient returned error: %v", err)
	}
	if err := client.Set(ctx, "cache-decode-b", "raw", time.Minute); err != nil {
		t.Fatalf("seed cache value: %v", err)
	}
	if _, err := badDecode.Get(ctx, "cache-decode-b"); !IsKind(err, ErrorKindValidation) || !errors.Is(err, decodeErr) {
		t.Fatalf("Get decode error = %v, want validation wrapping %v", err, decodeErr)
	}
}

func TestCoverageBGenericCacheHelperAndLocks(t *testing.T) {
	ctx := context.Background()

	var nilCache Cache[string]
	if _, _, err := nilCache.Get(ctx, "key"); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("nil Cache.Get error = %v, want validation", err)
	}
	if err := nilCache.Set(ctx, "key", "value"); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("nil Cache.Set error = %v, want validation", err)
	}

	nilClient := &Client{initialized: true, provider: &coverageProvider{err: internalprovider.ErrNil}}
	value, found, err := (Cache[string]{Client: nilClient}).Get(ctx, "missing-b")
	if err != nil || found || value != "" {
		t.Fatalf("Cache.Get nil provider result = %q found=%v err=%v; want zero false nil", value, found, err)
	}

	client, err := New(ctx, Config{Name: "redisx-generic-cache-b"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	cache := Cache[string]{Client: client, TTL: time.Minute}
	if err := cache.Set(ctx, "generic-cache-b", "stored"); err != nil {
		t.Fatalf("Cache.Set returned error: %v", err)
	}
	value, found, err = cache.Get(ctx, "generic-cache-b")
	if err != nil || !found || value != "stored" {
		t.Fatalf("Cache.Get = %q found=%v err=%v; want stored true nil", value, found, err)
	}

	if _, err := cache.GetOrLoad(ctx, "generic-cache-loader-b", nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("Cache.GetOrLoad nil loader error = %v, want validation", err)
	}
	loadErr := errors.New("generic load failed")
	if _, err := cache.GetOrLoad(ctx, "generic-cache-load-error-b", func(context.Context) (string, error) {
		return "", loadErr
	}); !errors.Is(err, loadErr) {
		t.Fatalf("Cache.GetOrLoad loader error = %v, want %v", err, loadErr)
	}

	calls := 0
	value, err = cache.GetOrLoad(ctx, "generic-cache-load-b", func(context.Context) (string, error) {
		calls++
		return "loaded", nil
	})
	if err != nil || value != "loaded" {
		t.Fatalf("Cache.GetOrLoad miss = %q, %v; want loaded nil", value, err)
	}
	value, err = cache.GetOrLoad(ctx, "generic-cache-load-b", func(context.Context) (string, error) {
		calls++
		return "should-not-run", nil
	})
	if err != nil || value != "loaded" || calls != 1 {
		t.Fatalf("Cache.GetOrLoad hit = %q, %v calls=%d; want loaded nil calls=1", value, err, calls)
	}

	errClient := &Client{initialized: true, provider: &coverageProvider{err: internalprovider.ErrNetwork}}
	if _, _, err := (Cache[string]{Client: errClient}).Get(ctx, "network-b"); !IsKind(err, ErrorKindNetwork) {
		t.Fatalf("Cache.Get provider error = %v, want network", err)
	}
	if _, err := (Cache[string]{Client: errClient}).GetOrLoad(ctx, "network-b", func(context.Context) (string, error) {
		return "value", nil
	}); !IsKind(err, ErrorKindNetwork) {
		t.Fatalf("Cache.GetOrLoad provider error = %v, want network", err)
	}

	encodeErr := errors.New("generic encode failed")
	badEncode := Cache[string]{Client: client, Codec: coverageStringCodec{marshalErr: encodeErr}}
	if err := badEncode.Set(ctx, "generic-encode-b", "value"); !IsKind(err, ErrorKindValidation) || !errors.Is(err, encodeErr) {
		t.Fatalf("Cache.Set encode error = %v, want validation wrapping %v", err, encodeErr)
	}
	if _, err := badEncode.GetOrLoad(ctx, "generic-set-error-b", func(context.Context) (string, error) {
		return "value", nil
	}); !IsKind(err, ErrorKindValidation) || !errors.Is(err, encodeErr) {
		t.Fatalf("Cache.GetOrLoad set error = %v, want validation wrapping %v", err, encodeErr)
	}

	decodeErr := errors.New("generic decode failed")
	if err := client.Set(ctx, "generic-decode-b", "raw", time.Minute); err != nil {
		t.Fatalf("seed generic decode value: %v", err)
	}
	badDecode := Cache[string]{Client: client, Codec: coverageStringCodec{unmarshalErr: decodeErr}}
	if _, _, err := badDecode.Get(ctx, "generic-decode-b"); !IsKind(err, ErrorKindValidation) || !errors.Is(err, decodeErr) {
		t.Fatalf("Cache.Get decode error = %v, want validation wrapping %v", err, decodeErr)
	}

	var nilLock *Lock
	if token := nilLock.Token(); token != "" {
		t.Fatalf("nil Lock.Token = %q, want empty", token)
	}
	if _, err := nilLock.Release(ctx); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("nil Lock.Release error = %v, want validation", err)
	}
	if _, err := (&Lock{}).Release(ctx); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("empty Lock.Release error = %v, want validation", err)
	}

	lockClient := &Client{initialized: true, provider: &coverageProvider{}}
	if _, _, err := lockClient.NewLock(ctx, "lock-invalid-b", 0); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("NewLock zero ttl error = %v, want validation", err)
	}
	lock, acquired, err := lockClient.NewLock(ctx, "lock-helper-b", time.Minute)
	if err != nil || !acquired || lock == nil || lock.Token() == "" {
		t.Fatalf("NewLock = lock=%#v acquired=%v err=%v; want acquired lock", lock, acquired, err)
	}
	released, err := lock.Release(ctx)
	if err != nil || !released {
		t.Fatalf("Lock.Release = %v, %v; want true nil", released, err)
	}

	notAcquired := &Client{initialized: true, provider: &coverageProvider{acquireLockSet: true, acquireLock: false}}
	lock, acquired, err = notAcquired.NewLock(ctx, "lock-not-acquired-b", time.Minute)
	if err != nil || acquired || lock != nil {
		t.Fatalf("NewLock not acquired = lock=%#v acquired=%v err=%v; want nil false nil", lock, acquired, err)
	}

	lockErrClient := &Client{initialized: true, provider: &coverageProvider{err: internalprovider.ErrNetwork}}
	if _, _, err := lockErrClient.NewLock(ctx, "lock-error-b", time.Minute); !IsKind(err, ErrorKindNetwork) {
		t.Fatalf("NewLock provider error = %v, want network", err)
	}
}

func TestCoverageBRateLimiterHelperAndHealth(t *testing.T) {
	ctx := context.Background()
	var nilCtx context.Context

	var emptyLimiter FixedWindowRateLimiter
	if _, err := emptyLimiter.Allow(ctx, "subject"); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("empty limiter error = %v, want validation", err)
	}

	baseClient := &Client{initialized: true, provider: &coverageProvider{}, metrics: &recordingMetrics{}}
	validationLimiters := []struct {
		name    string
		limiter FixedWindowRateLimiter
		subject string
	}{
		{"subject", FixedWindowRateLimiter{Client: baseClient, Limit: 1, Window: time.Minute}, ""},
		{"limit", FixedWindowRateLimiter{Client: baseClient, Limit: 0, Window: time.Minute}, "subject"},
		{"window", FixedWindowRateLimiter{Client: baseClient, Limit: 1, Window: 0}, "subject"},
	}
	for _, tc := range validationLimiters {
		t.Run("limiter-validation-"+tc.name, func(t *testing.T) {
			if _, err := tc.limiter.Allow(ctx, tc.subject); !IsKind(err, ErrorKindValidation) {
				t.Fatalf("Allow error = %v, want validation", err)
			}
		})
	}

	fixedClock := func() time.Time { return time.Unix(10, 0) }
	result, err := (FixedWindowRateLimiter{
		Client: baseClient,
		Limit:  2,
		Window: time.Minute,
		Clock:  fixedClock,
	}).Allow(ctx, "subject-b")
	if err != nil || !result.Allowed || result.Count != 1 || result.Remaining != 1 || result.Limit != 2 || result.ResetAfter <= 0 {
		t.Fatalf("Allow result = %#v, %v; want allowed count=1 remaining=1", result, err)
	}

	result, err = (FixedWindowRateLimiter{
		Client: baseClient,
		Limit:  2,
		Window: time.Minute,
	}).Allow(ctx, "subject-default-clock-b")
	if err != nil || !result.Allowed || result.Count != 1 || result.Remaining != 1 || result.Limit != 2 || result.ResetAfter <= 0 {
		t.Fatalf("Allow default clock result = %#v, %v; want allowed count=1 remaining=1", result, err)
	}

	overClient := &Client{initialized: true, provider: &coverageProvider{incrValue: 3}}
	result, err = (FixedWindowRateLimiter{
		Client: overClient,
		Limit:  1,
		Window: time.Minute,
		Clock:  fixedClock,
	}).Allow(ctx, "subject-over-b")
	if err != nil || result.Allowed || result.Count != 3 || result.Remaining != 0 {
		t.Fatalf("over-limit result = %#v, %v; want denied count=3 remaining=0", result, err)
	}

	prefixClient := &Client{initialized: true, provider: &coverageProvider{incrValue: 2}}
	result, err = (FixedWindowRateLimiter{
		Client: prefixClient,
		Prefix: "custom-prefix-b",
		Limit:  3,
		Window: time.Minute,
		Clock:  fixedClock,
	}).Allow(ctx, "subject-prefix-b")
	if err != nil || !result.Allowed || result.Count != 2 || result.Remaining != 1 {
		t.Fatalf("custom-prefix result = %#v, %v; want allowed count=2 remaining=1", result, err)
	}

	expireErrClient := &Client{initialized: true, provider: &coverageProvider{expireErr: internalprovider.ErrNetwork}}
	if _, err := (FixedWindowRateLimiter{Client: expireErrClient, Limit: 2, Window: time.Minute, Clock: fixedClock}).Allow(ctx, "expire-error-b"); !IsKind(err, ErrorKindNetwork) {
		t.Fatalf("Allow expire error = %v, want network", err)
	}

	incrErrClient := &Client{initialized: true, provider: &coverageProvider{err: internalprovider.ErrNetwork}}
	if _, err := (FixedWindowRateLimiter{Client: incrErrClient, Limit: 2, Window: time.Minute, Clock: fixedClock}).Allow(ctx, "incr-error-b"); !IsKind(err, ErrorKindNetwork) {
		t.Fatalf("Allow incr error = %v, want network", err)
	}

	var nilClient *Client
	status := nilClient.Health(ctx)
	if status.Status != HealthUnhealthy || status.ErrorClass != string(ErrorKindValidation) {
		t.Fatalf("nil client health = %#v, want unhealthy validation", status)
	}

	metrics := &recordingMetrics{}
	healthClient := &Client{
		cfg:         Config{Name: "health-b", Timeout: time.Hour},
		initialized: true,
		provider:    &coverageProvider{},
		metrics:     metrics,
	}
	shortCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	status = healthClient.Health(shortCtx)
	if status.Status != HealthDegraded || status.Metadata["reason"] != "deadline_below_timeout" {
		t.Fatalf("short-deadline health = %#v, want degraded deadline metadata", status)
	}
	if !metrics.hasGauge(MetricClientHealthStatus) || !metrics.hasHistogram(MetricClientHealthLatencyMS) {
		t.Fatalf("expected health gauges and histogram, got %#v", metrics)
	}

	status = healthClient.Health(nilCtx)
	if status.Status != HealthUnhealthy || status.ErrorClass != string(ErrorKindValidation) {
		t.Fatalf("nil context health = %#v, want unhealthy validation", status)
	}

	canceledCtx, cancelCanceled := context.WithCancel(ctx)
	cancelCanceled()
	status = healthClient.Health(canceledCtx)
	if status.Status != HealthUnhealthy || status.ErrorClass != string(ErrorKindCanceled) {
		t.Fatalf("canceled context health = %#v, want unhealthy canceled", status)
	}

	closedClient := &Client{
		cfg:         Config{Name: "closed-health-b"},
		initialized: true,
		closed:      true,
		provider:    &coverageProvider{},
		metrics:     &recordingMetrics{},
	}
	status = closedClient.Health(ctx)
	if status.Status != HealthUnhealthy || status.ErrorClass != string(ErrorKindClosed) {
		t.Fatalf("closed client health = %#v, want unhealthy closed", status)
	}

	failingHealth := &Client{
		cfg:         Config{Name: "health-fail-b"},
		initialized: true,
		provider:    &coverageProvider{err: internalprovider.ErrAuth},
		metrics:     &recordingMetrics{},
	}
	status = failingHealth.Health(ctx)
	if status.Status != HealthUnhealthy || status.ErrorClass != string(ErrorKindAuth) {
		t.Fatalf("provider error health = %#v, want unhealthy auth", status)
	}

	healthyClient := &Client{initialized: true, provider: &coverageProvider{}, metrics: &recordingMetrics{}}
	status = healthyClient.Health(ctx)
	if status.Status != HealthHealthy || status.Name != "redisx" || status.Message != "ok" {
		t.Fatalf("healthy status = %#v, want default-name healthy ok", status)
	}
	if alias := healthyClient.HealthCheck(ctx); alias.Status != HealthHealthy {
		t.Fatalf("HealthCheck status = %#v, want healthy", alias)
	}
}

func TestCoverageBJSONCodecErrors(t *testing.T) {
	if _, err := (JSONCodec[chan int]{}).Marshal(make(chan int)); err == nil {
		t.Fatal("JSONCodec Marshal of channel returned nil error")
	}
	if _, err := (JSONCodec[map[string]string]{}).Unmarshal("{"); err == nil {
		t.Fatal("JSONCodec Unmarshal of invalid JSON returned nil error")
	}
}

func TestCoverageBNewRedisProviderConstructionError(t *testing.T) {
	original := newRedisProvider
	defer func() { newRedisProvider = original }()
	newRedisProvider = func(redisprovider.Config) (Provider, error) {
		return nil, errors.New("constructor failed")
	}

	_, err := New(context.Background(), Config{
		Name: "redis-provider-constructor-b",
		Redis: RedisConfig{
			Addr: "127.0.0.1:6379",
		},
	})
	if !IsKind(err, ErrorKindInvalidConfig) {
		t.Fatalf("New redis provider constructor error = %v, want invalid config", err)
	}
}

func TestCoverageBNewLockTokenReadError(t *testing.T) {
	original := readLockTokenRandom
	defer func() { readLockTokenRandom = original }()
	readLockTokenRandom = func([]byte) (int, error) {
		return 0, errors.New("entropy unavailable")
	}

	client := &Client{initialized: true, provider: &coverageProvider{}, metrics: NoopMetrics{}}
	lock, acquired, err := client.NewLock(context.Background(), "lock-token-error-b", time.Second)
	if lock != nil || acquired || !IsKind(err, ErrorKindInternal) {
		t.Fatalf("NewLock token error = lock=%#v acquired=%v err=%v, want nil false internal", lock, acquired, err)
	}
}

func TestCoverageBFixedWindowResetAfterClamp(t *testing.T) {
	now := time.Unix(10, 0)
	if got := fixedWindowResetAfter(time.Second, now, 0); got != 0 {
		t.Fatalf("fixedWindowResetAfter past bucket = %s, want 0", got)
	}
	bucket := now.UnixNano() / int64(time.Second)
	if got := fixedWindowResetAfter(time.Second, now, bucket); got <= 0 {
		t.Fatalf("fixedWindowResetAfter current bucket = %s, want positive duration", got)
	}
}
