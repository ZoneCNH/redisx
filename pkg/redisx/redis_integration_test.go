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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := redisIntegrationDB(t)
	client, err := newRedisIntegrationClient(ctx, "redisx-integration", db)
	if err != nil {
		t.Fatalf("new redis client with options: %v", err)
	}

	prefix := "redisx:integration:" + strconv.FormatInt(time.Now().UnixNano(), 10)
	alpha := prefix + ":alpha"
	beta := prefix + ":beta"
	gamma := prefix + ":gamma"
	counter := prefix + ":counter"
	ttlKey := prefix + ":ttl"
	directTTL := prefix + ":direct-ttl"
	nonNumeric := prefix + ":non-numeric"
	missing := prefix + ":missing"
	keys := []string{alpha, beta, gamma, counter, ttlKey, directTTL, nonNumeric, missing}
	closed := false
	t.Cleanup(func() {
		if closed {
			return
		}
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
	health := client.Health(ctx)
	if health.Status != HealthHealthy || health.Name != "redisx-integration" || health.Component != "redis" {
		t.Fatalf("health = %#v, want healthy redisx-integration redis", health)
	}
	healthCheck := client.HealthCheck(ctx)
	if healthCheck.Status != HealthHealthy || healthCheck.Name != health.Name || healthCheck.Component != health.Component {
		t.Fatalf("health check = %#v, want healthy match %#v", healthCheck, health)
	}

	if err := client.Set(ctx, alpha, "1", 0); err != nil {
		skipIfRedisEnvironmentBlocked(t, "set", err)
		t.Fatalf("set: %v", err)
	}
	value, err := client.Get(ctx, alpha)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if value != "1" {
		t.Fatalf("get = %q, want 1", value)
	}
	ttl, err := client.TTL(ctx, alpha)
	if err != nil {
		t.Fatalf("ttl permanent alpha: %v", err)
	}
	if ttl != -time.Second {
		t.Fatalf("ttl permanent alpha = %v, want -1s", ttl)
	}

	recoveryClient, err := newRedisIntegrationClient(ctx, "redisx-integration-reconnect", db)
	if err != nil {
		t.Fatalf("new reconnect redis client with options: %v", err)
	}
	defer func() {
		if err := recoveryClient.Close(context.Background()); err != nil && !IsKind(err, ErrorKindClosed) {
			t.Fatalf("close reconnect redis client: %v", err)
		}
	}()
	if err := recoveryClient.Ping(ctx); err != nil {
		t.Fatalf("reconnect ping: %v", err)
	}
	recoveryHealth := recoveryClient.Health(ctx)
	if recoveryHealth.Status != HealthHealthy || recoveryHealth.Name != "redisx-integration-reconnect" || recoveryHealth.Component != "redis" {
		t.Fatalf("reconnect health = %#v, want healthy redisx-integration-reconnect redis", recoveryHealth)
	}
	recoveredValue, err := recoveryClient.Get(ctx, alpha)
	if err != nil {
		t.Fatalf("reconnect get alpha: %v", err)
	}
	if recoveredValue != "1" {
		t.Fatalf("reconnect get alpha = %q, want 1", recoveredValue)
	}

	if err := client.Set(ctx, directTTL, "direct", time.Minute); err != nil {
		t.Fatalf("set direct ttl: %v", err)
	}
	ttl, err = client.TTL(ctx, directTTL)
	if err != nil {
		t.Fatalf("ttl direct ttl: %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("ttl direct ttl = %v, want positive", ttl)
	}

	ttl, err = client.TTL(ctx, missing)
	if err != nil {
		t.Fatalf("ttl missing: %v", err)
	}
	if ttl != -2*time.Second {
		t.Fatalf("ttl missing = %v, want -2s", ttl)
	}

	if _, err := client.Get(ctx, missing); !IsKind(err, ErrorKindNil) {
		t.Fatalf("missing get kind = %v, want nil", err)
	}

	if err := client.MSet(ctx, map[string]string{beta: "2", gamma: "3"}); err != nil {
		t.Fatalf("mset: %v", err)
	}
	values, err := client.MGet(ctx, alpha, beta, missing, gamma)
	if err != nil {
		t.Fatalf("mget: %v", err)
	}
	if len(values) != 4 || !values[0].Found || values[0].Value != "1" || !values[1].Found || values[1].Value != "2" || values[2].Found || !values[3].Found || values[3].Value != "3" {
		t.Fatalf("unexpected mget values: %#v", values)
	}

	if err := client.Set(ctx, counter, "0", 0); err != nil {
		t.Fatalf("set counter: %v", err)
	}
	if value, err := client.Incr(ctx, counter); err != nil || value != 1 {
		t.Fatalf("incr = %d, %v; want 1, nil", value, err)
	}
	if value, err := client.Decr(ctx, counter); err != nil || value != 0 {
		t.Fatalf("decr = %d, %v; want 0, nil", value, err)
	}
	if err := client.Set(ctx, nonNumeric, "not-an-integer", 0); err != nil {
		t.Fatalf("set non-numeric: %v", err)
	}
	if _, err := client.Incr(ctx, nonNumeric); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("incr non-numeric kind = %v, want validation", err)
	}
	if _, err := client.Decr(ctx, nonNumeric); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("decr non-numeric kind = %v, want validation", err)
	}

	if err := client.Set(ctx, ttlKey, "ttl", 0); err != nil {
		t.Fatalf("set ttl: %v", err)
	}
	updated, err := client.Expire(ctx, ttlKey, time.Minute)
	if err != nil {
		t.Fatalf("expire: %v", err)
	}
	if !updated {
		t.Fatal("expected expire to update existing key")
	}
	ttl, err = client.TTL(ctx, ttlKey)
	if err != nil {
		t.Fatalf("ttl: %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("ttl = %v, want positive", ttl)
	}
	updated, err = client.Expire(ctx, missing, time.Minute)
	if err != nil {
		t.Fatalf("expire missing: %v", err)
	}
	if updated {
		t.Fatal("expire missing updated = true, want false")
	}

	count, err := client.Exists(ctx, alpha, beta, gamma, counter, ttlKey, directTTL, nonNumeric, missing)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if count != 7 {
		t.Fatalf("exists = %d, want 7", count)
	}
	deleted, err := client.Del(ctx, alpha, beta, gamma, counter, ttlKey, directTTL, nonNumeric, missing)
	if err != nil {
		t.Fatalf("del: %v", err)
	}
	if deleted != 7 {
		t.Fatalf("del = %d, want 7", deleted)
	}

	if err := client.Close(ctx); err != nil {
		t.Fatalf("close redis client: %v", err)
	}
	closed = true
	if err := client.Ping(ctx); !IsKind(err, ErrorKindClosed) {
		t.Fatalf("ping after close kind = %v, want closed", err)
	}
}

func TestRedisIntegrationPersistenceRecoveryWithEnv(t *testing.T) {
	if os.Getenv("REDISX_INTEGRATION") != "1" {
		t.Skip("set REDISX_INTEGRATION=1 with REDISX_REDIS_* to run real Redis integration")
	}
	if os.Getenv("REDISX_PERSISTENCE_RECOVERY") != "1" {
		t.Skip("set REDISX_PERSISTENCE_RECOVERY=1 to run persistence recovery integration")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := redisIntegrationDB(t)
	client, err := newRedisIntegrationClient(ctx, "redisx-persistence", db)
	if err != nil {
		t.Fatalf("new redis persistence client with options: %v", err)
	}
	defer func() {
		if err := client.Close(context.Background()); err != nil && !IsKind(err, ErrorKindClosed) {
			t.Fatalf("close persistence redis client: %v", err)
		}
	}()

	if err := client.Ping(ctx); err != nil {
		skipIfRedisEnvironmentBlocked(t, "ping", err)
		t.Fatalf("ping persistence redis: %v", err)
	}

	key := envOrDefault("REDISX_PERSISTENCE_KEY", "redisx:integration:persistence:"+strconv.FormatInt(time.Now().UnixNano(), 10))
	value := envOrDefault("REDISX_PERSISTENCE_VALUE", "survives-redis-restart")

	if os.Getenv("REDISX_PERSISTENCE_EXPECT_EXISTING") == "1" {
		recovered, err := client.Get(ctx, key)
		if err != nil {
			t.Fatalf("get persisted key after restart: %v", err)
		}
		if recovered != value {
			t.Fatalf("persisted key = %q, want configured value", recovered)
		}
		ttl, err := client.TTL(ctx, key)
		if err != nil {
			t.Fatalf("ttl persisted key: %v", err)
		}
		if ttl != -time.Second {
			t.Fatalf("ttl persisted key = %v, want -1s", ttl)
		}
		if os.Getenv("REDISX_PERSISTENCE_CLEANUP") != "0" {
			if deleted, err := client.Del(ctx, key); err != nil || deleted != 1 {
				t.Fatalf("cleanup persisted key deleted = %d, err = %v; want 1, nil", deleted, err)
			}
		}
		return
	}

	if err := client.Set(ctx, key, value, 0); err != nil {
		skipIfRedisEnvironmentBlocked(t, "set persistence key", err)
		t.Fatalf("set persistence key: %v", err)
	}
	recovered, err := client.Get(ctx, key)
	if err != nil {
		t.Fatalf("get persistence key before restart: %v", err)
	}
	if recovered != value {
		t.Fatalf("persistence key = %q, want configured value", recovered)
	}
	ttl, err := client.TTL(ctx, key)
	if err != nil {
		t.Fatalf("ttl persistence key before restart: %v", err)
	}
	if ttl != -time.Second {
		t.Fatalf("ttl persistence key before restart = %v, want -1s", ttl)
	}
}

func newRedisIntegrationClient(ctx context.Context, name string, db int) (*Client, error) {
	return NewWithOptions(ctx, Options{Config: Config{
		Name: name,
		Redis: RedisConfig{
			Addr:         os.Getenv("REDISX_REDIS_ADDR"),
			Username:     os.Getenv("REDISX_REDIS_USERNAME"),
			Password:     os.Getenv("REDISX_REDIS_PASSWORD"),
			DB:           db,
			DialTimeout:  time.Second,
			ReadTimeout:  time.Second,
			WriteTimeout: time.Second,
			PoolSize:     2,
			MinIdleConns: 1,
			MaxRetries:   1,
		},
	}})
}

func redisIntegrationDB(t *testing.T) int {
	t.Helper()
	dbText := envOrDefault("REDISX_REDIS_DB", "0")
	db, err := strconv.Atoi(dbText)
	if err != nil {
		t.Fatalf("parse REDISX_REDIS_DB: %v", err)
	}
	return db
}

func envOrDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func skipIfRedisEnvironmentBlocked(t *testing.T, op string, err error) {
	t.Helper()
	if IsKind(err, ErrorKindLoading) || IsKind(err, ErrorKindTryAgain) {
		t.Skipf("real Redis %s reached the server but the environment is not ready: %v", op, err)
	}
}
