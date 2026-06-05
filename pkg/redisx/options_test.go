package redisx

import (
	"context"
	"testing"
)

func TestOptionsValidateUsesConfigContract(t *testing.T) {
	err := Options{Config: Config{}}.Validate()
	if err == nil {
		t.Fatal("expected missing config name to fail validation")
	}
	if !IsKind(err, ErrorKindValidation) {
		t.Fatalf("expected validation error, got %T %[1]v", err)
	}
}

func TestNewWithOptionsUsesDefaultInMemoryProvider(t *testing.T) {
	client, err := NewWithOptions(context.Background(), Options{Config: Config{Name: "redisx-options"}})
	if err != nil {
		t.Fatalf("new with options: %v", err)
	}
	if err := client.Set(context.Background(), "key", "value", 0); err != nil {
		t.Fatalf("set via default provider: %v", err)
	}
	value, err := client.Get(context.Background(), "key")
	if err != nil {
		t.Fatalf("get via default provider: %v", err)
	}
	if value != "value" {
		t.Fatalf("value = %q, want value", value)
	}
}
