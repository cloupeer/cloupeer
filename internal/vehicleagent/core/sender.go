package core

import (
	"context"

	"google.golang.org/protobuf/proto"
)

type Sender interface {
	Send(ctx context.Context, event EventType, payload []byte) error
	SendProto(ctx context.Context, event EventType, msg proto.Message) error
}
