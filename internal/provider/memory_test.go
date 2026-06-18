package provider

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func newTestMemory(now time.Time) (*Memory, *time.Time) {
	m := NewMemory()
	current := now
	m.clock = func() time.Time { return current }
	return m, &current
}

func requireErrIs(t *testing.T, err error, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func requireNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPipelineCommandCompatibilityFields(t *testing.T) {
	if got := PipelineCommandKind(PipelineCommand{Type: PipelineGet, Op: PipelineSet, Kind: PipelineIncr}); got != PipelineGet {
		t.Fatalf("PipelineCommandKind Type precedence = %q", got)
	}
	if got := PipelineCommandKind(PipelineCommand{Op: PipelineSet, Kind: PipelineIncr}); got != PipelineSet {
		t.Fatalf("PipelineCommandKind Op fallback = %q", got)
	}
	if got := PipelineCommandKind(PipelineCommand{Kind: PipelineIncr}); got != PipelineIncr {
		t.Fatalf("PipelineCommandKind Kind fallback = %q", got)
	}

	command := PipelineCommand{
		ListValues: []string{"list-values"},
		List:       []string{"list"},
		Items:      []string{"items"},
	}
	if got := PipelineCommandListValues(command); !reflect.DeepEqual(got, []string{"list-values"}) {
		t.Fatalf("PipelineCommandListValues ListValues precedence = %v", got)
	}
	command.ListValues = nil
	if got := PipelineCommandListValues(command); !reflect.DeepEqual(got, []string{"list"}) {
		t.Fatalf("PipelineCommandListValues List fallback = %v", got)
	}
	command.List = nil
	if got := PipelineCommandListValues(command); !reflect.DeepEqual(got, []string{"items"}) {
		t.Fatalf("PipelineCommandListValues Items fallback = %v", got)
	}
}

func TestMemoryStringTTLAndCounters(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC)
	m, current := newTestMemory(now)

	requireNoErr(t, m.Ping(ctx))
	requireNoErr(t, m.Set(ctx, "plain", "value", 0))
	if got, err := m.Get(ctx, "plain"); err != nil || got != "value" {
		t.Fatalf("Get plain = %q, %v", got, err)
	}
	if ttl, err := m.TTL(ctx, "plain"); err != nil || ttl != -time.Second {
		t.Fatalf("TTL no expiration = %s, %v", ttl, err)
	}

	requireNoErr(t, m.Set(ctx, "temporary", "value", 10*time.Second))
	if ttl, err := m.TTL(ctx, "temporary"); err != nil || ttl != 10*time.Second {
		t.Fatalf("TTL temporary = %s, %v", ttl, err)
	}
	*current = current.Add(11 * time.Second)
	if ttl, err := m.TTL(ctx, "temporary"); err != nil || ttl != -2*time.Second {
		t.Fatalf("TTL expired = %s, %v", ttl, err)
	}
	_, err := m.Get(ctx, "temporary")
	requireErrIs(t, err, ErrNil)

	if ok, err := m.SetNX(ctx, "nx", "first", time.Second); err != nil || !ok {
		t.Fatalf("SetNX first = %v, %v", ok, err)
	}
	if ok, err := m.SetNX(ctx, "nx", "second", time.Second); err != nil || ok {
		t.Fatalf("SetNX existing = %v, %v", ok, err)
	}
	if count, err := m.Exists(ctx, "plain", "nx", "missing"); err != nil || count != 2 {
		t.Fatalf("Exists = %d, %v", count, err)
	}
	if count, err := m.Del(ctx, "nx", "missing"); err != nil || count != 1 {
		t.Fatalf("Del = %d, %v", count, err)
	}
	if ok, err := m.Expire(ctx, "missing", time.Second); err != nil || ok {
		t.Fatalf("Expire missing = %v, %v", ok, err)
	}
	if ok, err := m.Expire(ctx, "plain", time.Second); err != nil || !ok {
		t.Fatalf("Expire existing = %v, %v", ok, err)
	}

	requireNoErr(t, m.MSet(ctx, map[string]string{"m1": "one", "m2": "two"}))
	values, err := m.MGet(ctx, "m1", "missing", "m2")
	requireNoErr(t, err)
	wantValues := []Value{{Value: "one", Found: true}, {}, {Value: "two", Found: true}}
	if !reflect.DeepEqual(values, wantValues) {
		t.Fatalf("MGet = %#v, want %#v", values, wantValues)
	}

	if got, err := m.Incr(ctx, "counter"); err != nil || got != 1 {
		t.Fatalf("Incr missing = %d, %v", got, err)
	}
	if got, err := m.Decr(ctx, "counter"); err != nil || got != 0 {
		t.Fatalf("Decr existing = %d, %v", got, err)
	}
	requireNoErr(t, m.Set(ctx, "not-int", "abc", 0))
	_, err = m.Incr(ctx, "not-int")
	requireErrIs(t, err, ErrInvalidInt)

	_, err = m.HSet(ctx, "hash-type", map[string]string{"field": "value"})
	requireNoErr(t, err)
	_, err = m.Get(ctx, "hash-type")
	requireErrIs(t, err, ErrWrongType)
	_, err = m.MGet(ctx, "hash-type")
	requireErrIs(t, err, ErrWrongType)
	_, err = m.Incr(ctx, "hash-type")
	requireErrIs(t, err, ErrWrongType)
}

func TestMemoryHashOperations(t *testing.T) {
	ctx := context.Background()
	m, _ := newTestMemory(time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC))

	if got, err := m.HSet(ctx, "hash", map[string]string{"a": "1", "b": "2"}); err != nil || got != 2 {
		t.Fatalf("HSet new = %d, %v", got, err)
	}
	if got, err := m.HSet(ctx, "hash", map[string]string{"a": "updated", "c": "3"}); err != nil || got != 1 {
		t.Fatalf("HSet update = %d, %v", got, err)
	}
	if got, err := m.HGet(ctx, "hash", "a"); err != nil || got != "updated" {
		t.Fatalf("HGet = %q, %v", got, err)
	}
	_, err := m.HGet(ctx, "hash", "missing")
	requireErrIs(t, err, ErrNil)
	_, err = m.HGet(ctx, "missing", "field")
	requireErrIs(t, err, ErrNil)

	all, err := m.HGetAll(ctx, "hash")
	requireNoErr(t, err)
	all["a"] = "mutated"
	again, err := m.HGetAll(ctx, "hash")
	requireNoErr(t, err)
	if again["a"] != "updated" {
		t.Fatalf("HGetAll returned map was not cloned: %v", again)
	}
	empty, err := m.HGetAll(ctx, "missing")
	requireNoErr(t, err)
	if len(empty) != 0 {
		t.Fatalf("HGetAll missing = %v", empty)
	}
	if got, err := m.HDel(ctx, "hash", "b", "missing"); err != nil || got != 1 {
		t.Fatalf("HDel = %d, %v", got, err)
	}
	if got, err := m.HDel(ctx, "missing", "field"); err != nil || got != 0 {
		t.Fatalf("HDel missing key = %d, %v", got, err)
	}

	requireNoErr(t, m.Set(ctx, "string", "value", 0))
	_, err = m.HSet(ctx, "string", map[string]string{"field": "value"})
	requireErrIs(t, err, ErrWrongType)
	_, err = m.HGet(ctx, "string", "field")
	requireErrIs(t, err, ErrWrongType)
	_, err = m.HGetAll(ctx, "string")
	requireErrIs(t, err, ErrWrongType)
	_, err = m.HDel(ctx, "string", "field")
	requireErrIs(t, err, ErrWrongType)

	m.items["nil-hash"] = entry{kind: entryKindHash}
	if got, err := m.HSet(ctx, "nil-hash", map[string]string{"field": "value"}); err != nil || got != 1 {
		t.Fatalf("HSet nil hash = %d, %v", got, err)
	}
	if got, err := m.HGet(ctx, "nil-hash", "field"); err != nil || got != "value" {
		t.Fatalf("HGet nil hash initialized field = %q, %v", got, err)
	}
	m.items["nil-hash-direct"] = entry{kind: entryKindHash}
	if got, err := m.hsetLocked(time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC), "nil-hash-direct", map[string]string{"field": "value"}); err != nil || got != 1 {
		t.Fatalf("hsetLocked nil hash = %d, %v", got, err)
	}
}

func TestMemoryListOperations(t *testing.T) {
	ctx := context.Background()
	m, _ := newTestMemory(time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC))

	if got, err := m.RPush(ctx, "list", "c", "d"); err != nil || got != 2 {
		t.Fatalf("RPush = %d, %v", got, err)
	}
	if got, err := m.LPush(ctx, "list", "b", "a"); err != nil || got != 4 {
		t.Fatalf("LPush = %d, %v", got, err)
	}
	if got, err := m.LLen(ctx, "list"); err != nil || got != 4 {
		t.Fatalf("LLen = %d, %v", got, err)
	}
	if got, err := m.LRange(ctx, "list", 0, -1); err != nil || !reflect.DeepEqual(got, []string{"a", "b", "c", "d"}) {
		t.Fatalf("LRange all = %v, %v", got, err)
	}
	if got, err := m.LRange(ctx, "list", -3, -2); err != nil || !reflect.DeepEqual(got, []string{"b", "c"}) {
		t.Fatalf("LRange negative = %v, %v", got, err)
	}
	if got, err := m.LRange(ctx, "list", 10, 11); err != nil || len(got) != 0 {
		t.Fatalf("LRange out of bounds = %v, %v", got, err)
	}
	if got, err := m.LRange(ctx, "missing", 0, -1); err != nil || len(got) != 0 {
		t.Fatalf("LRange missing = %v, %v", got, err)
	}
	if got, err := m.LLen(ctx, "missing"); err != nil || got != 0 {
		t.Fatalf("LLen missing = %d, %v", got, err)
	}
	if got, err := m.LPop(ctx, "list"); err != nil || got != "a" {
		t.Fatalf("LPop = %q, %v", got, err)
	}
	if got, err := m.RPop(ctx, "list"); err != nil || got != "d" {
		t.Fatalf("RPop = %q, %v", got, err)
	}
	_, err := m.LPop(ctx, "missing")
	requireErrIs(t, err, ErrNil)
	m.items["empty"] = entry{kind: entryKindList}
	_, err = m.RPop(ctx, "empty")
	requireErrIs(t, err, ErrNil)

	requireNoErr(t, m.Set(ctx, "string", "value", 0))
	for name, call := range map[string]func() error{
		"LPush":  func() error { _, err := m.LPush(ctx, "string", "x"); return err },
		"RPush":  func() error { _, err := m.RPush(ctx, "string", "x"); return err },
		"LRange": func() error { _, err := m.LRange(ctx, "string", 0, -1); return err },
		"LLen":   func() error { _, err := m.LLen(ctx, "string"); return err },
		"LPop":   func() error { _, err := m.LPop(ctx, "string"); return err },
		"RPop":   func() error { _, err := m.RPop(ctx, "string"); return err },
	} {
		t.Run(name, func(t *testing.T) { requireErrIs(t, call(), ErrWrongType) })
	}
}

func TestMemoryPipeline(t *testing.T) {
	ctx := context.Background()
	m, _ := newTestMemory(time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC))

	results, err := m.Pipeline(ctx, []PipelineCommand{
		{Type: PipelineSet, Key: "s", Value: "value"},
		{Type: PipelineGet, Key: "s"},
		{Type: PipelineMSet, Values: map[string]string{"m": "set"}},
		{Type: PipelineHSet, Key: "h", Values: map[string]string{"f": "v"}},
		{Type: PipelineHGet, Key: "h", Field: "f"},
		{Op: PipelineRPush, Key: "l", List: []string{"a", "b"}},
		{Kind: PipelineLRange, Key: "l", Start: 0, Stop: -1},
		{Type: PipelineIncr, Key: "n"},
		{Type: PipelineGet, Key: "missing"},
		{Type: PipelineHGet, Key: "missing-hash", Field: "field"},
		{Type: PipelineHGet, Key: "h", Field: "missing"},
		{Type: PipelineLRange, Key: "missing-list", Start: 0, Stop: -1},
	})
	requireNoErr(t, err)
	if len(results) != 12 {
		t.Fatalf("Pipeline result length = %d", len(results))
	}
	checks := []struct {
		index int
		want  PipelineResult
	}{
		{0, PipelineResult{Type: PipelineSet, Key: "s", Bool: true}},
		{1, PipelineResult{Type: PipelineGet, Key: "s", Found: true, Value: "value", Values: []Value{{Value: "value", Found: true}}}},
		{2, PipelineResult{Type: PipelineMSet, Bool: true}},
		{3, PipelineResult{Type: PipelineHSet, Key: "h", Int: 1, Count: 1}},
		{4, PipelineResult{Type: PipelineHGet, Key: "h", Found: true, Value: "v"}},
		{5, PipelineResult{Type: PipelineRPush, Key: "l", Int: 2, Count: 2}},
		{6, PipelineResult{Type: PipelineLRange, Key: "l", Strings: []string{"a", "b"}}},
		{7, PipelineResult{Type: PipelineIncr, Key: "n", Int: 1}},
		{8, PipelineResult{Type: PipelineGet, Key: "missing"}},
		{9, PipelineResult{Type: PipelineHGet, Key: "missing-hash"}},
		{10, PipelineResult{Type: PipelineHGet, Key: "h"}},
		{11, PipelineResult{Type: PipelineLRange, Key: "missing-list", Strings: []string{}}},
	}
	for _, check := range checks {
		if !reflect.DeepEqual(results[check.index], check.want) {
			t.Fatalf("result[%d] = %#v, want %#v", check.index, results[check.index], check.want)
		}
	}

	_, err = m.Pipeline(ctx, []PipelineCommand{{Type: "unsupported", Key: "x"}})
	if err == nil || !strings.Contains(err.Error(), "unsupported pipeline command type") {
		t.Fatalf("unsupported pipeline err = %v", err)
	}
	requireNoErr(t, m.Set(ctx, "wrong", "value", 0))
	_, err = m.Pipeline(ctx, []PipelineCommand{{Type: PipelineHSet, Key: "wrong", Values: map[string]string{"field": "value"}}})
	requireErrIs(t, err, ErrWrongType)
	_, err = m.Pipeline(ctx, []PipelineCommand{{Type: PipelineRPush, Key: "wrong", ListValues: []string{"value"}}})
	requireErrIs(t, err, ErrWrongType)
	_, err = m.Pipeline(ctx, []PipelineCommand{{Type: PipelineLRange, Key: "wrong", Start: 0, Stop: -1}})
	requireErrIs(t, err, ErrWrongType)
	_, err = m.Pipeline(ctx, []PipelineCommand{{Type: PipelineGet, Key: "h"}})
	requireErrIs(t, err, ErrWrongType)
	_, err = m.Pipeline(ctx, []PipelineCommand{{Type: PipelineHGet, Key: "wrong", Field: "field"}})
	requireErrIs(t, err, ErrWrongType)
	requireNoErr(t, m.Set(ctx, "existing-int", "9", 0))
	results, err = m.Pipeline(ctx, []PipelineCommand{{Type: PipelineIncr, Key: "existing-int"}})
	requireNoErr(t, err)
	if results[0].Int != 10 {
		t.Fatalf("Pipeline incr existing = %#v", results[0])
	}
	_, err = m.RPush(ctx, "list-int", "value")
	requireNoErr(t, err)
	_, err = m.Pipeline(ctx, []PipelineCommand{{Type: PipelineIncr, Key: "list-int"}})
	requireErrIs(t, err, ErrWrongType)
	requireNoErr(t, m.Set(ctx, "badint", "value", 0))
	_, err = m.Pipeline(ctx, []PipelineCommand{{Type: PipelineIncr, Key: "badint"}})
	if err == nil || !strings.Contains(err.Error(), "invalid syntax") {
		t.Fatalf("Pipeline incr invalid int err = %v", err)
	}
}

func TestMemoryLocksAndRateLimit(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC)
	m, current := newTestMemory(now)

	if ok, err := m.AcquireLock(ctx, "lock", "token", time.Second); err != nil || !ok {
		t.Fatalf("AcquireLock first = %v, %v", ok, err)
	}
	if ok, err := m.AcquireLock(ctx, "lock", "other", time.Second); err != nil || ok {
		t.Fatalf("AcquireLock existing = %v, %v", ok, err)
	}
	if ok, err := m.ReleaseLock(ctx, "lock", "other"); err != nil || ok {
		t.Fatalf("ReleaseLock wrong token = %v, %v", ok, err)
	}
	if ok, err := m.ReleaseLock(ctx, "lock", "token"); err != nil || !ok {
		t.Fatalf("ReleaseLock token = %v, %v", ok, err)
	}
	if ok, err := m.ReleaseLock(ctx, "lock", "token"); err != nil || ok {
		t.Fatalf("ReleaseLock missing = %v, %v", ok, err)
	}
	_, err := m.RPush(ctx, "list-lock", "value")
	requireNoErr(t, err)
	_, err = m.ReleaseLock(ctx, "list-lock", "token")
	requireErrIs(t, err, ErrWrongType)

	first, err := m.FixedWindowRateLimit(ctx, "rate", 2, 10*time.Second)
	requireNoErr(t, err)
	if !first.Allowed || first.Count != 1 || first.Remaining != 1 || first.ResetAfter != 10*time.Second || first.Limit != 2 {
		t.Fatalf("first rate result = %#v", first)
	}
	second, err := m.FixedWindowRateLimit(ctx, "rate", 2, 10*time.Second)
	requireNoErr(t, err)
	if !second.Allowed || second.Count != 2 || second.Remaining != 0 {
		t.Fatalf("second rate result = %#v", second)
	}
	third, err := m.FixedWindowRateLimit(ctx, "rate", 2, 10*time.Second)
	requireNoErr(t, err)
	if third.Allowed || third.Count != 3 || third.Remaining != 0 {
		t.Fatalf("third rate result = %#v", third)
	}
	*current = current.Add(11 * time.Second)
	reset, err := m.FixedWindowRateLimit(ctx, "rate", 2, 10*time.Second)
	requireNoErr(t, err)
	if reset.Count != 1 || !reset.Allowed {
		t.Fatalf("reset rate result = %#v", reset)
	}

	m.items["rate-no-expiry"] = entry{kind: entryKindString, value: "0"}
	zeroExpiry, err := m.FixedWindowRateLimit(ctx, "rate-no-expiry", 1, 5*time.Second)
	requireNoErr(t, err)
	if !zeroExpiry.Allowed || zeroExpiry.Count != 1 || zeroExpiry.ResetAfter != 5*time.Second {
		t.Fatalf("rate with zero expiry = %#v", zeroExpiry)
	}

	requireNoErr(t, m.Set(ctx, "bad-rate", "not-int", 0))
	_, err = m.FixedWindowRateLimit(ctx, "bad-rate", 1, time.Second)
	if err == nil {
		t.Fatal("FixedWindowRateLimit parse error = nil")
	}
	_, err = m.RPush(ctx, "list-rate", "value")
	requireNoErr(t, err)
	_, err = m.FixedWindowRateLimit(ctx, "list-rate", 1, time.Second)
	requireErrIs(t, err, ErrWrongType)
}

func TestMemoryContextCloseAndHelpers(t *testing.T) {
	ctx := context.Background()
	var nilCtx context.Context
	now := time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC)
	m, _ := newTestMemory(now)

	if err := m.Ping(nilCtx); err == nil || !strings.Contains(err.Error(), "context is required") {
		t.Fatalf("Ping nil context err = %v", err)
	}
	if err := m.Close(nilCtx); err == nil || !strings.Contains(err.Error(), "context is required") {
		t.Fatalf("Close nil context err = %v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := m.Ping(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("Ping canceled err = %v", err)
	}
	requireNoErr(t, m.Close(ctx))
	requireErrIs(t, m.Ping(ctx), ErrClosed)

	if got := expiresAt(now, 0); !got.IsZero() {
		t.Fatalf("expiresAt zero = %s", got)
	}
	if got := expiresAt(now, -time.Second); !got.IsZero() {
		t.Fatalf("expiresAt negative = %s", got)
	}
	if got := expiresAt(now, time.Second); !got.Equal(now.Add(time.Second)) {
		t.Fatalf("expiresAt positive = %s", got)
	}
	listTests := []struct {
		name   string
		values []string
		start  int64
		stop   int64
		want   []string
	}{
		{name: "empty", values: []string{}, start: 0, stop: -1, want: []string{}},
		{name: "clamp start below zero", values: []string{"a", "b"}, start: -5, stop: 1, want: []string{"a", "b"}},
		{name: "clamp stop beyond length", values: []string{"a", "b"}, start: 0, stop: 10, want: []string{"a", "b"}},
		{name: "negative stop before list", values: []string{"a", "b"}, start: 0, stop: -3, want: []string{}},
		{name: "inverted", values: []string{"a", "b"}, start: 1, stop: 0, want: []string{}},
	}
	for _, tt := range listTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := listRange(tt.values, tt.start, tt.stop); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("listRange(%v, %d, %d) = %v, want %v", tt.values, tt.start, tt.stop, got, tt.want)
			}
		})
	}
	cloned := cloneMap(map[string]string{"a": "b"})
	cloned["a"] = "c"
	if cloned["a"] != "c" {
		t.Fatalf("cloneMap mutation failed: %v", cloned)
	}
}
