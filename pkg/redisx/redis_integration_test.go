package redisx

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestRedisIntegrationWithEnv(t *testing.T) {
	if os.Getenv("REDISX_INTEGRATION") != "1" {
		t.Skip("set REDISX_INTEGRATION=1 with REDISX_REDIS_* to run real Redis integration")
	}

	db, err := strconv.Atoi(envOrDefault("REDISX_REDIS_DB", "0"))
	if err != nil {
		t.Fatalf("parse REDISX_REDIS_DB: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := NewWithOptions(ctx, Options{Config: Config{
		Name: "redisx-integration",
		Redis: RedisConfig{
			Addr:         os.Getenv("REDISX_REDIS_ADDR"),
			Username:     os.Getenv("REDISX_REDIS_USERNAME"),
			Password:     os.Getenv("REDISX_REDIS_PASSWORD"),
			DB:           db,
			DialTimeout:  time.Second,
			ReadTimeout:  time.Second,
			WriteTimeout: time.Second,
			MaxRetries:   1,
		},
	}})
	if err != nil {
		t.Fatalf("new redis client with options: %v", err)
	}

	prefix := "redisx:integration:" + strconv.FormatInt(time.Now().UnixNano(), 10)
	keys := []string{
		prefix + ":alpha",
		prefix + ":beta",
		prefix + ":gamma",
		prefix + ":counter",
		prefix + ":ttl",
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cleanupCancel()
		_, _ = client.Del(cleanupCtx, keys...)
		if err := client.Close(context.Background()); err != nil {
			t.Fatalf("close redis client: %v", err)
		}
	})

	if err := client.Ping(ctx); err != nil {
		skipIfRedisEnvironmentBlocked(t, "ping", err)
		t.Fatalf("ping: %v", err)
	}
	if err := client.Set(ctx, keys[0], "1", 0); err != nil {
		skipIfRedisEnvironmentBlocked(t, "set", err)
		t.Fatalf("set: %v", err)
	}
	value, err := client.Get(ctx, keys[0])
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if value != "1" {
		t.Fatalf("get = %q, want 1", value)
	}

	if err := client.MSet(ctx, map[string]string{keys[1]: "2", keys[2]: "3"}); err != nil {
		t.Fatalf("mset: %v", err)
	}
	values, err := client.MGet(ctx, keys[0], keys[1], prefix+":missing")
	if err != nil {
		t.Fatalf("mget: %v", err)
	}
	if len(values) != 3 || !values[0].Found || values[0].Value != "1" || !values[1].Found || values[1].Value != "2" || values[2].Found {
		t.Fatalf("unexpected mget values: %#v", values)
	}

	if err := client.Set(ctx, keys[3], "0", 0); err != nil {
		t.Fatalf("set counter: %v", err)
	}
	if value, err := client.Incr(ctx, keys[3]); err != nil || value != 1 {
		t.Fatalf("incr = %d, %v; want 1, nil", value, err)
	}
	if value, err := client.Decr(ctx, keys[3]); err != nil || value != 0 {
		t.Fatalf("decr = %d, %v; want 0, nil", value, err)
	}

	if err := client.Set(ctx, keys[4], "ttl", 0); err != nil {
		t.Fatalf("set ttl: %v", err)
	}
	updated, err := client.Expire(ctx, keys[4], time.Minute)
	if err != nil {
		t.Fatalf("expire: %v", err)
	}
	if !updated {
		t.Fatal("expected expire to update existing key")
	}
	ttl, err := client.TTL(ctx, keys[4])
	if err != nil {
		t.Fatalf("ttl: %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("ttl = %v, want positive", ttl)
	}

	count, err := client.Exists(ctx, keys[0], keys[1], prefix+":missing")
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if count != 2 {
		t.Fatalf("exists = %d, want 2", count)
	}
	deleted, err := client.Del(ctx, keys[0], prefix+":missing")
	if err != nil {
		t.Fatalf("del: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("del = %d, want 1", deleted)
	}
}

func envOrDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func skipIfRedisEnvironmentBlocked(t *testing.T, op string, err error) {
	t.Helper()
	if IsKind(err, ErrorKindReadOnly) || IsKind(err, ErrorKindLoading) || IsKind(err, ErrorKindTryAgain) {
		t.Skipf("real Redis %s reached the server but the environment is not writable or ready: %v", op, err)
	}
}
