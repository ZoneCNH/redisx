package redisx

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestClientHashListPipelinePrimitives(t *testing.T) {
	ctx := context.Background()
	client, err := New(ctx, Config{Name: "redisx"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	added, err := client.HSet(ctx, "profile", map[string]string{"name": "redisx", "version": "1"})
	if err != nil {
		t.Fatalf("hset profile: %v", err)
	}
	if added != 2 {
		t.Fatalf("hset added = %d, want 2", added)
	}
	name, err := client.HGet(ctx, "profile", "name")
	if err != nil {
		t.Fatalf("hget name: %v", err)
	}
	if name != "redisx" {
		t.Fatalf("hget name = %q, want redisx", name)
	}
	values, err := client.HGetAll(ctx, "profile")
	if err != nil {
		t.Fatalf("hgetall profile: %v", err)
	}
	if !reflect.DeepEqual(values, map[string]string{"name": "redisx", "version": "1"}) {
		t.Fatalf("hgetall profile = %#v", values)
	}
	deleted, err := client.HDel(ctx, "profile", "version")
	if err != nil {
		t.Fatalf("hdel version: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("hdel deleted = %d, want 1", deleted)
	}
	if _, err := client.HGet(ctx, "profile", "version"); !IsKind(err, ErrorKindNil) {
		t.Fatalf("missing hash field kind = %v, want nil", err)
	}

	length, err := client.RPush(ctx, "queue", "a", "b")
	if err != nil {
		t.Fatalf("rpush queue: %v", err)
	}
	if length != 2 {
		t.Fatalf("rpush length = %d, want 2", length)
	}
	length, err = client.LPush(ctx, "queue", "z")
	if err != nil {
		t.Fatalf("lpush queue: %v", err)
	}
	if length != 3 {
		t.Fatalf("lpush length = %d, want 3", length)
	}
	items, err := client.LRange(ctx, "queue", 0, -1)
	if err != nil {
		t.Fatalf("lrange queue: %v", err)
	}
	if !reflect.DeepEqual(items, []string{"z", "a", "b"}) {
		t.Fatalf("lrange queue = %#v", items)
	}
	left, err := client.LPop(ctx, "queue")
	if err != nil {
		t.Fatalf("lpop queue: %v", err)
	}
	if left != "z" {
		t.Fatalf("lpop = %q, want z", left)
	}
	right, err := client.RPop(ctx, "queue")
	if err != nil {
		t.Fatalf("rpop queue: %v", err)
	}
	if right != "b" {
		t.Fatalf("rpop = %q, want b", right)
	}

	results, err := client.Pipeline(ctx, []PipelineCommand{
		{Type: PipelineSet, Key: "pipe:string", Value: "value"},
		{Type: PipelineMSet, Values: map[string]string{"pipe:m1": "1", "pipe:m2": "2"}},
		{Type: PipelineHSet, Key: "pipe:hash", Values: map[string]string{"field": "value"}},
		{Type: PipelineRPush, Key: "pipe:list", ListValues: []string{"x", "y"}},
		{Type: PipelineIncr, Key: "pipe:counter"},
	})
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}
	if len(results) != 5 || results[4].Int != 1 {
		t.Fatalf("pipeline results = %#v", results)
	}
	if got, err := client.Get(ctx, "pipe:string"); err != nil || got != "value" {
		t.Fatalf("pipeline string = %q, %v; want value, nil", got, err)
	}
	if got, err := client.HGet(ctx, "pipe:hash", "field"); err != nil || got != "value" {
		t.Fatalf("pipeline hash = %q, %v; want value, nil", got, err)
	}
	if got, err := client.LRange(ctx, "pipe:list", 0, -1); err != nil || !reflect.DeepEqual(got, []string{"x", "y"}) {
		t.Fatalf("pipeline list = %#v, %v; want [x y], nil", got, err)
	}

	if err := client.Set(ctx, "plain", "value", 0); err != nil {
		t.Fatalf("set plain: %v", err)
	}
	if _, err := client.HGetAll(ctx, "plain"); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("wrong-type hash kind = %v, want validation", err)
	}
	if _, err := client.LRange(ctx, "plain", 0, -1); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("wrong-type list kind = %v, want validation", err)
	}
}

func TestClientLocksRateLimitAndCacheHelpers(t *testing.T) {
	ctx := context.Background()
	client, err := New(ctx, Config{Name: "redisx"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	acquired, err := client.AcquireLock(ctx, "lock", "token", time.Minute)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	if !acquired {
		t.Fatal("first lock acquire = false, want true")
	}
	acquired, err = client.AcquireLock(ctx, "lock", "other", time.Minute)
	if err != nil {
		t.Fatalf("acquire locked key: %v", err)
	}
	if acquired {
		t.Fatal("second lock acquire = true, want false")
	}
	released, err := client.ReleaseLock(ctx, "lock", "other")
	if err != nil {
		t.Fatalf("release wrong token: %v", err)
	}
	if released {
		t.Fatal("release wrong token = true, want false")
	}
	released, err = client.ReleaseLock(ctx, "lock", "token")
	if err != nil {
		t.Fatalf("release token: %v", err)
	}
	if !released {
		t.Fatal("release token = false, want true")
	}

	first, err := client.FixedWindowRateLimit(ctx, "rate", 2, time.Minute)
	if err != nil {
		t.Fatalf("rate first: %v", err)
	}
	second, err := client.FixedWindowRateLimit(ctx, "rate", 2, time.Minute)
	if err != nil {
		t.Fatalf("rate second: %v", err)
	}
	third, err := client.FixedWindowRateLimit(ctx, "rate", 2, time.Minute)
	if err != nil {
		t.Fatalf("rate third: %v", err)
	}
	if !first.Allowed || first.Remaining != 1 || !second.Allowed || second.Remaining != 0 || third.Allowed || third.Count != 3 || third.ResetAfter <= 0 {
		t.Fatalf("unexpected rate results: first=%#v second=%#v third=%#v", first, second, third)
	}

	type cachedProfile struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	cache, err := NewCacheClient[cachedProfile](client, JSONCodec[cachedProfile]{})
	if err != nil {
		t.Fatalf("new cache client: %v", err)
	}
	want := cachedProfile{Name: "redisx", Age: 1}
	if err := cache.Set(ctx, "cache:profile", want, 0); err != nil {
		t.Fatalf("cache set: %v", err)
	}
	got, err := cache.Get(ctx, "cache:profile")
	if err != nil {
		t.Fatalf("cache get: %v", err)
	}
	if got != want {
		t.Fatalf("cache get = %#v, want %#v", got, want)
	}
	loads := 0
	loaded, err := cache.GetOrLoad(ctx, "cache:missing", 0, func(context.Context) (cachedProfile, error) {
		loads++
		return cachedProfile{Name: "loaded", Age: 2}, nil
	})
	if err != nil {
		t.Fatalf("cache get or load: %v", err)
	}
	if loaded.Name != "loaded" || loads != 1 {
		t.Fatalf("loaded = %#v loads=%d", loaded, loads)
	}
	loaded, err = cache.GetOrLoad(ctx, "cache:missing", 0, func(context.Context) (cachedProfile, error) {
		loads++
		return cachedProfile{}, errors.New("must not be called")
	})
	if err != nil {
		t.Fatalf("cache get or load cached: %v", err)
	}
	if loaded.Name != "loaded" || loads != 1 {
		t.Fatalf("cached loaded = %#v loads=%d", loaded, loads)
	}

	builder := NewKeyBuilder("app")
	if key := builder.Build(":users:", "42", "profile"); key != "app:users:42:profile" {
		t.Fatalf("built key = %q", key)
	}
}
