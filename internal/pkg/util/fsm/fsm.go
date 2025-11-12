package fsm

import (
	"context"

	"github.com/looplab/fsm"
)

func WrapEvent(fn func(ctx context.Context, event *fsm.Event) error) fsm.Callback {
	return func(ctx context.Context, event *fsm.Event) {
		if err := fn(ctx, event); err != nil {
			event.Err = err
		}
	}
}
