package core

import (
	"context"

	"github.com/autopeer-io/autopeer/internal/pkg/mqtt/adapter"
)

type Module interface {
	Name() string

	Setup(ctx context.Context, hal HAL, sender Sender) error

	Routes() map[EventType]adapter.HandlerFunc
}
