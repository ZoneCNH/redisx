package testkit

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewFakeRedisContract(t *testing.T) {
	ctx := context.Background()
	fake := NewFakeRedis()

	RequireNoError(t, fake.Ping(ctx))
	RequireNoError(t, fake.Set(ctx, "alpha", "1", time.Minute))
	value, err := fake.Get(ctx, "alpha")
	RequireNoError(t, err)
	if value != "1" {
		t.Fatalf("Get(alpha) = %q, want 1", value)
	}

	RequireNoError(t, fake.MSet(ctx, map[string]string{"beta": "2", "gamma": "3"}))
	values, err := fake.MGet(ctx, "alpha", "missing", "beta")
	RequireNoError(t, err)
	if len(values) != 3 || !values[0].Found || values[0].Value != "1" || values[1].Found || !values[2].Found || values[2].Value != "2" {
		t.Fatalf("unexpected MGet values: %#v", values)
	}

	exists, err := fake.Exists(ctx, "alpha", "missing", "beta")
	RequireNoError(t, err)
	if exists != 2 {
		t.Fatalf("Exists = %d, want 2", exists)
	}
	updated, err := fake.Expire(ctx, "alpha", time.Minute)
	RequireNoError(t, err)
	if !updated {
		t.Fatal("expected Expire to update alpha")
	}
	ttl, err := fake.TTL(ctx, "alpha")
	RequireNoError(t, err)
	if ttl <= 0 {
		t.Fatalf("TTL = %s, want positive", ttl)
	}

	next, err := fake.Incr(ctx, "counter")
	RequireNoError(t, err)
	if next != 1 {
		t.Fatalf("Incr = %d, want 1", next)
	}
	next, err = fake.Decr(ctx, "counter")
	RequireNoError(t, err)
	if next != 0 {
		t.Fatalf("Decr = %d, want 0", next)
	}
	deleted, err := fake.Del(ctx, "alpha", "missing")
	RequireNoError(t, err)
	if deleted != 1 {
		t.Fatalf("Del = %d, want 1", deleted)
	}
	RequireNoError(t, fake.Close(ctx))
	if err := fake.Ping(ctx); err == nil {
		t.Fatal("expected closed fake provider to reject Ping")
	}

	RequireGolden(t, "testdata/golden/fake_redis_contract.txt", []byte("fake-redis:in-memory:no-network\n"))
}

func TestNewFakeRedisInstancesAreIsolated(t *testing.T) {
	ctx := context.Background()
	first := NewFakeRedis()
	second := NewFakeRedis()

	RequireNoError(t, first.Set(ctx, "shared", "first", 0))
	if _, err := second.Get(ctx, "shared"); err == nil {
		t.Fatal("expected independent fake providers to isolate keys")
	}
}

func TestFakeAndDefaultClientsDoNotUseRealRedisEnvironment(t *testing.T) {
	t.Setenv("REDIS_URL", "redis://127.0.0.1:1")
	t.Setenv("REDIS_HOST", "127.0.0.1")
	t.Setenv("REDIS_PORT", "1")
	if os.Getenv("REDIS_URL") == "" {
		t.Fatal("expected Redis environment guard to be set")
	}

	ctx := context.Background()
	fakeClient, err := NewClientWithFakeRedis(ctx, Config("fake-client"))
	RequireNoError(t, err)
	RequireNoError(t, fakeClient.Set(ctx, "key", "fake", 0))

	defaultClient, err := redisxNew(ctx, Config("default-client"))
	RequireNoError(t, err)
	RequireNoError(t, defaultClient.Set(ctx, "key", "default", 0))
}
