package testkit

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ZoneCNH/redisx/pkg/redisx"
)

func TestNewFakeRedisKeyValueContract(t *testing.T) {
	ctx := context.Background()
	fake := NewFakeRedis()
	if err := fake.Ping(ctx); err != nil {
		t.Fatalf("Ping() unexpected error: %v", err)
	}
	if err := fake.Set(ctx, "k", "v", 0); err != nil {
		t.Fatalf("Set() unexpected error: %v", err)
	}
	got, err := fake.Get(ctx, "k")
	if err != nil || got != "v" {
		t.Fatalf("Get() = %q, %v; want v, nil", got, err)
	}
	values, err := fake.MGet(ctx, "k", "missing")
	if err != nil {
		t.Fatalf("MGet() unexpected error: %v", err)
	}
	if len(values) != 2 || !values[0].Found || values[0].Value != "v" || values[1].Found {
		t.Fatalf("MGet() = %#v", values)
	}
}

func TestNewFakeRedisTTLAndCloseContract(t *testing.T) {
	ctx := context.Background()
	fake := NewFakeRedis()
	if err := fake.Set(ctx, "ttl", "v", time.Second); err != nil {
		t.Fatalf("Set() unexpected error: %v", err)
	}
	if ttl, err := fake.TTL(ctx, "ttl"); err != nil || ttl <= 0 {
		t.Fatalf("TTL() = %v, %v; want positive ttl", ttl, err)
	}
	if err := fake.Close(ctx); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}
	if err := fake.Ping(ctx); err == nil {
		t.Fatal("Ping() after Close expected error")
	}
}

func TestNewFakeRedisWorksWithClientWithoutRedisEnvironment(t *testing.T) {
	t.Setenv("REDIS_URL", "redis://should-not-be-used:6379")
	t.Setenv("REDIS_HOST", "should-not-be-used")
	ctx := context.Background()
	client, err := redisx.New(ctx, Config("fake-client"), redisx.WithProvider(NewFakeRedis()))
	if err != nil {
		t.Fatalf("redisx.New() unexpected error: %v", err)
	}
	defer client.Close(ctx)
	if err := client.Set(ctx, "k", "v", 0); err != nil {
		t.Fatalf("Set() unexpected error: %v", err)
	}
	if got, err := client.Get(ctx, "k"); err != nil || got != "v" {
		t.Fatalf("Get() = %q, %v; want v, nil", got, err)
	}
	if os.Getenv("REDIS_URL") == "" || os.Getenv("REDIS_HOST") == "" {
		t.Fatal("test setup did not preserve forbidden real Redis environment markers")
	}
}

func TestNewFakeRedisMissingKeyMapsToRedisNil(t *testing.T) {
	ctx := context.Background()
	client, err := redisx.New(ctx, Config("fake-nil"), redisx.WithProvider(NewFakeRedis()))
	if err != nil {
		t.Fatalf("redisx.New() unexpected error: %v", err)
	}
	defer client.Close(ctx)
	_, err = client.Get(ctx, "missing")
	if !redisx.IsKind(err, redisx.ErrorKindNil) {
		t.Fatalf("missing key error = %v; want redis nil taxonomy", err)
	}
}
