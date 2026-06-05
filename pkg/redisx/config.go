package redisx

import (
	"errors"
	"time"

	"github.com/ZoneCNH/redisx/internal/sanitize"
	"github.com/ZoneCNH/redisx/internal/validation"
)

type Config struct {
	Name    string
	Timeout time.Duration
	Secret  string
}

type SanitizedConfig struct {
	Name    string
	Timeout time.Duration
	Secret  string
}

type Options struct {
	Name           string
	Address        string
	Username       string
	Password       string
	DB             int
	TLS            bool
	ConnectTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	PoolSize       int
}

type SanitizedOptions struct {
	Name           string
	Address        string
	Username       string
	Password       string
	DB             int
	TLS            bool
	ConnectTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	PoolSize       int
}

func (o Options) Validate() error {
	if o.DB < 0 {
		err := errors.New("db must not be negative")
		return validationError("Options.Validate", err.Error(), err)
	}
	if o.ConnectTimeout < 0 {
		err := errors.New("connect timeout must not be negative")
		return validationError("Options.Validate", err.Error(), err)
	}
	if o.ReadTimeout < 0 {
		err := errors.New("read timeout must not be negative")
		return validationError("Options.Validate", err.Error(), err)
	}
	if o.WriteTimeout < 0 {
		err := errors.New("write timeout must not be negative")
		return validationError("Options.Validate", err.Error(), err)
	}
	if o.PoolSize < 0 {
		err := errors.New("pool size must not be negative")
		return validationError("Options.Validate", err.Error(), err)
	}
	return nil
}

func (o Options) ToConfig() Config {
	name := o.Name
	if name == "" {
		name = "redisx"
	}
	timeout := o.ConnectTimeout
	if timeout == 0 {
		timeout = o.ReadTimeout
	}
	return Config{
		Name:    name,
		Timeout: timeout,
		Secret:  o.Password,
	}
}

func (o Options) Sanitize() SanitizedOptions {
	return SanitizedOptions{
		Name:           o.Name,
		Address:        o.Address,
		Username:       o.Username,
		Password:       sanitize.Secret(o.Password),
		DB:             o.DB,
		TLS:            o.TLS,
		ConnectTimeout: o.ConnectTimeout,
		ReadTimeout:    o.ReadTimeout,
		WriteTimeout:   o.WriteTimeout,
		PoolSize:       o.PoolSize,
	}
}

func (c Config) Validate() error {
	if err := validation.RequireNonEmpty("name", c.Name); err != nil {
		return validationError("Config.Validate", err.Error(), err)
	}
	if c.Timeout < 0 {
		err := errors.New("timeout must not be negative")
		return validationError("Config.Validate", err.Error(), err)
	}
	return nil
}

func (c Config) Sanitize() SanitizedConfig {
	return SanitizedConfig{
		Name:    c.Name,
		Timeout: c.Timeout,
		Secret:  sanitize.Secret(c.Secret),
	}
}
