package log

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

// toFields converts a flexible list of arguments to a slice of zap.Field.
// It intelligently handles different argument patterns:
// 1. A single `error` argument becomes `zap.Error(err)`.
// 2. A single `zap.Field` argument is passed through as-is.
// 3. A pair of arguments (string, any) becomes a typed `zap.Field`.
// 4. Any other argument is treated as a key-value pair, with fallbacks for non-string keys.
func toFields(args ...any) []zap.Field {
	if len(args) == 0 {
		return nil
	}

	fields := make([]zap.Field, 0, len(args)/2+1)

	for i := 0; i < len(args); {
		// Case 1: The argument is already a zap.Field.
		if f, ok := args[i].(zap.Field); ok {
			fields = append(fields, f)
			i++
			continue
		}

		// Case 2: The argument is an error.
		if err, ok := args[i].(error); ok {
			fields = append(fields, zap.Error(err))
			i++
			continue
		}

		// Case 3: We are at the last argument, which is an unpaired value.
		if i == len(args)-1 {
			fields = append(fields, zap.Any(fmt.Sprintf("arg#%d", i), args[i]))
			break
		}

		// Case 4: Treat as a key-value pair.
		key, val := args[i], args[i+1]
		i += 2

		// Key 必须是 string，否则 fallback
		keyStr, ok := key.(string)
		if !ok {
			// If the key isn't a string, log it as a structured field to avoid losing data.
			fields = append(fields, zap.Any(fmt.Sprintf("invalid_key_%d", i/2), map[string]any{
				"key":   key,
				"value": val,
			}))
			continue
		}

		switch v := val.(type) {
		case string:
			fields = append(fields, zap.String(keyStr, v))
		case bool:
			fields = append(fields, zap.Bool(keyStr, v))
		case int:
			fields = append(fields, zap.Int(keyStr, v))
		case int8:
			fields = append(fields, zap.Int8(keyStr, v))
		case int16:
			fields = append(fields, zap.Int16(keyStr, v))
		case int32:
			fields = append(fields, zap.Int32(keyStr, v))
		case int64:
			fields = append(fields, zap.Int64(keyStr, v))
		case uint:
			fields = append(fields, zap.Uint(keyStr, v))
		case uint8:
			fields = append(fields, zap.Uint8(keyStr, v))
		case uint16:
			fields = append(fields, zap.Uint16(keyStr, v))
		case uint32:
			fields = append(fields, zap.Uint32(keyStr, v))
		case uint64:
			fields = append(fields, zap.Uint64(keyStr, v))
		case float32:
			fields = append(fields, zap.Float32(keyStr, v))
		case float64:
			fields = append(fields, zap.Float64(keyStr, v))
		case time.Duration:
			fields = append(fields, zap.Duration(keyStr, v))
		case time.Time:
			fields = append(fields, zap.Time(keyStr, v))
		case error:
			fields = append(fields, zap.NamedError(keyStr, v))
		case fmt.Stringer:
			fields = append(fields, zap.String(keyStr, v.String()))
		case []byte:
			fields = append(fields, zap.Binary(keyStr, v))
		default:
			fields = append(fields, zap.Any(keyStr, v))
		}
	}

	return fields
}
