package redisx

import (
	"errors"
	"strings"
	"time"

	"github.com/ZoneCNH/redisx/internal/sanitize"
	"github.com/ZoneCNH/redisx/internal/validation"
)

type Config struct {
	Name    string
	Timeout time.Duration
	Secret  string
	Redis   RedisConfig
}

type RedisConfig struct {
	Addr         string
	Username     string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
}

type SanitizedConfig struct {
	Name    string
	Timeout time.Duration
	Secret  string
	Redis   SanitizedRedisConfig
}

type SanitizedRedisConfig struct {
	Addr         string
	Username     string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
}

func (c Config) Validate() error {
	if err := validation.RequireNonEmpty("name", c.Name); err != nil {
		return validationError("Config.Validate", err.Error(), err)
	}
	if c.Timeout < 0 {
		err := errors.New("timeout must not be negative")
		return validationError("Config.Validate", err.Error(), err)
	}
	if err := c.Redis.Validate(); err != nil {
		return err
	}
	return nil
}

func (c Config) Sanitize() SanitizedConfig {
	return SanitizedConfig{
		Name:    c.Name,
		Timeout: c.Timeout,
		Secret:  sanitize.Secret(c.Secret),
		Redis:   c.Redis.Sanitize(),
	}
}

func (c RedisConfig) Enabled() bool {
	return strings.TrimSpace(c.Addr) != "" ||
		c.Username != "" ||
		c.Password != "" ||
		c.DB != 0 ||
		c.DialTimeout != 0 ||
		c.ReadTimeout != 0 ||
		c.WriteTimeout != 0 ||
		c.PoolSize != 0 ||
		c.MinIdleConns != 0 ||
		c.MaxRetries != 0
}

func (c RedisConfig) Validate() error {
	const op = "RedisConfig.Validate"
	if !c.Enabled() {
		return nil
	}
	if strings.TrimSpace(c.Addr) == "" {
		err := errors.New("redis addr is required")
		return validationError(op, err.Error(), err)
	}
	checks := []struct {
		name string
		bad  bool
	}{
		{name: "redis db must not be negative", bad: c.DB < 0},
		{name: "redis dial timeout must not be negative", bad: c.DialTimeout < 0},
		{name: "redis read timeout must not be negative", bad: c.ReadTimeout < 0},
		{name: "redis write timeout must not be negative", bad: c.WriteTimeout < 0},
		{name: "redis pool size must not be negative", bad: c.PoolSize < 0},
		{name: "redis min idle conns must not be negative", bad: c.MinIdleConns < 0},
		{name: "redis max retries must not be negative", bad: c.MaxRetries < 0},
	}
	for _, check := range checks {
		if check.bad {
			err := errors.New(check.name)
			return validationError(op, err.Error(), err)
		}
	}
	return nil
}

func (c RedisConfig) Sanitize() SanitizedRedisConfig {
	return SanitizedRedisConfig{
		Addr:         c.Addr,
		Username:     c.Username,
		Password:     sanitize.Secret(c.Password),
		DB:           c.DB,
		DialTimeout:  c.DialTimeout,
		ReadTimeout:  c.ReadTimeout,
		WriteTimeout: c.WriteTimeout,
		PoolSize:     c.PoolSize,
		MinIdleConns: c.MinIdleConns,
		MaxRetries:   c.MaxRetries,
	}
}
