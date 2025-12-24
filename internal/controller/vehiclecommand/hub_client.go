package vehiclecommand

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pb "github.com/autopeer-io/autopeer/api/proto/v1"
	"github.com/autopeer-io/autopeer/internal/pkg/metrics"
	grpcmiddleware "github.com/autopeer-io/autopeer/internal/pkg/middleware/grpc"
)

// HubClient defines the interface for interacting with the Autopeer Hub.
// Note: It now implicitly satisfies manager.Runnable because of Start(context.Context) error.
type HubClient interface {
	Start(ctx context.Context) error
	SendCommand(ctx context.Context, req *pb.SendCommandRequest) (*pb.SendCommandResponse, error)
}

// GrpcHubClient is the real implementation using gRPC.
type GrpcHubClient struct {
	client pb.HubServiceClient
	conn   *grpc.ClientConn
}

var _ HubClient = (*GrpcHubClient)(nil)

// NewGrpcHubClient creates a new GrpcHubClient.
func NewGrpcHubClient(addr string) *GrpcHubClient {
	// Establish connection
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(grpcmiddleware.UnaryTimeoutInterceptor),
		// Add KeepAlive params here for production robustness
	)
	if err != nil {
		panic(fmt.Sprintf("FATAL: failed to initialize gRPC client for hub addr '%s': %v", addr, err))
	}

	return &GrpcHubClient{
		client: pb.NewHubServiceClient(conn),
		conn:   conn,
	}
}

func (c *GrpcHubClient) SendCommand(ctx context.Context, req *pb.SendCommandRequest) (*pb.SendCommandResponse, error) {
	return c.client.SendCommand(ctx, req)
}

// Start is manages the lifecycle of the gRPC connection.
func (c *GrpcHubClient) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("HubClient lifecycle manager started")

	go c.monitorConnection(ctx)

	// BLOCK until the context is closed
	<-ctx.Done()

	// Cleanup (Graceful Shutdown)
	logger.Info("HubClient shutting down, closing gRPC connection...")
	return c.conn.Close()
}

func (c *GrpcHubClient) monitorConnection(ctx context.Context) {
	logger := log.FromContext(ctx).WithName("grpc-monitor")

	// 初始状态检查
	lastState := c.conn.GetState()
	c.updateMetric(lastState)

	for {
		// WaitForStateChange 会阻塞，直到状态发生变化
		// 如果 ctx 被取消，它也会返回 false
		if !c.conn.WaitForStateChange(ctx, lastState) {
			// 上下文已取消，退出监控
			return
		}

		newState := c.conn.GetState()
		logger.Info("Hub connection state changed", "from", lastState, "to", newState)

		c.updateMetric(newState)
		lastState = newState
	}
}

func (c *GrpcHubClient) updateMetric(state connectivity.State) {
	if state == connectivity.Ready {
		metrics.HubConnectivityStatus.Set(1)
	} else {
		// Idle, Connecting, TransientFailure, Shutdown 都视为 0
		metrics.HubConnectivityStatus.Set(0)
	}
}
