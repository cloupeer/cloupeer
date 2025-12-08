package mqtt

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type HandlerFunc func(ctx context.Context, payload []byte) error

type TypedHandlerFunc[T any, P interface {
	*T
	proto.Message
}] func(ctx context.Context, msg P) error

func ProtoAdapter[T any, P interface {
	*T
	proto.Message
}](handler TypedHandlerFunc[T, P]) HandlerFunc {
	return func(ctx context.Context, payload []byte) error {
		var msg P = new(T)

		unmarshaler := protojson.UnmarshalOptions{DiscardUnknown: true}
		if err := unmarshaler.Unmarshal(payload, msg); err != nil {
			return fmt.Errorf("proto unmarshal failed: %w", err)
		}

		return handler(ctx, msg)
	}
}
