package vehiclecommand

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pb "cloupeer.io/cloupeer/api/proto/v1"
	grpcmiddleware "cloupeer.io/cloupeer/internal/pkg/middleware/grpc"
)

// HubClient defines the interface for interacting with the Cloupeer Hub.
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

	// BLOCK until the context is closed
	<-ctx.Done()

	// Cleanup (Graceful Shutdown)
	logger.Info("HubClient shutting down, closing gRPC connection...")
	return c.conn.Close()
}
