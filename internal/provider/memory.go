package provider

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"
)

var (
	ErrClosed = errors.New("provider is closed")
	ErrNil    = errors.New("redis nil")
)

type Memory struct {
	mu     sync.Mutex
	closed bool
	clock  func() time.Time
	items  map[string]entry
}

type entry struct {
	value     string
	expiresAt time.Time
}

func NewMemory() *Memory {
	return &Memory{clock: time.Now, items: make(map[string]entry)}
}

func (m *Memory) Ping(ctx context.Context) error {
	return m.withState(ctx, func(now time.Time) error { return nil })
}

func (m *Memory) Close(ctx context.Context) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	return nil
}

func (m *Memory) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := m.withState(ctx, func(now time.Time) error {
		item, ok := m.live(key, now)
		if !ok {
			return ErrNil
		}
		value = item.value
		return nil
	})
	return value, err
}

func (m *Memory) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return m.withState(ctx, func(now time.Time) error {
		m.items[key] = entry{value: value, expiresAt: expiresAt(now, ttl)}
		return nil
	})
}

func (m *Memory) Del(ctx context.Context, keys ...string) (int64, error) {
	var deleted int64
	err := m.withState(ctx, func(now time.Time) error {
		for _, key := range keys {
			if _, ok := m.live(key, now); ok {
				delete(m.items, key)
				deleted++
			}
		}
		return nil
	})
	return deleted, err
}

func (m *Memory) Exists(ctx context.Context, keys ...string) (int64, error) {
	var count int64
	err := m.withState(ctx, func(now time.Time) error {
		for _, key := range keys {
			if _, ok := m.live(key, now); ok {
				count++
			}
		}
		return nil
	})
	return count, err
}

func (m *Memory) Expire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	var updated bool
	err := m.withState(ctx, func(now time.Time) error {
		item, ok := m.live(key, now)
		if !ok {
			return nil
		}
		item.expiresAt = expiresAt(now, ttl)
		m.items[key] = item
		updated = true
		return nil
	})
	return updated, err
}

func (m *Memory) TTL(ctx context.Context, key string) (time.Duration, error) {
	var ttl time.Duration
	err := m.withState(ctx, func(now time.Time) error {
		item, ok := m.live(key, now)
		if !ok {
			ttl = -2 * time.Second
			return nil
		}
		if item.expiresAt.IsZero() {
			ttl = -time.Second
			return nil
		}
		ttl = item.expiresAt.Sub(now)
		if ttl < 0 {
			delete(m.items, key)
			ttl = -2 * time.Second
		}
		return nil
	})
	return ttl, err
}

func (m *Memory) MGet(ctx context.Context, keys ...string) ([]Value, error) {
	values := make([]Value, len(keys))
	err := m.withState(ctx, func(now time.Time) error {
		for i, key := range keys {
			item, ok := m.live(key, now)
			if ok {
				values[i] = Value{Value: item.value, Found: true}
			}
		}
		return nil
	})
	return values, err
}

func (m *Memory) MSet(ctx context.Context, values map[string]string) error {
	return m.withState(ctx, func(now time.Time) error {
		for key, value := range values {
			m.items[key] = entry{value: value}
		}
		return nil
	})
}

func (m *Memory) Incr(ctx context.Context, key string) (int64, error) {
	return m.add(ctx, key, 1)
}

func (m *Memory) Decr(ctx context.Context, key string) (int64, error) {
	return m.add(ctx, key, -1)
}

func (m *Memory) add(ctx context.Context, key string, delta int64) (int64, error) {
	var next int64
	err := m.withState(ctx, func(now time.Time) error {
		item, ok := m.live(key, now)
		if ok {
			parsed, err := strconv.ParseInt(item.value, 10, 64)
			if err != nil {
				return err
			}
			next = parsed + delta
			item.value = strconv.FormatInt(next, 10)
			m.items[key] = item
			return nil
		}
		next = delta
		m.items[key] = entry{value: strconv.FormatInt(next, 10)}
		return nil
	})
	return next, err
}

func (m *Memory) withState(ctx context.Context, fn func(time.Time) error) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrClosed
	}
	return fn(m.clock())
}

func (m *Memory) live(key string, now time.Time) (entry, bool) {
	item, ok := m.items[key]
	if !ok {
		return entry{}, false
	}
	if !item.expiresAt.IsZero() && !now.Before(item.expiresAt) {
		delete(m.items, key)
		return entry{}, false
	}
	return item, true
}

func ctxErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	return ctx.Err()
}

func expiresAt(now time.Time, ttl time.Duration) time.Time {
	if ttl <= 0 {
		return time.Time{}
	}
	return now.Add(ttl)
}
