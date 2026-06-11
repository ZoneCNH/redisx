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
	count, err := client.Exists(context.Background(), "alpha", "beta", "missing")
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if count != 2 {
		t.Fatalf("exists count = %d, want 2", count)
	}
	deleted, err := client.Del(context.Background(), "alpha", "missing")
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
	ttl, err := client.TTL(context.Background(), "ttl")
	if err != nil {
		t.Fatalf("ttl: %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("ttl = %v, want positive", ttl)
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

	if err := client.Ping(context.Background()); !IsKind(err, ErrorKindClosed) {
		t.Fatalf("ping after close kind = %v, want closed", err)
	}
	if _, err := client.Get(context.Background(), "key"); !IsKind(err, ErrorKindClosed) {
		t.Fatalf("get after close kind = %v, want closed", err)
	}
}
