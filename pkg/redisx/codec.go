package redisx

import "encoding/json"

// Codec marshals typed cache values to Redis string values.
type Codec[T any] interface {
	Marshal(T) (string, error)
	Unmarshal(string) (T, error)
}

// JSONCodec encodes typed cache values as JSON strings.
type JSONCodec[T any] struct{}

func (JSONCodec[T]) Marshal(value T) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func (JSONCodec[T]) Unmarshal(raw string) (T, error) {
	var value T
	err := json.Unmarshal([]byte(raw), &value)
	return value, err
}
