package redisx

import (
	"context"
	"testing"
	"time"
)

func TestClientKeyValueOperations(t *testing.T) {
	client, err := New(context.Background(), Config{Name: "redisx"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if err := client.Set(context.Background(), "alpha", "1", time.Minute); err != nil {
		t.Fatalf("set alpha: %v", err)
	}
	value, err := client.Get(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("get alpha: %v", err)
	}
	if value != "1" {
		t.Fatalf("get alpha = %q, want 1", value)
	}

	values, err := client.MGet(context.Background(), "alpha", "missing")
	if err != nil {
		t.Fatalf("mget: %v", err)
	}
	if len(values) != 2 || !values[0].Found || values[0].Value != "1" || values[1].Found {
		t.Fatalf("unexpected mget values: %#v", values)
	}

	if err := client.MSet(context.Background(), map[string]string{"beta": "2", "gamma": "3"}); err != nil {
		t.Fatalf("mset: %v", err)
	}
	values, err = client.MGet(context.Background(), "beta", "missing", "gamma")
	if err != nil {
		t.Fatalf("mget after mset: %v", err)
	}
	if len(values) != 3 || !values[0].Found || values[0].Value != "2" || values[1].Found || !values[2].Found || values[2].Value != "3" {
		t.Fatalf("unexpected mget after mset values: %#v", values)
	}
	count, err := client.Exists(context.Background(), "alpha", "beta", "missing")
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if count != 2 {
		t.Fatalf("exists count = %d, want 2", count)
	}
	deleted, err := client.Del(context.Background(), "beta", "gamma", "missing")
	if err != nil {
		t.Fatalf("del beta gamma missing: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("deleted beta gamma missing = %d, want 2", deleted)
	}
	count, err = client.Exists(context.Background(), "alpha", "beta", "gamma", "missing")
	if err != nil {
		t.Fatalf("exists after del: %v", err)
	}
	if count != 1 {
		t.Fatalf("exists after del count = %d, want 1", count)
	}
	deleted, err = client.Del(context.Background(), "alpha", "missing")
	if err != nil {
		t.Fatalf("del: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	if _, err := client.Get(context.Background(), "alpha"); !IsKind(err, ErrorKindNil) {
		t.Fatalf("missing get kind = %v, want nil", err)
	}
}

func TestClientHashListAndPipelineOperations(t *testing.T) {
	client, err := New(context.Background(), Config{Name: "redisx"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	added, err := client.HSet(context.Background(), "hash", map[string]string{"name": "redisx", "version": "1"})
	if err != nil {
		t.Fatalf("hset: %v", err)
	}
	if added != 2 {
		t.Fatalf("hset added = %d, want 2", added)
	}
	value, err := client.HGet(context.Background(), "hash", "name")
	if err != nil {
		t.Fatalf("hget: %v", err)
	}
	if value != "redisx" {
		t.Fatalf("hget name = %q, want redisx", value)
	}
	fields, err := client.HGetAll(context.Background(), "hash")
	if err != nil {
		t.Fatalf("hgetall: %v", err)
	}
	if fields["name"] != "redisx" || fields["version"] != "1" {
		t.Fatalf("hgetall = %#v, want name/version", fields)
	}
	removed, err := client.HDel(context.Background(), "hash", "version")
	if err != nil {
		t.Fatalf("hdel: %v", err)
	}
	if removed != 1 {
		t.Fatalf("hdel removed = %d, want 1", removed)
	}

	length, err := client.RPush(context.Background(), "list", "one", "two")
	if err != nil {
		t.Fatalf("rpush: %v", err)
	}
	if length != 2 {
		t.Fatalf("rpush length = %d, want 2", length)
	}
	length, err = client.LPush(context.Background(), "list", "zero")
	if err != nil {
		t.Fatalf("lpush: %v", err)
	}
	if length != 3 {
		t.Fatalf("lpush length = %d, want 3", length)
	}
	items, err := client.LRange(context.Background(), "list", 0, -1)
	if err != nil {
		t.Fatalf("lrange: %v", err)
	}
	if len(items) != 3 || items[0] != "zero" || items[1] != "one" || items[2] != "two" {
		t.Fatalf("lrange = %#v, want zero/one/two", items)
	}

	results, err := client.Pipeline(context.Background(), []PipelineCommand{
		{Type: PipelineSet, Key: "pipe:string", Value: "value"},
		{Type: PipelineMSet, Values: map[string]string{"pipe:m1": "one", "pipe:m2": "two"}},
		{Type: PipelineHSet, Key: "pipe:hash", Values: map[string]string{"field": "hash-value"}},
		{Type: PipelineRPush, Key: "pipe:list", ListValues: []string{"a", "b"}},
		{Type: PipelineIncr, Key: "pipe:counter"},
	})
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}
	if len(results) != 5 {
		t.Fatalf("pipeline results len = %d, want 5", len(results))
	}
	if !results[0].Bool {
		t.Fatalf("pipeline set result = %#v, want true", results[0])
	}
	if !results[1].Bool {
		t.Fatalf("pipeline mset result = %#v, want true", results[1])
	}
	if results[2].Int != 1 {
		t.Fatalf("pipeline hset result = %#v, want int 1", results[2])
	}
	if results[3].Int != 2 {
		t.Fatalf("pipeline rpush result = %#v, want int 2", results[3])
	}
	if results[4].Int != 1 {
		t.Fatalf("pipeline incr result = %#v, want int 1", results[4])
	}
	pipeValue, err := client.Get(context.Background(), "pipe:string")
	if err != nil {
		t.Fatalf("pipeline get verification: %v", err)
	}
	if pipeValue != "value" {
		t.Fatalf("pipeline get verification = %q, want value", pipeValue)
	}
	msetValues, err := client.MGet(context.Background(), "pipe:m1", "pipe:m2")
	if err != nil {
		t.Fatalf("pipeline mget verification: %v", err)
	}
	if len(msetValues) != 2 || !msetValues[0].Found || msetValues[0].Value != "one" || !msetValues[1].Found || msetValues[1].Value != "two" {
		t.Fatalf("pipeline mget verification = %#v, want one/two", msetValues)
	}
	hashValue, err := client.HGet(context.Background(), "pipe:hash", "field")
	if err != nil {
		t.Fatalf("pipeline hget verification: %v", err)
	}
	if hashValue != "hash-value" {
		t.Fatalf("pipeline hget verification = %q, want hash-value", hashValue)
	}
	listItems, err := client.LRange(context.Background(), "pipe:list", 0, -1)
	if err != nil {
		t.Fatalf("pipeline lrange verification: %v", err)
	}
	if len(listItems) != 2 || listItems[0] != "a" || listItems[1] != "b" {
		t.Fatalf("pipeline lrange verification = %#v, want a/b", listItems)
	}

	readResults, err := client.Pipeline(context.Background(), []PipelineCommand{
		{Type: PipelineGet, Key: "pipe:string"},
		{Type: PipelineGet, Key: "pipe:missing"},
		{Type: PipelineHGet, Key: "pipe:hash", Field: "field"},
		{Type: PipelineHGet, Key: "pipe:hash", Field: "missing"},
		{Type: PipelineLRange, Key: "pipe:list", Start: 0, Stop: -1},
		{Type: PipelineLRange, Key: "pipe:missing-list", Start: 0, Stop: -1},
	})
	if err != nil {
		t.Fatalf("pipeline read miss handling: %v", err)
	}
	if len(readResults) != 6 {
		t.Fatalf("pipeline read results len = %d, want 6", len(readResults))
	}
	if !readResults[0].Found || readResults[0].Value != "value" || readResults[1].Found {
		t.Fatalf("pipeline get read results = %#v, want value/missing", readResults[:2])
	}
	if !readResults[2].Found || readResults[2].Value != "hash-value" || readResults[3].Found {
		t.Fatalf("pipeline hget read results = %#v, want hash-value/missing", readResults[2:4])
	}
	if len(readResults[4].Strings) != 2 || readResults[4].Strings[0] != "a" || readResults[4].Strings[1] != "b" || len(readResults[5].Strings) != 0 {
		t.Fatalf("pipeline lrange read results = %#v, want [a b]/[]", readResults[4:])
	}

	if err := client.Set(context.Background(), "plain", "value", 0); err != nil {
		t.Fatalf("set plain: %v", err)
	}
	if _, err := client.HGet(context.Background(), "plain", "field"); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("wrong-type hget kind = %v, want validation", err)
	}
}

func TestClientPingRecordsOperationMetric(t *testing.T) {
	metrics := &recordingMetrics{}
	client, err := New(context.Background(), Config{Name: "redisx"}, WithMetrics(metrics))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
	if !metrics.counterWithLabel(MetricRedisOperationsTotal, "op", "ping") {
		t.Fatalf("expected ping operation metric, got %#v", metrics.counters)
	}
}

func TestClientExpirationAndCounters(t *testing.T) {
	metrics := &recordingMetrics{}
	client, err := New(context.Background(), Config{Name: "redisx"}, WithMetrics(metrics))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if err := client.Set(context.Background(), "counter", "0", 0); err != nil {
		t.Fatalf("set counter: %v", err)
	}
	value, err := client.Incr(context.Background(), "counter")
	if err != nil {
		t.Fatalf("incr: %v", err)
	}
	if value != 1 {
		t.Fatalf("incr = %d, want 1", value)
	}
	value, err = client.Decr(context.Background(), "counter")
	if err != nil {
		t.Fatalf("decr: %v", err)
	}
	if value != 0 {
		t.Fatalf("decr = %d, want 0", value)
	}
	if !metrics.hasCounter(MetricRedisOperationsTotal) {
		t.Fatalf("expected redis operation metric, got %#v", metrics.counters)
	}

	if err := client.Set(context.Background(), "non-numeric", "not-an-integer", 0); err != nil {
		t.Fatalf("set non-numeric: %v", err)
	}
	if _, err := client.Incr(context.Background(), "non-numeric"); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("incr non-numeric kind = %v, want validation", err)
	}
	if _, err := client.Decr(context.Background(), "non-numeric"); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("decr non-numeric kind = %v, want validation", err)
	}

	if err := client.Set(context.Background(), "permanent", "value", 0); err != nil {
		t.Fatalf("set permanent: %v", err)
	}
	ttl, err := client.TTL(context.Background(), "permanent")
	if err != nil {
		t.Fatalf("ttl permanent: %v", err)
	}
	if ttl != -time.Second {
		t.Fatalf("ttl permanent = %v, want -1s", ttl)
	}

	if err := client.Set(context.Background(), "direct-ttl", "value", time.Minute); err != nil {
		t.Fatalf("set direct ttl: %v", err)
	}
	ttl, err = client.TTL(context.Background(), "direct-ttl")
	if err != nil {
		t.Fatalf("ttl direct ttl: %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("ttl direct ttl = %v, want positive", ttl)
	}

	ttl, err = client.TTL(context.Background(), "missing")
	if err != nil {
		t.Fatalf("ttl missing: %v", err)
	}
	if ttl != -2*time.Second {
		t.Fatalf("ttl missing = %v, want -2s", ttl)
	}

	if err := client.Set(context.Background(), "ttl", "value", 0); err != nil {
		t.Fatalf("set ttl: %v", err)
	}
	ok, err := client.Expire(context.Background(), "ttl", time.Minute)
	if err != nil {
		t.Fatalf("expire: %v", err)
	}
	if !ok {
		t.Fatal("expected expire to update existing key")
	}
	ttl, err = client.TTL(context.Background(), "ttl")
	if err != nil {
		t.Fatalf("ttl: %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("ttl = %v, want positive", ttl)
	}

	updated, err := client.Expire(context.Background(), "missing", time.Minute)
	if err != nil {
		t.Fatalf("expire missing: %v", err)
	}
	if updated {
		t.Fatal("expire missing updated = true, want false")
	}
}

func TestClientOperationsRejectClosedClient(t *testing.T) {
	client, err := New(context.Background(), Config{Name: "redisx"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if err := client.Close(context.Background()); err != nil {
		t.Fatalf("close client: %v", err)
	}

	tests := []struct {
		name string
		run  func() error
	}{
		{name: "ping", run: func() error { return client.Ping(context.Background()) }},
		{name: "get", run: func() error { _, err := client.Get(context.Background(), "key"); return err }},
		{name: "set", run: func() error { return client.Set(context.Background(), "key", "value", 0) }},
		{name: "mget", run: func() error { _, err := client.MGet(context.Background(), "key", "other"); return err }},
		{name: "mset", run: func() error { return client.MSet(context.Background(), map[string]string{"key": "value"}) }},
		{name: "exists", run: func() error { _, err := client.Exists(context.Background(), "key", "other"); return err }},
		{name: "del", run: func() error { _, err := client.Del(context.Background(), "key", "other"); return err }},
		{name: "expire", run: func() error { _, err := client.Expire(context.Background(), "key", time.Minute); return err }},
		{name: "ttl", run: func() error { _, err := client.TTL(context.Background(), "key"); return err }},
		{name: "incr", run: func() error { _, err := client.Incr(context.Background(), "key"); return err }},
		{name: "decr", run: func() error { _, err := client.Decr(context.Background(), "key"); return err }},
		{name: "hset", run: func() error {
			_, err := client.HSet(context.Background(), "key", map[string]string{"field": "value"})
			return err
		}},
		{name: "hget", run: func() error { _, err := client.HGet(context.Background(), "key", "field"); return err }},
		{name: "hgetall", run: func() error { _, err := client.HGetAll(context.Background(), "key"); return err }},
		{name: "hdel", run: func() error { _, err := client.HDel(context.Background(), "key", "field"); return err }},
		{name: "lpush", run: func() error { _, err := client.LPush(context.Background(), "key", "value"); return err }},
		{name: "rpush", run: func() error { _, err := client.RPush(context.Background(), "key", "value"); return err }},
		{name: "lrange", run: func() error { _, err := client.LRange(context.Background(), "key", 0, -1); return err }},
		{name: "lpop", run: func() error { _, err := client.LPop(context.Background(), "key"); return err }},
		{name: "rpop", run: func() error { _, err := client.RPop(context.Background(), "key"); return err }},
		{name: "pipeline", run: func() error {
			_, err := client.Pipeline(context.Background(), []PipelineCommand{{Type: PipelineSet, Key: "key", Value: "value"}})
			return err
		}},
		{name: "acquire lock", run: func() error {
			_, err := client.AcquireLock(context.Background(), "key", "token", time.Minute)
			return err
		}},
		{name: "release lock", run: func() error { _, err := client.ReleaseLock(context.Background(), "key", "token"); return err }},
		{name: "fixed window rate limit", run: func() error {
			_, err := client.FixedWindowRateLimit(context.Background(), "key", 1, time.Minute)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !IsKind(err, ErrorKindClosed) {
				t.Fatalf("%s after close kind = %v, want closed", tt.name, err)
			}
		})
	}
}

func TestClientStructuredWriteOperations(t *testing.T) {
	ctx := context.Background()
	client, err := New(ctx, Config{Name: "redisx"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	added, err := client.HSet(ctx, "hash", map[string]string{"name": "redisx", "version": "1.0.0"})
	if err != nil {
		t.Fatalf("hset: %v", err)
	}
	if added != 2 {
		t.Fatalf("hset added = %d, want 2", added)
	}
	name, err := client.HGet(ctx, "hash", "name")
	if err != nil || name != "redisx" {
		t.Fatalf("hget name = %q, %v; want redisx, nil", name, err)
	}
	fields, err := client.HGetAll(ctx, "hash")
	if err != nil {
		t.Fatalf("hgetall: %v", err)
	}
	if fields["version"] != "1.0.0" {
		t.Fatalf("hgetall version = %q, want 1.0.0", fields["version"])
	}
	deletedFields, err := client.HDel(ctx, "hash", "version", "missing")
	if err != nil || deletedFields != 1 {
		t.Fatalf("hdel = %d, %v; want 1, nil", deletedFields, err)
	}

	length, err := client.RPush(ctx, "list", "a", "b")
	if err != nil || length != 2 {
		t.Fatalf("rpush = %d, %v; want 2, nil", length, err)
	}
	length, err = client.LPush(ctx, "list", "c")
	if err != nil || length != 3 {
		t.Fatalf("lpush = %d, %v; want 3, nil", length, err)
	}
	list, err := client.LRange(ctx, "list", 0, -1)
	if err != nil {
		t.Fatalf("lrange: %v", err)
	}
	if got, want := list, []string{"c", "a", "b"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("lrange = %#v, want %#v", got, want)
	}
	left, err := client.LPop(ctx, "list")
	if err != nil || left != "c" {
		t.Fatalf("lpop = %q, %v; want c, nil", left, err)
	}
	right, err := client.RPop(ctx, "list")
	if err != nil || right != "b" {
		t.Fatalf("rpop = %q, %v; want b, nil", right, err)
	}

	acquired, err := client.AcquireLock(ctx, "once", "first", time.Minute)
	if err != nil || !acquired {
		t.Fatalf("acquire lock first = %v, %v; want true, nil", acquired, err)
	}
	acquired, err = client.AcquireLock(ctx, "once", "second", time.Minute)
	if err != nil || acquired {
		t.Fatalf("acquire lock second = %v, %v; want false, nil", acquired, err)
	}
	released, err := client.ReleaseLock(ctx, "once", "first")
	if err != nil || !released {
		t.Fatalf("release lock = %v, %v; want true, nil", released, err)
	}

	results, err := client.Pipeline(ctx, []PipelineCommand{
		{Type: PipelineSet, Key: "pipe:string", Value: "value"},
		{Type: PipelineHSet, Key: "pipe:hash", Values: map[string]string{"field": "value"}},
		{Type: PipelineRPush, Key: "pipe:list", ListValues: []string{"x", "y"}},
		{Type: PipelineIncr, Key: "pipe:counter"},
	})
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}
	if len(results) != 4 || !results[0].Bool || results[1].Int != 1 || results[2].Int != 2 || results[3].Int != 1 {
		t.Fatalf("pipeline results = %#v", results)
	}
	got, err := client.Get(ctx, "pipe:string")
	if err != nil || got != "value" {
		t.Fatalf("pipeline get verification = %q, %v; want value, nil", got, err)
	}

	if err := client.Set(ctx, "wrong-type", "string", 0); err != nil {
		t.Fatalf("set wrong-type: %v", err)
	}
	if _, err := client.HSet(ctx, "wrong-type", map[string]string{"field": "value"}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("hset wrong-type kind = %v, want validation", err)
	}
}
