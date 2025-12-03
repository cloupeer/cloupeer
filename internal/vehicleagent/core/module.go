package core

import (
	"context"
)

type Module interface {
	Name() string

	Setup(ctx context.Context, sender Sender) error

	Routes() map[EventType]HandlerFunc
}

var modules []Module

func Register(m Module, opts ...string) {
	modules = append(modules, m)
}

func GetModules() []Module {
	return modules
}

func init() {
	modules = make([]Module, 0)
}
