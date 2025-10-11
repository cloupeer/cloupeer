package log

import (
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestToFields(t *testing.T) {
	now := time.Now()
	err := errors.New("boom")

	tests := []struct {
		name  string
		input []any
	}{
		{"empty input", []any{}},
		{"string-int-bool", []any{"a", "x", "b", 123, "c", true}},
		{"time type", []any{"t", now}},
		{"float type", []any{"pi", 3.14}},
		{"bytes", []any{"data", []byte("xyz")}},
		{"error only", []any{err}},
		{"multiple errors", []any{err, errors.New("again")}},
		{"mixed field types", []any{"msg", "ok", zap.String("x", "y"), "num", 42}},
		{"odd number of args", []any{"key1", "val1", "key2"}},
		{"non-string key", []any{123, "value", true, 99}},
		{"nil values", []any{"a", nil, "b", (*int)(nil)}},
		{"map value", []any{"a", map[string]string{"xyz": "123"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := toFields(tt.input...)

			// 检查不 panic
			if fields == nil && len(tt.input) > 0 {
				t.Errorf("nil fields for non-empty input: %v", tt.input)
			}

			// 检查每个 field key/value 可打印
			for _, f := range fields {
				if f.Key == "" {
					t.Errorf("field has empty key: %+v", f)
				}
			}
		})
	}
}
