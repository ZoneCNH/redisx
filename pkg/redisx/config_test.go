package redisx

import (
	"testing"
	"time"
)

func TestConfigValidateRequiresName(t *testing.T) {
	err := Config{Timeout: time.Second}.Validate()
	if err == nil {
		t.Fatal("expected missing name to fail validation")
	}
	if !IsKind(err, ErrorKindValidation) {
		t.Fatalf("expected validation error, got %T %[1]v", err)
	}
}

func TestConfigValidateRejectsNegativeTimeout(t *testing.T) {
	err := Config{Name: "redisx", Timeout: -time.Second}.Validate()
	if err == nil {
		t.Fatal("expected negative timeout to fail validation")
	}
	if !IsKind(err, ErrorKindValidation) {
		t.Fatalf("expected validation error, got %T %[1]v", err)
	}
}

func TestConfigSanitizeMasksSecret(t *testing.T) {
	sanitized := Config{
		Name:    "redisx",
		Timeout: time.Second,
		Secret:  "plain-text",
		Redis: RedisConfig{
			Addr:     "127.0.0.1:6379",
			Username: "default",
			Password: "redis-password",
			DB:       2,
		},
	}.Sanitize()
	if sanitized.Secret != "***" {
		t.Fatalf("expected masked secret, got %q", sanitized.Secret)
	}
	if sanitized.Redis.Password != "***" {
		t.Fatalf("expected masked redis password, got %q", sanitized.Redis.Password)
	}
	if sanitized.Name != "redisx" {
		t.Fatalf("expected name to be preserved, got %q", sanitized.Name)
	}
	if sanitized.Redis.Addr != "127.0.0.1:6379" || sanitized.Redis.Username != "default" || sanitized.Redis.DB != 2 {
		t.Fatalf("expected non-secret redis config to be preserved, got %#v", sanitized.Redis)
	}
}

func TestRedisConfigValidateRequiresAddrWhenEnabled(t *testing.T) {
	err := Config{Name: "redisx", Redis: RedisConfig{Password: "redis-password"}}.Validate()
	if err == nil {
		t.Fatal("expected redis config without addr to fail validation")
	}
	if !IsKind(err, ErrorKindValidation) {
		t.Fatalf("expected validation error, got %T %[1]v", err)
	}
}

func TestRedisConfigValidateRejectsNegativeValues(t *testing.T) {
	cases := map[string]RedisConfig{
		"db":            {Addr: "127.0.0.1:6379", DB: -1},
		"dial_timeout":  {Addr: "127.0.0.1:6379", DialTimeout: -time.Second},
		"read_timeout":  {Addr: "127.0.0.1:6379", ReadTimeout: -time.Second},
		"write_timeout": {Addr: "127.0.0.1:6379", WriteTimeout: -time.Second},
		"pool_size":     {Addr: "127.0.0.1:6379", PoolSize: -1},
		"min_idle":      {Addr: "127.0.0.1:6379", MinIdleConns: -1},
		"max_retries":   {Addr: "127.0.0.1:6379", MaxRetries: -1},
	}
	for name, redisConfig := range cases {
		t.Run(name, func(t *testing.T) {
			err := Config{Name: "redisx", Redis: redisConfig}.Validate()
			if err == nil {
				t.Fatal("expected negative redis config value to fail validation")
			}
			if !IsKind(err, ErrorKindValidation) {
				t.Fatalf("expected validation error, got %T %[1]v", err)
			}
		})
	}
}
