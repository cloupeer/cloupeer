package core

import (
	"context"
)

type Module interface {
	Name() string

	Setup(ctx context.Context, hal HAL, sender Sender) error

	Routes() map[EventType]HandlerFunc
}
