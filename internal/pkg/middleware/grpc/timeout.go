package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

const DefaultRPCTimeout = 10 * time.Second

func UnaryTimeoutInterceptor(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultRPCTimeout)
		defer cancel()
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}
