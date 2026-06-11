package testkit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ZoneCNH/redisx/pkg/redisx"
)

type fakeRedisContract struct {
	Ping        string         `json:"ping"`
	Get         string         `json:"get"`
	MGet        []redisx.Value `json:"mget"`
	Exists      int64          `json:"exists"`
	Expire      bool           `json:"expire"`
	TTL         string         `json:"ttl"`
	Incr        int64          `json:"incr"`
	Decr        int64          `json:"decr"`
	Deleted     int64          `json:"deleted"`
	MissingTTL  string         `json:"missing_ttl"`
	ClosedError bool           `json:"closed_error"`
}

func TestNewFakeRedisJSONContract(t *testing.T) {
	ctx := context.Background()
	fake := NewFakeRedis()

	RequireNoError(t, fake.Ping(ctx))
	RequireNoError(t, fake.Set(ctx, "alpha", "one", 0))
	RequireNoError(t, fake.MSet(ctx, map[string]string{"beta": "two"}))
	RequireNoError(t, fake.Set(ctx, "counter", "1", 0))

	got, err := fake.Get(ctx, "alpha")
	RequireNoError(t, err)
	values, err := fake.MGet(ctx, "alpha", "beta", "missing")
	RequireNoError(t, err)
	exists, err := fake.Exists(ctx, "alpha", "beta", "missing")
	RequireNoError(t, err)
	expire, err := fake.Expire(ctx, "alpha", 0)
	RequireNoError(t, err)
	ttl, err := fake.TTL(ctx, "alpha")
	RequireNoError(t, err)
	incr, err := fake.Incr(ctx, "counter")
	RequireNoError(t, err)
	decr, err := fake.Decr(ctx, "counter")
	RequireNoError(t, err)
	deleted, err := fake.Del(ctx, "alpha", "beta")
	RequireNoError(t, err)
	missingTTL, err := fake.TTL(ctx, "alpha")
	RequireNoError(t, err)
	RequireNoError(t, fake.Close(ctx))
	closedErr := fake.Ping(ctx) != nil

	actual, err := json.MarshalIndent(fakeRedisContract{
		Ping:        "ok",
		Get:         got,
		MGet:        values,
		Exists:      exists,
		Expire:      expire,
		TTL:         ttl.String(),
		Incr:        incr,
		Decr:        decr,
		Deleted:     deleted,
		MissingTTL:  missingTTL.String(),
		ClosedError: closedErr,
	}, "", "  ")
	RequireNoError(t, err)
	RequireGolden(t, "testdata/fake_redis_contract.golden", append(actual, '\n'))
}

func TestNewFakeRedisWorksWithClient(t *testing.T) {
	ctx := context.Background()
	client, err := redisx.New(ctx, Config("fake-redis"), redisx.WithProvider(NewFakeRedis()))
	RequireNoError(t, err)
	defer func() { RequireNoError(t, client.Close(ctx)) }()

	RequireNoError(t, client.Set(ctx, "key", "value", 0))
	got, err := client.Get(ctx, "key")
	RequireNoError(t, err)
	if got != "value" {
		t.Fatalf("client.Get() = %q, want value", got)
	}
}

func TestDefaultClientDoesNotDialRealRedis(t *testing.T) {
	t.Setenv("REDIS_URL", "redis://127.0.0.1:1")
	t.Setenv("REDIS_ADDR", "127.0.0.1:1")

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	client, err := redisx.New(ctx, Config("default-memory"))
	RequireNoError(t, err)
	defer func() { RequireNoError(t, client.Close(context.Background())) }()

	RequireNoError(t, client.Set(ctx, "offline", "memory", 0))
	got, err := client.Get(ctx, "offline")
	RequireNoError(t, err)
	if got != "memory" {
		t.Fatalf("default client read = %q, want memory", got)
	}
}
