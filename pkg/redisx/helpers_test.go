package redisx

import (
	"context"
	"testing"
	"time"
)

type cacheUser struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestKeyBuilderBuild(t *testing.T) {
	if got := NewKeyBuilder("app").Build("users", "42"); got != "app:users:42" {
		t.Fatalf("Build with prefix = %q, want app:users:42", got)
	}
	builder := KeyBuilder{Prefix: "app", Separator: "/"}
	if got := builder.Build("", "users", "42"); got != "app/users/42" {
		t.Fatalf("Build custom separator = %q, want app/users/42", got)
	}
	if got := (KeyBuilder{}).Build("users", "42"); got != "users:42" {
		t.Fatalf("Build without prefix = %q, want users:42", got)
	}
}

func TestJSONCacheGetSetGetOrLoad(t *testing.T) {
	ctx := context.Background()
	client, err := New(ctx, Config{Name: "redisx"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cache := Cache[cacheUser]{Client: client, TTL: time.Minute}

	if _, found, err := cache.Get(ctx, "users:42"); err != nil || found {
		t.Fatalf("missing cache get found=%v err=%v; want false nil", found, err)
	}
	loaded := 0
	value, err := cache.GetOrLoad(ctx, "users:42", func(context.Context) (cacheUser, error) {
		loaded++
		return cacheUser{Name: "Ada", Count: 1}, nil
	})
	if err != nil {
		t.Fatalf("GetOrLoad miss: %v", err)
	}
	if value != (cacheUser{Name: "Ada", Count: 1}) || loaded != 1 {
		t.Fatalf("GetOrLoad miss value=%#v loaded=%d, want Ada once", value, loaded)
	}
	value, err = cache.GetOrLoad(ctx, "users:42", func(context.Context) (cacheUser, error) {
		loaded++
		return cacheUser{Name: "Grace", Count: 2}, nil
	})
	if err != nil {
		t.Fatalf("GetOrLoad hit: %v", err)
	}
	if value != (cacheUser{Name: "Ada", Count: 1}) || loaded != 1 {
		t.Fatalf("GetOrLoad hit value=%#v loaded=%d, want cached Ada without reload", value, loaded)
	}
	if err := cache.Set(ctx, "users:7", cacheUser{Name: "Grace", Count: 2}); err != nil {
		t.Fatalf("cache set: %v", err)
	}
	got, found, err := cache.Get(ctx, "users:7")
	if err != nil || !found || got != (cacheUser{Name: "Grace", Count: 2}) {
		t.Fatalf("cache get set value=%#v found=%v err=%v; want Grace true nil", got, found, err)
	}
}

func TestLockAcquireRelease(t *testing.T) {
	ctx := context.Background()
	client, err := New(ctx, Config{Name: "redisx"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	lock, acquired, err := client.NewLock(ctx, "locks:job", time.Minute)
	if err != nil || !acquired || lock.Token() == "" {
		t.Fatalf("NewLock acquired=%v token=%q err=%v; want true non-empty nil", acquired, lock.Token(), err)
	}
	if _, acquired, err := client.NewLock(ctx, "locks:job", time.Minute); err != nil || acquired {
		t.Fatalf("NewLock duplicate acquired=%v err=%v; want false nil", acquired, err)
	}
	if released, err := lock.Release(ctx); err != nil || !released {
		t.Fatalf("Release acquired lock released=%v err=%v; want true nil", released, err)
	}
	if released, err := lock.Release(ctx); err != nil || released {
		t.Fatalf("Release already released lock released=%v err=%v; want false nil", released, err)
	}
	if _, acquired, err := client.NewLock(ctx, "locks:job", time.Minute); err != nil || !acquired {
		t.Fatalf("NewLock after release acquired=%v err=%v; want true nil", acquired, err)
	}
}

func TestFixedWindowRateLimiterAllow(t *testing.T) {
	ctx := context.Background()
	client, err := New(ctx, Config{Name: "redisx"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	clock := time.Date(2026, 6, 13, 3, 0, 10, 0, time.UTC)
	limiter := FixedWindowRateLimiter{
		Client: client,
		Prefix: "rate",
		Limit:  2,
		Window: time.Minute,
		Clock:  func() time.Time { return clock },
	}

	first, err := limiter.Allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("first allow: %v", err)
	}
	if !first.Allowed || first.Remaining != 1 || first.Limit != 2 || first.ResetAfter != 50*time.Second {
		t.Fatalf("first result = %#v, want allowed remaining 1 reset 50s", first)
	}
	second, err := limiter.Allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("second allow: %v", err)
	}
	if !second.Allowed || second.Remaining != 0 || second.ResetAfter != 50*time.Second {
		t.Fatalf("second result = %#v, want allowed remaining 0 reset 50s", second)
	}
	third, err := limiter.Allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("third allow: %v", err)
	}
	if third.Allowed || third.Remaining != 0 || third.ResetAfter != 50*time.Second {
		t.Fatalf("third result = %#v, want blocked remaining 0 reset 50s", third)
	}
}
