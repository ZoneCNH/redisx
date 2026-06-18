package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

type Memory struct {
	mu     sync.Mutex
	closed bool
	clock  func() time.Time
	items  map[string]entry
}

type entryKind uint8

const (
	entryKindString entryKind = iota + 1
	entryKindHash
	entryKindList
)

type entry struct {
	kind      entryKind
	value     string
	hash      map[string]string
	list      []string
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
		if item.kind != entryKindString {
			return ErrWrongType
		}
		value = item.value
		return nil
	})
	return value, err
}

func (m *Memory) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return m.withState(ctx, func(now time.Time) error {
		m.items[key] = entry{kind: entryKindString, value: value, expiresAt: expiresAt(now, ttl)}
		return nil
	})
}

func (m *Memory) SetNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error) {
	var set bool
	err := m.withState(ctx, func(now time.Time) error {
		if _, ok := m.live(key, now); ok {
			return nil
		}
		m.items[key] = entry{kind: entryKindString, value: value, expiresAt: expiresAt(now, ttl)}
		set = true
		return nil
	})
	return set, err
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
		return nil
	})
	return ttl, err
}

func (m *Memory) MGet(ctx context.Context, keys ...string) ([]Value, error) {
	values := make([]Value, len(keys))
	err := m.withState(ctx, func(now time.Time) error {
		for i, key := range keys {
			item, ok := m.live(key, now)
			if !ok {
				continue
			}
			if item.kind != entryKindString {
				return ErrWrongType
			}
			values[i] = Value{Value: item.value, Found: true}
		}
		return nil
	})
	return values, err
}

func (m *Memory) MSet(ctx context.Context, values map[string]string) error {
	return m.withState(ctx, func(now time.Time) error {
		for key, value := range values {
			m.items[key] = entry{kind: entryKindString, value: value}
		}
		return nil
	})
}

func (m *Memory) Incr(ctx context.Context, key string) (int64, error) { return m.add(ctx, key, 1) }
func (m *Memory) Decr(ctx context.Context, key string) (int64, error) { return m.add(ctx, key, -1) }

func (m *Memory) add(ctx context.Context, key string, delta int64) (int64, error) {
	var next int64
	err := m.withState(ctx, func(now time.Time) error {
		item, ok := m.live(key, now)
		if ok {
			if item.kind != entryKindString {
				return ErrWrongType
			}
			parsed, err := strconv.ParseInt(item.value, 10, 64)
			if err != nil {
				return ErrInvalidInt
			}
			next = parsed + delta
			item.value = strconv.FormatInt(next, 10)
			m.items[key] = item
			return nil
		}
		next = delta
		m.items[key] = entry{kind: entryKindString, value: strconv.FormatInt(next, 10)}
		return nil
	})
	return next, err
}

func (m *Memory) HSet(ctx context.Context, key string, values map[string]string) (int64, error) {
	var added int64
	err := m.withState(ctx, func(now time.Time) error {
		item, ok := m.live(key, now)
		if !ok {
			item = entry{kind: entryKindHash, hash: map[string]string{}}
		} else if item.kind != entryKindHash {
			return ErrWrongType
		}
		if item.hash == nil {
			item.hash = map[string]string{}
		}
		for field, value := range values {
			if _, exists := item.hash[field]; !exists {
				added++
			}
			item.hash[field] = value
		}
		m.items[key] = item
		return nil
	})
	return added, err
}

func (m *Memory) HGet(ctx context.Context, key string, field string) (string, error) {
	var value string
	err := m.withState(ctx, func(now time.Time) error {
		item, ok := m.live(key, now)
		if !ok {
			return ErrNil
		}
		if item.kind != entryKindHash {
			return ErrWrongType
		}
		fieldValue, ok := item.hash[field]
		if !ok {
			return ErrNil
		}
		value = fieldValue
		return nil
	})
	return value, err
}

func (m *Memory) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	values := map[string]string{}
	err := m.withState(ctx, func(now time.Time) error {
		item, ok := m.live(key, now)
		if !ok {
			return nil
		}
		if item.kind != entryKindHash {
			return ErrWrongType
		}
		values = cloneMap(item.hash)
		return nil
	})
	return values, err
}

func (m *Memory) HDel(ctx context.Context, key string, fields ...string) (int64, error) {
	var deleted int64
	err := m.withState(ctx, func(now time.Time) error {
		item, ok := m.live(key, now)
		if !ok {
			return nil
		}
		if item.kind != entryKindHash {
			return ErrWrongType
		}
		for _, field := range fields {
			if _, ok := item.hash[field]; ok {
				delete(item.hash, field)
				deleted++
			}
		}
		m.items[key] = item
		return nil
	})
	return deleted, err
}

func (m *Memory) LPush(ctx context.Context, key string, values ...string) (int64, error) {
	return m.push(ctx, key, true, values...)
}

func (m *Memory) RPush(ctx context.Context, key string, values ...string) (int64, error) {
	return m.push(ctx, key, false, values...)
}

func (m *Memory) push(ctx context.Context, key string, left bool, values ...string) (int64, error) {
	var length int64
	err := m.withState(ctx, func(now time.Time) error {
		item, ok := m.live(key, now)
		if !ok {
			item = entry{kind: entryKindList}
		} else if item.kind != entryKindList {
			return ErrWrongType
		}
		if left {
			for _, value := range values {
				item.list = append([]string{value}, item.list...)
			}
		} else {
			item.list = append(item.list, values...)
		}
		length = int64(len(item.list))
		m.items[key] = item
		return nil
	})
	return length, err
}

func (m *Memory) LRange(ctx context.Context, key string, start int64, stop int64) ([]string, error) {
	values := []string{}
	err := m.withState(ctx, func(now time.Time) error {
		item, ok := m.live(key, now)
		if !ok {
			return nil
		}
		if item.kind != entryKindList {
			return ErrWrongType
		}
		values = listRange(item.list, start, stop)
		return nil
	})
	return values, err
}

func (m *Memory) LLen(ctx context.Context, key string) (int64, error) {
	var length int64
	err := m.withState(ctx, func(now time.Time) error {
		item, ok := m.live(key, now)
		if !ok {
			return nil
		}
		if item.kind != entryKindList {
			return ErrWrongType
		}
		length = int64(len(item.list))
		return nil
	})
	return length, err
}

func (m *Memory) LPop(ctx context.Context, key string) (string, error) {
	return m.pop(ctx, key, true)
}

func (m *Memory) RPop(ctx context.Context, key string) (string, error) {
	return m.pop(ctx, key, false)
}

func (m *Memory) pop(ctx context.Context, key string, left bool) (string, error) {
	var value string
	err := m.withState(ctx, func(now time.Time) error {
		item, ok := m.live(key, now)
		if !ok {
			return ErrNil
		}
		if item.kind != entryKindList {
			return ErrWrongType
		}
		if len(item.list) == 0 {
			return ErrNil
		}
		if left {
			value = item.list[0]
			item.list = item.list[1:]
		} else {
			value = item.list[len(item.list)-1]
			item.list = item.list[:len(item.list)-1]
		}
		m.items[key] = item
		return nil
	})
	return value, err
}

func (m *Memory) Pipeline(ctx context.Context, commands []PipelineCommand) ([]PipelineResult, error) {
	results := make([]PipelineResult, 0, len(commands))
	err := m.withState(ctx, func(now time.Time) error {
		for _, command := range commands {
			result, err := m.pipelineCommand(now, command)
			if err != nil {
				return err
			}
			results = append(results, result)
		}
		return nil
	})
	return results, err
}

func (m *Memory) pipelineCommand(now time.Time, command PipelineCommand) (PipelineResult, error) {
	commandType := PipelineCommandKind(command)
	result := PipelineResult{Type: commandType, Key: command.Key}
	switch commandType {
	case PipelineSet:
		m.items[command.Key] = entry{kind: entryKindString, value: command.Value, expiresAt: expiresAt(now, command.TTL)}
		result.Bool = true
	case PipelineMSet:
		for key, value := range command.Values {
			m.items[key] = entry{kind: entryKindString, value: value}
		}
		result.Bool = true
	case PipelineGet:
		item, ok := m.live(command.Key, now)
		if !ok {
			return result, nil
		}
		if item.kind != entryKindString {
			return result, ErrWrongType
		}
		result.Found = true
		result.Value = item.value
		result.Values = []Value{{Value: item.value, Found: true}}
	case PipelineHSet:
		added, err := m.hsetLocked(now, command.Key, command.Values)
		if err != nil {
			return result, err
		}
		result.Int = added
		result.Count = added
	case PipelineHGet:
		item, ok := m.live(command.Key, now)
		if !ok {
			return result, nil
		}
		if item.kind != entryKindHash {
			return result, ErrWrongType
		}
		value, ok := item.hash[command.Field]
		if !ok {
			return result, nil
		}
		result.Found = true
		result.Value = value
	case PipelineRPush:
		length, err := m.rpushLocked(now, command.Key, PipelineCommandListValues(command)...)
		if err != nil {
			return result, err
		}
		result.Int = length
		result.Count = length
	case PipelineLRange:
		item, ok := m.live(command.Key, now)
		if !ok {
			result.Strings = []string{}
			return result, nil
		}
		if item.kind != entryKindList {
			return result, ErrWrongType
		}
		result.Strings = listRange(item.list, command.Start, command.Stop)
	case PipelineIncr:
		next, err := m.addLocked(now, command.Key, 1)
		if err != nil {
			return result, err
		}
		result.Int = next
	default:
		return result, fmt.Errorf("unsupported pipeline command type %q", commandType)
	}
	return result, nil
}

func (m *Memory) AcquireLock(ctx context.Context, key string, token string, ttl time.Duration) (bool, error) {
	var acquired bool
	err := m.withState(ctx, func(now time.Time) error {
		if _, ok := m.live(key, now); ok {
			return nil
		}
		m.items[key] = entry{kind: entryKindString, value: token, expiresAt: expiresAt(now, ttl)}
		acquired = true
		return nil
	})
	return acquired, err
}

func (m *Memory) ReleaseLock(ctx context.Context, key string, token string) (bool, error) {
	var released bool
	err := m.withState(ctx, func(now time.Time) error {
		item, ok := m.live(key, now)
		if !ok {
			return nil
		}
		if item.kind != entryKindString {
			return ErrWrongType
		}
		if item.value != token {
			return nil
		}
		delete(m.items, key)
		released = true
		return nil
	})
	return released, err
}

func (m *Memory) FixedWindowRateLimit(ctx context.Context, key string, limit int64, window time.Duration) (RateLimitResult, error) {
	result := RateLimitResult{Limit: limit}
	err := m.withState(ctx, func(now time.Time) error {
		item, ok := m.live(key, now)
		if ok && item.kind != entryKindString {
			return ErrWrongType
		}
		var count int64
		if ok {
			parsed, err := strconv.ParseInt(item.value, 10, 64)
			if err != nil {
				return err
			}
			count = parsed
		} else {
			item = entry{kind: entryKindString, expiresAt: now.Add(window)}
		}
		count++
		item.value = strconv.FormatInt(count, 10)
		if item.expiresAt.IsZero() {
			item.expiresAt = now.Add(window)
		}
		m.items[key] = item
		remaining := limit - count
		if remaining < 0 {
			remaining = 0
		}
		resetAfter := item.expiresAt.Sub(now)
		result.Allowed = count <= limit
		result.Remaining = remaining
		result.ResetAfter = resetAfter
		result.Count = count
		return nil
	})
	return result, err
}

func (m *Memory) hsetLocked(now time.Time, key string, values map[string]string) (int64, error) {
	item, ok := m.live(key, now)
	if !ok {
		item = entry{kind: entryKindHash, hash: map[string]string{}}
	} else if item.kind != entryKindHash {
		return 0, ErrWrongType
	}
	if item.hash == nil {
		item.hash = map[string]string{}
	}
	var added int64
	for field, value := range values {
		if _, exists := item.hash[field]; !exists {
			added++
		}
		item.hash[field] = value
	}
	m.items[key] = item
	return added, nil
}

func (m *Memory) rpushLocked(now time.Time, key string, values ...string) (int64, error) {
	item, ok := m.live(key, now)
	if !ok {
		item = entry{kind: entryKindList}
	} else if item.kind != entryKindList {
		return 0, ErrWrongType
	}
	item.list = append(item.list, values...)
	m.items[key] = item
	return int64(len(item.list)), nil
}

func (m *Memory) addLocked(now time.Time, key string, delta int64) (int64, error) {
	item, ok := m.live(key, now)
	if ok {
		if item.kind != entryKindString {
			return 0, ErrWrongType
		}
		parsed, err := strconv.ParseInt(item.value, 10, 64)
		if err != nil {
			return 0, err
		}
		parsed += delta
		item.value = strconv.FormatInt(parsed, 10)
		m.items[key] = item
		return parsed, nil
	}
	m.items[key] = entry{kind: entryKindString, value: strconv.FormatInt(delta, 10)}
	return delta, nil
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

func cloneMap(values map[string]string) map[string]string {
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func listRange(values []string, start int64, stop int64) []string {
	length := int64(len(values))
	if length == 0 {
		return []string{}
	}
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}
	if start < 0 {
		start = 0
	}
	if stop >= length {
		stop = length - 1
	}
	if start >= length || stop < 0 || start > stop {
		return []string{}
	}
	out := make([]string, stop-start+1)
	copy(out, values[start:stop+1])
	return out
}
