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
	hashKey := prefix + ":hash"
	listKey := prefix + ":list"
	ttlKey := prefix + ":ttl"
	directTTL := prefix + ":direct-ttl"
	nonNumeric := prefix + ":non-numeric"
	pipeString := prefix + ":pipe:string"
	pipeHash := prefix + ":pipe:hash"
	pipeList := prefix + ":pipe:list"
	pipeCounter := prefix + ":pipe:counter"
	pipeMissing := prefix + ":pipe:missing"
	pipeMissingList := prefix + ":pipe:missing-list"
	lockKey := prefix + ":lock"
	rateKey := prefix + ":rate"
	missing := prefix + ":missing"
	keys := []string{alpha, beta, gamma, counter, ttlKey, directTTL, nonNumeric, hashKey, listKey, pipeString, pipeHash, pipeList, pipeCounter, lockKey, rateKey, missing}
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

	if added, err := client.HSet(ctx, hashKey, map[string]string{"name": "redisx", "version": "1"}); err != nil || added != 2 {
		t.Fatalf("hset = %d, %v; want 2, nil", added, err)
	}
	if value, err := client.HGet(ctx, hashKey, "name"); err != nil || value != "redisx" {
		t.Fatalf("hget = %q, %v; want redisx, nil", value, err)
	}
	if length, err := client.RPush(ctx, listKey, "a", "b"); err != nil || length != 2 {
		t.Fatalf("rpush = %d, %v; want 2, nil", length, err)
	}
	if values, err := client.LRange(ctx, listKey, 0, -1); err != nil || len(values) != 2 || values[0] != "a" || values[1] != "b" {
		t.Fatalf("lrange = %#v, %v; want [a b], nil", values, err)
	}
	results, err := client.Pipeline(ctx, []PipelineCommand{
		{Type: PipelineSet, Key: pipeString, Value: "pipe"},
		{Type: PipelineHSet, Key: pipeHash, Values: map[string]string{"field": "value"}},
		{Type: PipelineRPush, Key: pipeList, ListValues: []string{"x", "y"}},
		{Type: PipelineIncr, Key: pipeCounter},
		{Type: PipelineGet, Key: pipeString},
		{Type: PipelineGet, Key: pipeMissing},
		{Type: PipelineHGet, Key: pipeHash, Field: "field"},
		{Type: PipelineHGet, Key: pipeHash, Field: "missing"},
		{Type: PipelineLRange, Key: pipeList, Start: 0, Stop: -1},
		{Type: PipelineLRange, Key: pipeMissingList, Start: 0, Stop: -1},
	})
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}
	if len(results) != 10 || results[3].Int != 1 {
		t.Fatalf("pipeline results = %#v", results)
	}
	if !results[4].Found || results[4].Value != "pipe" || results[5].Found || !results[6].Found || results[6].Value != "value" || results[7].Found {
		t.Fatalf("pipeline read results = %#v", results[4:8])
	}
	if len(results[8].Strings) != 2 || results[8].Strings[0] != "x" || results[8].Strings[1] != "y" || len(results[9].Strings) != 0 {
		t.Fatalf("pipeline list results = %#v", results[8:])
	}
	if acquired, err := client.AcquireLock(ctx, lockKey, "token", time.Minute); err != nil || !acquired {
		t.Fatalf("acquire lock = %v, %v; want true, nil", acquired, err)
	}
	if acquired, err := client.AcquireLock(ctx, lockKey, "other", time.Minute); err != nil || acquired {
		t.Fatalf("acquire held lock = %v, %v; want false, nil", acquired, err)
	}
	if released, err := client.ReleaseLock(ctx, lockKey, "token"); err != nil || !released {
		t.Fatalf("release lock = %v, %v; want true, nil", released, err)
	}
	first, err := client.FixedWindowRateLimit(ctx, rateKey, 1, time.Minute)
	if err != nil {
		t.Fatalf("rate first: %v", err)
	}
	second, err := client.FixedWindowRateLimit(ctx, rateKey, 1, time.Minute)
	if err != nil {
		t.Fatalf("rate second: %v", err)
	}
	if !first.Allowed || first.Remaining != 0 || second.Allowed || second.Count != 2 || second.ResetAfter <= 0 {
		t.Fatalf("unexpected rate results: first=%#v second=%#v", first, second)
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

	count, err := client.Exists(ctx, alpha, beta, gamma, counter, ttlKey, directTTL, nonNumeric, hashKey, listKey, pipeString, pipeHash, pipeList, pipeCounter, rateKey, missing)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if count != 14 {
		t.Fatalf("exists = %d, want 14", count)
	}
	deleted, err := client.Del(ctx, keys...)
	if err != nil {
		t.Fatalf("del: %v", err)
	}
	if deleted != 14 {
		t.Fatalf("del = %d, want 14", deleted)
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

	stringKey := envOrDefault("REDISX_PERSISTENCE_KEY", "redisx:integration:persistence:"+strconv.FormatInt(time.Now().UnixNano(), 10))
	value := envOrDefault("REDISX_PERSISTENCE_VALUE", "survives-redis-restart")
	msetKey := envOrDefault("REDISX_PERSISTENCE_MSET_KEY", stringKey+":mset")
	hashKey := envOrDefault("REDISX_PERSISTENCE_HASH_KEY", stringKey+":hash")
	listKey := envOrDefault("REDISX_PERSISTENCE_LIST_KEY", stringKey+":list")
	counterKey := envOrDefault("REDISX_PERSISTENCE_COUNTER_KEY", stringKey+":counter")
	pipeStringKey := envOrDefault("REDISX_PERSISTENCE_PIPE_STRING_KEY", stringKey+":pipe:string")
	pipeHashKey := envOrDefault("REDISX_PERSISTENCE_PIPE_HASH_KEY", stringKey+":pipe:hash")
	pipeListKey := envOrDefault("REDISX_PERSISTENCE_PIPE_LIST_KEY", stringKey+":pipe:list")
	persistenceKeys := []string{stringKey, msetKey, hashKey, listKey, counterKey, pipeStringKey, pipeHashKey, pipeListKey}

	if os.Getenv("REDISX_PERSISTENCE_EXPECT_EXISTING") == "1" {
		recovered, err := client.Get(ctx, stringKey)
		if err != nil {
			t.Fatalf("get persisted key after restart: %v", err)
		}
		if recovered != value {
			t.Fatalf("persisted key = %q, want configured value", recovered)
		}
		msetValues, err := client.MGet(ctx, msetKey, pipeStringKey)
		if err != nil {
			t.Fatalf("mget persisted keys after restart: %v", err)
		}
		if len(msetValues) != 2 || !msetValues[0].Found || msetValues[0].Value != value+"-mset" || !msetValues[1].Found || msetValues[1].Value != "pipe" {
			t.Fatalf("persisted mget = %#v, want mset value and stable pipeline value", msetValues)
		}
		ttl, err := client.TTL(ctx, stringKey)
		if err != nil {
			t.Fatalf("ttl persisted key: %v", err)
		}
		if ttl != -time.Second {
			t.Fatalf("ttl persisted key = %v, want -1s", ttl)
		}
		hashValues, err := client.HGetAll(ctx, hashKey)
		if err != nil {
			t.Fatalf("hgetall persisted hash: %v", err)
		}
		if hashValues["name"] != value+"-hash" || hashValues["state"] != "persistent" {
			t.Fatalf("persisted hash = %#v, want configured persistence fields", hashValues)
		}
		listValues, err := client.LRange(ctx, listKey, 0, -1)
		if err != nil {
			t.Fatalf("lrange persisted list: %v", err)
		}
		if len(listValues) != 2 || listValues[0] != value+"-a" || listValues[1] != value+"-b" {
			t.Fatalf("persisted list = %#v, want configured persistence list", listValues)
		}
		counterValue, err := client.Get(ctx, counterKey)
		if err != nil {
			t.Fatalf("get persisted counter: %v", err)
		}
		if counterValue != "1" {
			t.Fatalf("persisted counter = %q, want 1", counterValue)
		}
		pipeValue, err := client.Get(ctx, pipeStringKey)
		if err != nil {
			t.Fatalf("get persisted pipeline string: %v", err)
		}
		if pipeValue != "pipe" {
			t.Fatalf("persisted pipeline string = %q, want pipe", pipeValue)
		}
		pipeHashValue, err := client.HGet(ctx, pipeHashKey, "field")
		if err != nil {
			t.Fatalf("hget persisted pipeline hash: %v", err)
		}
		if pipeHashValue != "value" {
			t.Fatalf("persisted pipeline hash = %q, want value", pipeHashValue)
		}
		pipeListValues, err := client.LRange(ctx, pipeListKey, 0, -1)
		if err != nil {
			t.Fatalf("lrange persisted pipeline list: %v", err)
		}
		if len(pipeListValues) != 2 || pipeListValues[0] != "x" || pipeListValues[1] != "y" {
			t.Fatalf("persisted pipeline list = %#v, want [x y]", pipeListValues)
		}
		if os.Getenv("REDISX_PERSISTENCE_CLEANUP") != "0" {
			if deleted, err := client.Del(ctx, persistenceKeys...); err != nil || deleted != int64(len(persistenceKeys)) {
				t.Fatalf("cleanup persisted keys deleted = %d, err = %v; want %d, nil", deleted, err, len(persistenceKeys))
			}
		}
		return
	}

	if err := client.Set(ctx, stringKey, value, 0); err != nil {
		skipIfRedisEnvironmentBlocked(t, "set persistence string", err)
		t.Fatalf("set persistence string: %v", err)
	}
	if err := client.MSet(ctx, map[string]string{msetKey: value + "-mset"}); err != nil {
		t.Fatalf("mset persistence key: %v", err)
	}
	if _, err := client.HSet(ctx, hashKey, map[string]string{"name": value + "-hash", "state": "persistent"}); err != nil {
		t.Fatalf("hset persistence hash: %v", err)
	}
	if _, err := client.RPush(ctx, listKey, value+"-a", value+"-b"); err != nil {
		t.Fatalf("rpush persistence list: %v", err)
	}
	if err := client.Set(ctx, counterKey, "0", 0); err != nil {
		t.Fatalf("set persistence counter: %v", err)
	}
	if count, err := client.Incr(ctx, counterKey); err != nil || count != 1 {
		t.Fatalf("incr persistence counter = %d, err = %v; want 1, nil", count, err)
	}
	pipelineResults, err := client.Pipeline(ctx, []PipelineCommand{
		{Type: PipelineSet, Key: pipeStringKey, Value: value + "-pipe"},
	})
	if err != nil {
		t.Fatalf("pipeline persistence key: %v", err)
	}
	if len(pipelineResults) != 1 || !pipelineResults[0].Bool {
		t.Fatalf("pipeline persistence results = %#v, want persisted pipe value", pipelineResults)
	}
	assertPersistentRedisWrites(t, ctx, client, value, stringKey, msetKey, hashKey, listKey, counterKey, pipeStringKey, pipeHashKey, pipeListKey)
}

func assertPersistentRedisWrites(t *testing.T, ctx context.Context, client *Client, value, stringKey, msetKey, hashKey, listKey, counterKey, pipeStringKey, pipeHashKey, pipeListKey string) {
	t.Helper()

	recovered, err := client.Get(ctx, stringKey)
	if err != nil {
		t.Fatalf("get persisted string: %v", err)
	}
	if recovered != value {
		t.Fatalf("persisted string = %q, want configured value", recovered)
	}

	multiValues, err := client.MGet(ctx, msetKey, pipeStringKey)
	if err != nil {
		t.Fatalf("mget persisted keys: %v", err)
	}
	if len(multiValues) != 2 || !multiValues[0].Found || multiValues[0].Value != value+"-mset" || !multiValues[1].Found || multiValues[1].Value != value+"-pipe" {
		t.Fatalf("persisted mget = %#v, want mset and pipeline values", multiValues)
	}

	hashValue, err := client.HGet(ctx, hashKey, "name")
	if err != nil {
		t.Fatalf("hget persisted hash: %v", err)
	}
	if hashValue != value+"-hash" {
		t.Fatalf("persisted hash = %q, want configured hash value", hashValue)
	}

	listValues, err := client.LRange(ctx, listKey, 0, -1)
	if err != nil {
		t.Fatalf("lrange persisted list: %v", err)
	}
	if len(listValues) != 2 || listValues[0] != value+"-a" || listValues[1] != value+"-b" {
		t.Fatalf("persisted list = %#v, want configured list", listValues)
	}

	counterValue, err := client.Get(ctx, counterKey)
	if err != nil {
		t.Fatalf("get persisted counter: %v", err)
	}
	if counterValue != "1" {
		t.Fatalf("persisted counter = %q, want 1", counterValue)
	}

	if _, err := client.Del(ctx, pipeHashKey, pipeListKey); err != nil {
		t.Fatalf("clear persistence pipeline fixtures: %v", err)
	}
	results, err := client.Pipeline(ctx, []PipelineCommand{
		{Type: PipelineSet, Key: pipeStringKey, Value: "pipe"},
		{Type: PipelineHSet, Key: pipeHashKey, Values: map[string]string{"field": "value"}},
		{Type: PipelineRPush, Key: pipeListKey, ListValues: []string{"x", "y"}},
	})
	if err != nil {
		t.Fatalf("pipeline persistence primitives: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("pipeline persistence results = %#v, want 3 results", results)
	}
	pipeValue, err := client.Get(ctx, pipeStringKey)
	if err != nil {
		t.Fatalf("get persistence pipeline string fixture: %v", err)
	}
	if pipeValue != "pipe" {
		t.Fatalf("persistence pipeline string fixture = %q, want pipe", pipeValue)
	}
	pipeHashValue, err := client.HGet(ctx, pipeHashKey, "field")
	if err != nil {
		t.Fatalf("hget persistence pipeline hash fixture: %v", err)
	}
	if pipeHashValue != "value" {
		t.Fatalf("persistence pipeline hash fixture = %q, want value", pipeHashValue)
	}
	pipeListValues, err := client.LRange(ctx, pipeListKey, 0, -1)
	if err != nil {
		t.Fatalf("lrange persistence pipeline list fixture: %v", err)
	}
	if len(pipeListValues) != 2 || pipeListValues[0] != "x" || pipeListValues[1] != "y" {
		t.Fatalf("persistence pipeline list fixture = %#v, want [x y]", pipeListValues)
	}

	for _, key := range []string{stringKey, msetKey, hashKey, listKey, counterKey, pipeStringKey, pipeHashKey, pipeListKey} {
		ttl, err := client.TTL(ctx, key)
		if err != nil {
			t.Fatalf("ttl persisted key %q: %v", key, err)
		}
		if ttl != -time.Second {
			t.Fatalf("ttl persisted key %q = %v, want -1s", key, ttl)
		}
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
