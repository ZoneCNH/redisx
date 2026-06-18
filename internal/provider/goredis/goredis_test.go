package goredis

import (
	"context"
	"errors"
	"net"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ZoneCNH/redisx/internal/provider"
	"github.com/redis/go-redis/v9"
)

type fakeRedisHook struct {
	process  func(context.Context, redis.Cmder) error
	pipeline func(context.Context, []redis.Cmder) error
}

func (h fakeRedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("unexpected redis dial")
	}
}

func (h fakeRedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if h.process != nil {
			return h.process(ctx, cmd)
		}
		return nil
	}
}

func (h fakeRedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		if h.pipeline != nil {
			return h.pipeline(ctx, cmds)
		}
		return nil
	}
}

type fakeNetError struct {
	timeout bool
}

func (e fakeNetError) Error() string {
	return "fake net error"
}

func (e fakeNetError) Timeout() bool {
	return e.timeout
}

func (e fakeNetError) Temporary() bool {
	return false
}

func newHookProvider(t *testing.T, hook fakeRedisHook) *Provider {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "hook.invalid:6379"})
	client.AddHook(hook)
	p := &Provider{client: client}
	t.Cleanup(func() {
		if p.client != nil {
			_ = p.client.Close()
		}
	})
	return p
}

func requireErrIs(t *testing.T, err error, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("error = %v, want %v", err, target)
	}
}

func requireNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func setCommandValue(t *testing.T, cmd redis.Cmder, value any) {
	t.Helper()
	switch c := cmd.(type) {
	case *redis.StatusCmd:
		c.SetVal(value.(string))
	case *redis.StringCmd:
		c.SetVal(value.(string))
	case *redis.BoolCmd:
		c.SetVal(value.(bool))
	case *redis.IntCmd:
		c.SetVal(value.(int64))
	case *redis.DurationCmd:
		c.SetVal(value.(time.Duration))
	case *redis.SliceCmd:
		c.SetVal(value.([]any))
	case *redis.MapStringStringCmd:
		c.SetVal(value.(map[string]string))
	case *redis.StringSliceCmd:
		c.SetVal(value.([]string))
	case *redis.Cmd:
		c.SetVal(value)
	default:
		t.Fatalf("unexpected command type %T for %s", cmd, cmd.Name())
	}
}

func TestConfigValidateAndNew(t *testing.T) {
	valid := Config{
		Addr:         "localhost:6379",
		Username:     "user",
		Password:     "pass",
		DB:           2,
		DialTimeout:  time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     4,
		MinIdleConns: 1,
		MaxRetries:   5,
	}
	requireNoErr(t, valid.Validate())

	p, err := New(valid)
	requireNoErr(t, err)
	if got := p.client.Options(); got.Addr != valid.Addr || got.Username != valid.Username || got.Password != valid.Password || got.DB != valid.DB || got.MaxRetries != valid.MaxRetries {
		t.Fatalf("New options = %#v, want config applied", got)
	}
	requireNoErr(t, p.Close(context.Background()))

	tests := []Config{
		{},
		{Addr: "   "},
		{Addr: "localhost:6379", DB: -1},
		{Addr: "localhost:6379", DialTimeout: -time.Nanosecond},
		{Addr: "localhost:6379", ReadTimeout: -time.Nanosecond},
		{Addr: "localhost:6379", WriteTimeout: -time.Nanosecond},
		{Addr: "localhost:6379", PoolSize: -1},
		{Addr: "localhost:6379", MinIdleConns: -1},
		{Addr: "localhost:6379", MaxRetries: -1},
	}
	for _, cfg := range tests {
		if err := cfg.Validate(); err == nil {
			t.Fatalf("Validate(%#v) = nil, want error", cfg)
		}
		if _, err := New(cfg); err == nil {
			t.Fatalf("New(%#v) = nil error, want validation error", cfg)
		}
	}
}

func TestProviderCommandsUseRedisClientResults(t *testing.T) {
	ctx := context.Background()
	p := newHookProvider(t, fakeRedisHook{process: func(ctx context.Context, cmd redis.Cmder) error {
		switch cmd.Name() {
		case "ping", "mset":
			setCommandValue(t, cmd, "OK")
		case "set":
			if _, ok := cmd.(*redis.BoolCmd); ok {
				setCommandValue(t, cmd, true)
			} else {
				setCommandValue(t, cmd, "OK")
			}
		case "get":
			setCommandValue(t, cmd, "value")
		case "setnx", "expire":
			setCommandValue(t, cmd, true)
		case "del":
			setCommandValue(t, cmd, int64(2))
		case "exists":
			setCommandValue(t, cmd, int64(3))
		case "ttl":
			setCommandValue(t, cmd, redisTTLNoExpire)
		case "mget":
			setCommandValue(t, cmd, []any{"one", []byte("two"), int64(3), nil})
		case "incr":
			setCommandValue(t, cmd, int64(11))
		case "decr":
			setCommandValue(t, cmd, int64(10))
		case "hset":
			setCommandValue(t, cmd, int64(2))
		case "hget":
			setCommandValue(t, cmd, "hash-value")
		case "hgetall":
			setCommandValue(t, cmd, map[string]string{"a": "1", "b": "2"})
		case "hdel":
			setCommandValue(t, cmd, int64(1))
		case "lpush":
			setCommandValue(t, cmd, int64(2))
		case "rpush":
			setCommandValue(t, cmd, int64(3))
		case "lrange":
			setCommandValue(t, cmd, []string{"a", "b"})
		case "llen":
			setCommandValue(t, cmd, int64(2))
		case "lpop":
			setCommandValue(t, cmd, "a")
		case "rpop":
			setCommandValue(t, cmd, "b")
		default:
			t.Fatalf("unexpected command %q args %#v", cmd.Name(), cmd.Args())
		}
		return nil
	}})

	requireNoErr(t, p.Ping(ctx))
	if got, err := p.Get(ctx, "k"); err != nil || got != "value" {
		t.Fatalf("Get = %q, %v", got, err)
	}
	requireNoErr(t, p.Set(ctx, "k", "value", time.Second))
	if got, err := p.SetNX(ctx, "k", "value", time.Second); err != nil || !got {
		t.Fatalf("SetNX = %v, %v", got, err)
	}
	if got, err := p.Del(ctx, "a", "b"); err != nil || got != 2 {
		t.Fatalf("Del = %d, %v", got, err)
	}
	if got, err := p.Exists(ctx, "a", "b", "c"); err != nil || got != 3 {
		t.Fatalf("Exists = %d, %v", got, err)
	}
	if got, err := p.Expire(ctx, "k", time.Minute); err != nil || !got {
		t.Fatalf("Expire = %v, %v", got, err)
	}
	if got, err := p.TTL(ctx, "k"); err != nil || got != -time.Second {
		t.Fatalf("TTL = %s, %v", got, err)
	}
	values, err := p.MGet(ctx, "a", "b", "c", "missing")
	requireNoErr(t, err)
	wantValues := []provider.Value{
		{Value: "one", Found: true},
		{Value: "two", Found: true},
		{Value: "3", Found: true},
		{},
	}
	if !reflect.DeepEqual(values, wantValues) {
		t.Fatalf("MGet = %#v, want %#v", values, wantValues)
	}
	requireNoErr(t, p.MSet(ctx, map[string]string{"a": "1", "b": "2"}))
	if got, err := p.Incr(ctx, "n"); err != nil || got != 11 {
		t.Fatalf("Incr = %d, %v", got, err)
	}
	if got, err := p.Decr(ctx, "n"); err != nil || got != 10 {
		t.Fatalf("Decr = %d, %v", got, err)
	}
	if got, err := p.HSet(ctx, "h", map[string]string{"a": "1"}); err != nil || got != 2 {
		t.Fatalf("HSet = %d, %v", got, err)
	}
	if got, err := p.HGet(ctx, "h", "a"); err != nil || got != "hash-value" {
		t.Fatalf("HGet = %q, %v", got, err)
	}
	hash, err := p.HGetAll(ctx, "h")
	requireNoErr(t, err)
	if !reflect.DeepEqual(hash, map[string]string{"a": "1", "b": "2"}) {
		t.Fatalf("HGetAll = %#v", hash)
	}
	if got, err := p.HDel(ctx, "h", "a"); err != nil || got != 1 {
		t.Fatalf("HDel = %d, %v", got, err)
	}
	if got, err := p.LPush(ctx, "l", "a", "b"); err != nil || got != 2 {
		t.Fatalf("LPush = %d, %v", got, err)
	}
	if got, err := p.RPush(ctx, "l", "c"); err != nil || got != 3 {
		t.Fatalf("RPush = %d, %v", got, err)
	}
	if got, err := p.LRange(ctx, "l", 0, -1); err != nil || !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("LRange = %#v, %v", got, err)
	}
	if got, err := p.LLen(ctx, "l"); err != nil || got != 2 {
		t.Fatalf("LLen = %d, %v", got, err)
	}
	if got, err := p.LPop(ctx, "l"); err != nil || got != "a" {
		t.Fatalf("LPop = %q, %v", got, err)
	}
	if got, err := p.RPop(ctx, "l"); err != nil || got != "b" {
		t.Fatalf("RPop = %q, %v", got, err)
	}
	if got, err := p.AcquireLock(ctx, "lock", "token", time.Second); err != nil || !got {
		t.Fatalf("AcquireLock = %v, %v", got, err)
	}
}

func TestProviderCommandErrorsAreMapped(t *testing.T) {
	ctx := context.Background()
	p := newHookProvider(t, fakeRedisHook{process: func(context.Context, redis.Cmder) error {
		return errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")
	}})
	checks := []struct {
		name string
		call func() error
	}{
		{name: "Ping", call: func() error { return p.Ping(ctx) }},
		{name: "Get", call: func() error { _, err := p.Get(ctx, "k"); return err }},
		{name: "Set", call: func() error { return p.Set(ctx, "k", "v", 0) }},
		{name: "SetNX", call: func() error { _, err := p.SetNX(ctx, "k", "v", 0); return err }},
		{name: "Del", call: func() error { _, err := p.Del(ctx, "k"); return err }},
		{name: "Exists", call: func() error { _, err := p.Exists(ctx, "k"); return err }},
		{name: "Expire", call: func() error { _, err := p.Expire(ctx, "k", time.Second); return err }},
		{name: "TTL", call: func() error { _, err := p.TTL(ctx, "k"); return err }},
		{name: "MGet", call: func() error { _, err := p.MGet(ctx, "k"); return err }},
		{name: "MSet", call: func() error { return p.MSet(ctx, map[string]string{"k": "v"}) }},
		{name: "Incr", call: func() error { _, err := p.Incr(ctx, "k"); return err }},
		{name: "Decr", call: func() error { _, err := p.Decr(ctx, "k"); return err }},
		{name: "HSet", call: func() error { _, err := p.HSet(ctx, "h", map[string]string{"f": "v"}); return err }},
		{name: "HGet", call: func() error { _, err := p.HGet(ctx, "h", "f"); return err }},
		{name: "HGetAll", call: func() error { _, err := p.HGetAll(ctx, "h"); return err }},
		{name: "HDel", call: func() error { _, err := p.HDel(ctx, "h", "f"); return err }},
		{name: "LPush", call: func() error { _, err := p.LPush(ctx, "l", "a"); return err }},
		{name: "RPush", call: func() error { _, err := p.RPush(ctx, "l", "a"); return err }},
		{name: "LRange", call: func() error { _, err := p.LRange(ctx, "l", 0, -1); return err }},
		{name: "LLen", call: func() error { _, err := p.LLen(ctx, "l"); return err }},
		{name: "LPop", call: func() error { _, err := p.LPop(ctx, "l"); return err }},
		{name: "RPop", call: func() error { _, err := p.RPop(ctx, "l"); return err }},
		{name: "AcquireLock", call: func() error { _, err := p.AcquireLock(ctx, "lock", "token", time.Second); return err }},
		{name: "ReleaseLock", call: func() error { _, err := p.ReleaseLock(ctx, "lock", "token"); return err }},
		{name: "FixedWindowRateLimit", call: func() error {
			_, err := p.FixedWindowRateLimit(ctx, "rate", 1, time.Second)
			return err
		}},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			requireErrIs(t, check.call(), provider.ErrWrongType)
		})
	}
}

func TestProviderNilResultsAreMapped(t *testing.T) {
	ctx := context.Background()
	p := newHookProvider(t, fakeRedisHook{process: func(context.Context, redis.Cmder) error {
		return redis.Nil
	}})
	for name, call := range map[string]func() error{
		"Get":  func() error { _, err := p.Get(ctx, "k"); return err },
		"HGet": func() error { _, err := p.HGet(ctx, "h", "f"); return err },
		"LPop": func() error { _, err := p.LPop(ctx, "l"); return err },
		"RPop": func() error { _, err := p.RPop(ctx, "l"); return err },
	} {
		t.Run(name, func(t *testing.T) {
			requireErrIs(t, call(), provider.ErrNil)
		})
	}
}

func TestProviderPipeline(t *testing.T) {
	ctx := context.Background()
	p := newHookProvider(t, fakeRedisHook{pipeline: func(ctx context.Context, cmds []redis.Cmder) error {
		for _, cmd := range cmds {
			args := cmd.Args()
			switch cmd.Name() {
			case "set", "mset":
				setCommandValue(t, cmd, "OK")
			case "get":
				if len(args) > 1 && args[1] == "missing" {
					cmd.SetErr(redis.Nil)
					continue
				}
				setCommandValue(t, cmd, "value")
			case "hset":
				setCommandValue(t, cmd, int64(2))
			case "hget":
				setCommandValue(t, cmd, "field-value")
			case "rpush":
				setCommandValue(t, cmd, int64(3))
			case "lrange":
				setCommandValue(t, cmd, []string{"a", "b"})
			case "incr":
				setCommandValue(t, cmd, int64(4))
			default:
				t.Fatalf("unexpected pipeline command %q args %#v", cmd.Name(), args)
			}
		}
		return redis.Nil
	}})

	results, err := p.Pipeline(ctx, []provider.PipelineCommand{
		{Type: provider.PipelineSet, Key: "k", Value: "v", TTL: time.Second},
		{Type: provider.PipelineMSet, Values: map[string]string{"a": "1"}},
		{Type: provider.PipelineGet, Key: "k"},
		{Type: provider.PipelineGet, Key: "missing"},
		{Type: provider.PipelineHSet, Key: "h", Values: map[string]string{"f": "v"}},
		{Type: provider.PipelineHGet, Key: "h", Field: "f"},
		{Type: provider.PipelineRPush, Key: "l", ListValues: []string{"a", "b"}},
		{Type: provider.PipelineLRange, Key: "l", Start: 0, Stop: -1},
		{Type: provider.PipelineIncr, Key: "n"},
	})
	requireNoErr(t, err)
	if !results[0].Bool || !results[1].Bool || !results[2].Found || results[2].Value != "value" || results[3].Found {
		t.Fatalf("unexpected string pipeline results: %#v", results[:4])
	}
	if results[4].Count != 2 || results[4].Int != 2 || !results[5].Found || results[5].Value != "field-value" {
		t.Fatalf("unexpected hash pipeline results: %#v", results[4:6])
	}
	if results[6].Count != 3 || !reflect.DeepEqual(results[7].Strings, []string{"a", "b"}) || results[8].Int != 4 {
		t.Fatalf("unexpected list/incr pipeline results: %#v", results[6:])
	}
	if !reflect.DeepEqual(results[2].Values, []provider.Value{{Value: "value", Found: true}}) {
		t.Fatalf("GET Values compatibility field = %#v", results[2].Values)
	}

	_, err = p.Pipeline(ctx, []provider.PipelineCommand{{Kind: "unknown"}})
	if err == nil || !strings.Contains(err.Error(), "unsupported pipeline command") {
		t.Fatalf("unsupported pipeline error = %v", err)
	}

	fail := newHookProvider(t, fakeRedisHook{pipeline: func(context.Context, []redis.Cmder) error {
		return errors.New("READONLY You can't write against a read only replica")
	}})
	_, err = fail.Pipeline(ctx, []provider.PipelineCommand{{Type: provider.PipelineGet, Key: "k"}})
	requireErrIs(t, err, provider.ErrReadOnly)

	for _, tt := range []struct {
		command provider.PipelineCommand
		want    error
	}{
		{command: provider.PipelineCommand{Type: provider.PipelineSet, Key: "k", Value: "v"}, want: provider.ErrReadOnly},
		{command: provider.PipelineCommand{Type: provider.PipelineMSet, Values: map[string]string{"k": "v"}}, want: provider.ErrReadOnly},
		{command: provider.PipelineCommand{Type: provider.PipelineGet, Key: "k"}, want: provider.ErrLoading},
		{command: provider.PipelineCommand{Type: provider.PipelineHGet, Key: "h", Field: "f"}, want: provider.ErrLoading},
		{command: provider.PipelineCommand{Type: provider.PipelineHSet, Key: "h", Values: map[string]string{"f": "v"}}, want: provider.ErrInvalidInt},
		{command: provider.PipelineCommand{Type: provider.PipelineRPush, Key: "l", ListValues: []string{"a"}}, want: provider.ErrInvalidInt},
		{command: provider.PipelineCommand{Type: provider.PipelineIncr, Key: "n"}, want: provider.ErrInvalidInt},
		{command: provider.PipelineCommand{Type: provider.PipelineLRange, Key: "l", Start: 0, Stop: -1}, want: provider.ErrWrongType},
	} {
		t.Run(string(provider.PipelineCommandKind(tt.command))+" command error", func(t *testing.T) {
			cmdErr := errors.New("READONLY You can't write against a read only replica")
			switch {
			case errors.Is(tt.want, provider.ErrLoading):
				cmdErr = errors.New("LOADING Redis is loading the dataset in memory")
			case errors.Is(tt.want, provider.ErrInvalidInt):
				cmdErr = errors.New("ERR value is not an integer or out of range")
			case errors.Is(tt.want, provider.ErrWrongType):
				cmdErr = errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")
			}
			p := newHookProvider(t, fakeRedisHook{pipeline: func(ctx context.Context, cmds []redis.Cmder) error {
				cmds[0].SetErr(cmdErr)
				return redis.Nil
			}})
			_, err := p.Pipeline(ctx, []provider.PipelineCommand{tt.command})
			requireErrIs(t, err, tt.want)
		})
	}
}

func TestProviderScripts(t *testing.T) {
	ctx := context.Background()

	release := newHookProvider(t, fakeRedisHook{process: func(ctx context.Context, cmd redis.Cmder) error {
		setCommandValue(t, cmd, int64(1))
		return nil
	}})
	if got, err := release.ReleaseLock(ctx, "lock", "token"); err != nil || !got {
		t.Fatalf("ReleaseLock = %v, %v", got, err)
	}

	notReleased := newHookProvider(t, fakeRedisHook{process: func(ctx context.Context, cmd redis.Cmder) error {
		setCommandValue(t, cmd, int64(0))
		return nil
	}})
	if got, err := notReleased.ReleaseLock(ctx, "lock", "token"); err != nil || got {
		t.Fatalf("ReleaseLock zero = %v, %v", got, err)
	}

	badRelease := newHookProvider(t, fakeRedisHook{process: func(ctx context.Context, cmd redis.Cmder) error {
		setCommandValue(t, cmd, struct{}{})
		return nil
	}})
	if _, err := badRelease.ReleaseLock(ctx, "lock", "token"); err == nil || !strings.Contains(err.Error(), "unexpected integer") {
		t.Fatalf("ReleaseLock bad integer error = %v", err)
	}

	authFailure := newHookProvider(t, fakeRedisHook{process: func(context.Context, redis.Cmder) error {
		return errors.New("NOAUTH Authentication required")
	}})
	_, err := authFailure.ReleaseLock(ctx, "lock", "token")
	requireErrIs(t, err, provider.ErrAuth)

	rate := newHookProvider(t, fakeRedisHook{process: func(ctx context.Context, cmd redis.Cmder) error {
		setCommandValue(t, cmd, []any{int64(3), int64(250)})
		return nil
	}})
	result, err := rate.FixedWindowRateLimit(ctx, "rate", 5, time.Second)
	requireNoErr(t, err)
	if !result.Allowed || result.Count != 3 || result.Remaining != 2 || result.ResetAfter != 250*time.Millisecond || result.Limit != 5 {
		t.Fatalf("FixedWindowRateLimit = %#v", result)
	}

	over := newHookProvider(t, fakeRedisHook{process: func(ctx context.Context, cmd redis.Cmder) error {
		setCommandValue(t, cmd, []any{int64(7), int64(-1)})
		return nil
	}})
	result, err = over.FixedWindowRateLimit(ctx, "rate", 5, 0)
	requireNoErr(t, err)
	if result.Allowed || result.Remaining != 0 || result.ResetAfter != 0 {
		t.Fatalf("FixedWindowRateLimit over = %#v", result)
	}

	for name, raw := range map[string]any{
		"wrong result type": "bad",
		"wrong count type":  []any{struct{}{}, int64(1)},
		"wrong ttl type":    []any{int64(1), struct{}{}},
		"wrong arity":       []any{int64(1)},
	} {
		t.Run(name, func(t *testing.T) {
			p := newHookProvider(t, fakeRedisHook{process: func(ctx context.Context, cmd redis.Cmder) error {
				setCommandValue(t, cmd, raw)
				return nil
			}})
			if _, err := p.FixedWindowRateLimit(ctx, "rate", 1, time.Second); err == nil {
				t.Fatal("FixedWindowRateLimit bad script result = nil error")
			}
		})
	}
}

func TestProviderContextAndClosedErrors(t *testing.T) {
	ctx := context.Background()
	var nilCtx context.Context
	p := newHookProvider(t, fakeRedisHook{})

	if err := p.Ping(nilCtx); err == nil || !strings.Contains(err.Error(), "context is required") {
		t.Fatalf("Ping nil context error = %v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	requireErrIs(t, p.Ping(canceled), context.Canceled)
	contextChecks := []struct {
		name string
		call func(context.Context) error
	}{
		{name: "Close", call: func(ctx context.Context) error { return p.Close(ctx) }},
		{name: "Get", call: func(ctx context.Context) error { _, err := p.Get(ctx, "k"); return err }},
		{name: "Set", call: func(ctx context.Context) error { return p.Set(ctx, "k", "v", 0) }},
		{name: "SetNX", call: func(ctx context.Context) error { _, err := p.SetNX(ctx, "k", "v", 0); return err }},
		{name: "Del", call: func(ctx context.Context) error { _, err := p.Del(ctx, "k"); return err }},
		{name: "Exists", call: func(ctx context.Context) error { _, err := p.Exists(ctx, "k"); return err }},
		{name: "Expire", call: func(ctx context.Context) error { _, err := p.Expire(ctx, "k", time.Second); return err }},
		{name: "TTL", call: func(ctx context.Context) error { _, err := p.TTL(ctx, "k"); return err }},
		{name: "MGet", call: func(ctx context.Context) error { _, err := p.MGet(ctx, "k"); return err }},
		{name: "MSet", call: func(ctx context.Context) error { return p.MSet(ctx, map[string]string{"k": "v"}) }},
		{name: "Incr", call: func(ctx context.Context) error { _, err := p.Incr(ctx, "k"); return err }},
		{name: "Decr", call: func(ctx context.Context) error { _, err := p.Decr(ctx, "k"); return err }},
		{name: "HSet", call: func(ctx context.Context) error { _, err := p.HSet(ctx, "h", map[string]string{"f": "v"}); return err }},
		{name: "HGet", call: func(ctx context.Context) error { _, err := p.HGet(ctx, "h", "f"); return err }},
		{name: "HGetAll", call: func(ctx context.Context) error { _, err := p.HGetAll(ctx, "h"); return err }},
		{name: "HDel", call: func(ctx context.Context) error { _, err := p.HDel(ctx, "h", "f"); return err }},
		{name: "LPush", call: func(ctx context.Context) error { _, err := p.LPush(ctx, "l", "a"); return err }},
		{name: "RPush", call: func(ctx context.Context) error { _, err := p.RPush(ctx, "l", "a"); return err }},
		{name: "LRange", call: func(ctx context.Context) error { _, err := p.LRange(ctx, "l", 0, -1); return err }},
		{name: "LLen", call: func(ctx context.Context) error { _, err := p.LLen(ctx, "l"); return err }},
		{name: "LPop", call: func(ctx context.Context) error { _, err := p.LPop(ctx, "l"); return err }},
		{name: "RPop", call: func(ctx context.Context) error { _, err := p.RPop(ctx, "l"); return err }},
		{name: "Pipeline", call: func(ctx context.Context) error {
			_, err := p.Pipeline(ctx, []provider.PipelineCommand{{Type: provider.PipelineGet, Key: "k"}})
			return err
		}},
		{name: "AcquireLock", call: func(ctx context.Context) error {
			_, err := p.AcquireLock(ctx, "lock", "token", time.Second)
			return err
		}},
		{name: "ReleaseLock", call: func(ctx context.Context) error { _, err := p.ReleaseLock(ctx, "lock", "token"); return err }},
		{name: "FixedWindowRateLimit", call: func(ctx context.Context) error {
			_, err := p.FixedWindowRateLimit(ctx, "rate", 1, time.Second)
			return err
		}},
	}
	for _, check := range contextChecks {
		t.Run(check.name+" canceled", func(t *testing.T) {
			requireErrIs(t, check.call(canceled), context.Canceled)
		})
	}

	var nilProvider *Provider
	requireErrIs(t, nilProvider.Ping(ctx), provider.ErrClosed)
	emptyProvider := &Provider{}
	closedChecks := []struct {
		name string
		call func(*Provider) error
	}{
		{name: "Ping", call: func(p *Provider) error { return p.Ping(ctx) }},
		{name: "Close", call: func(p *Provider) error { return p.Close(ctx) }},
		{name: "Get", call: func(p *Provider) error { _, err := p.Get(ctx, "k"); return err }},
		{name: "Set", call: func(p *Provider) error { return p.Set(ctx, "k", "v", 0) }},
		{name: "SetNX", call: func(p *Provider) error { _, err := p.SetNX(ctx, "k", "v", 0); return err }},
		{name: "Del", call: func(p *Provider) error { _, err := p.Del(ctx, "k"); return err }},
		{name: "Exists", call: func(p *Provider) error { _, err := p.Exists(ctx, "k"); return err }},
		{name: "Expire", call: func(p *Provider) error { _, err := p.Expire(ctx, "k", time.Second); return err }},
		{name: "TTL", call: func(p *Provider) error { _, err := p.TTL(ctx, "k"); return err }},
		{name: "MGet", call: func(p *Provider) error { _, err := p.MGet(ctx, "k"); return err }},
		{name: "MSet", call: func(p *Provider) error { return p.MSet(ctx, map[string]string{"k": "v"}) }},
		{name: "Incr", call: func(p *Provider) error { _, err := p.Incr(ctx, "k"); return err }},
		{name: "Decr", call: func(p *Provider) error { _, err := p.Decr(ctx, "k"); return err }},
		{name: "HSet", call: func(p *Provider) error { _, err := p.HSet(ctx, "h", map[string]string{"f": "v"}); return err }},
		{name: "HGet", call: func(p *Provider) error { _, err := p.HGet(ctx, "h", "f"); return err }},
		{name: "HGetAll", call: func(p *Provider) error { _, err := p.HGetAll(ctx, "h"); return err }},
		{name: "HDel", call: func(p *Provider) error { _, err := p.HDel(ctx, "h", "f"); return err }},
		{name: "LPush", call: func(p *Provider) error { _, err := p.LPush(ctx, "l", "a"); return err }},
		{name: "RPush", call: func(p *Provider) error { _, err := p.RPush(ctx, "l", "a"); return err }},
		{name: "LRange", call: func(p *Provider) error { _, err := p.LRange(ctx, "l", 0, -1); return err }},
		{name: "LLen", call: func(p *Provider) error { _, err := p.LLen(ctx, "l"); return err }},
		{name: "LPop", call: func(p *Provider) error { _, err := p.LPop(ctx, "l"); return err }},
		{name: "RPop", call: func(p *Provider) error { _, err := p.RPop(ctx, "l"); return err }},
		{name: "Pipeline", call: func(p *Provider) error {
			_, err := p.Pipeline(ctx, []provider.PipelineCommand{{Type: provider.PipelineGet, Key: "k"}})
			return err
		}},
		{name: "AcquireLock", call: func(p *Provider) error { _, err := p.AcquireLock(ctx, "lock", "token", time.Second); return err }},
		{name: "ReleaseLock", call: func(p *Provider) error { _, err := p.ReleaseLock(ctx, "lock", "token"); return err }},
		{name: "FixedWindowRateLimit", call: func(p *Provider) error {
			_, err := p.FixedWindowRateLimit(ctx, "rate", 1, time.Second)
			return err
		}},
	}
	for _, check := range closedChecks {
		t.Run(check.name+" closed", func(t *testing.T) {
			requireErrIs(t, check.call(emptyProvider), provider.ErrClosed)
		})
	}
	requireNoErr(t, p.Close(ctx))
	requireErrIs(t, p.Ping(ctx), provider.ErrClosed)
}

func TestHelpers(t *testing.T) {
	if got := stringArgs([]string{"a", "b"}); !reflect.DeepEqual(got, []any{"a", "b"}) {
		t.Fatalf("stringArgs = %#v", got)
	}
	args := stringMapToAny(map[string]string{"field": "value"})
	if len(args) != 2 || args[0] != "field" || args[1] != "value" {
		t.Fatalf("stringMapToAny = %#v", args)
	}
	args = mapArgs(map[string]string{"key": "value"})
	if len(args) != 2 || args[0] != "key" || args[1] != "value" {
		t.Fatalf("mapArgs = %#v", args)
	}

	intTests := []struct {
		value any
		want  int64
	}{
		{int64(1), 1},
		{int(2), 2},
		{"3", 3},
		{[]byte("4"), 4},
	}
	for _, tt := range intTests {
		got, err := int64Result(tt.value)
		if err != nil || got != tt.want {
			t.Fatalf("int64Result(%#v) = %d, %v; want %d", tt.value, got, err, tt.want)
		}
	}
	for _, value := range []any{"bad", []byte("bad"), uint64(1)} {
		if _, err := int64Result(value); err == nil {
			t.Fatalf("int64Result(%#v) = nil error, want parse/type error", value)
		}
	}
}

func TestMapErrorClassifications(t *testing.T) {
	if err := mapError(nil); err != nil {
		t.Fatalf("mapError(nil) = %v", err)
	}
	requireErrIs(t, mapError(redis.Nil), provider.ErrNil)
	requireErrIs(t, mapError(context.Canceled), context.Canceled)
	requireErrIs(t, mapError(context.DeadlineExceeded), context.DeadlineExceeded)
	requireErrIs(t, mapError(fakeNetError{timeout: true}), provider.ErrTimeout)
	requireErrIs(t, mapError(fakeNetError{}), provider.ErrNetwork)

	tests := []struct {
		message string
		want    error
	}{
		{"WRONGTYPE Operation against a key holding the wrong kind of value", provider.ErrWrongType},
		{"NOAUTH Authentication required", provider.ErrAuth},
		{"WRONGPASS invalid username-password pair or user is disabled", provider.ErrAuth},
		{"ERR value is not an integer or out of range", provider.ErrInvalidInt},
		{"ERR increment or decrement would overflow and exceed out of range", provider.ErrInvalidInt},
		{"READONLY You can't write against a read only replica", provider.ErrReadOnly},
		{"MISCONF Redis is configured to save RDB snapshots, stop-writes-on-bgsave-error option", provider.ErrReadOnly},
		{"LOADING Redis is loading the dataset in memory", provider.ErrLoading},
		{"TRYAGAIN Multiple keys request during rehashing of slot", provider.ErrTryAgain},
		{"MOVED 3999 127.0.0.1:6381", provider.ErrClusterMoved},
		{"ASK 3999 127.0.0.1:6381", provider.ErrClusterAsk},
		{"i/o timeout while reading", provider.ErrTimeout},
		{"dial tcp 127.0.0.1:6379: connection refused", provider.ErrNetwork},
		{"broken pipe on network connection", provider.ErrNetwork},
	}
	for _, tt := range tests {
		t.Run(tt.want.Error()+" "+strconv.Itoa(len(tt.message)), func(t *testing.T) {
			requireErrIs(t, mapError(errors.New(tt.message)), tt.want)
		})
	}

	original := errors.New("unclassified redis error")
	if got := mapError(original); !errors.Is(got, original) {
		t.Fatalf("mapError(default) = %v, want original", got)
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

func TestCloseMapsClientCloseError(t *testing.T) {
	p := newHookProvider(t, fakeRedisHook{})
	original := closeRedisClient
	closeRedisClient = func(*redis.Client) error {
		return fakeNetError{}
	}
	t.Cleanup(func() {
		closeRedisClient = original
	})

	requireErrIs(t, p.Close(context.Background()), provider.ErrNetwork)
	if p.client == nil {
		t.Fatal("Close cleared client after close error")
	}
}

func TestPipelineResultHelpersRejectUnexpectedCommandTypes(t *testing.T) {
	tests := []struct {
		name        string
		commandType provider.PipelineCommandType
		cmd         redis.Cmder
		call        func(*provider.PipelineResult, provider.PipelineCommandType, redis.Cmder) error
	}{
		{
			name:        "string",
			commandType: provider.PipelineGet,
			cmd:         redis.NewIntCmd(context.Background(), "incr", "k"),
			call:        applyPipelineStringResult,
		},
		{
			name:        "int",
			commandType: provider.PipelineHSet,
			cmd:         redis.NewStringCmd(context.Background(), "get", "k"),
			call:        applyPipelineIntResult,
		},
		{
			name:        "string slice",
			commandType: provider.PipelineLRange,
			cmd:         redis.NewStringCmd(context.Background(), "get", "k"),
			call:        applyPipelineStringSliceResult,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result provider.PipelineResult
			err := tt.call(&result, tt.commandType, tt.cmd)
			if err == nil || !strings.Contains(err.Error(), "unexpected pipeline result type") {
				t.Fatalf("helper error = %v, want unexpected type error", err)
			}
		})
	}
}
