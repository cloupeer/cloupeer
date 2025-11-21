package mqtt

import (
	"context"
)

// MessageHandler defines the callback function for processing received MQTT messages.
type MessageHandler func(ctx context.Context, topic string, payload []byte)

// Client defines the interface for a generic MQTT client.
// It abstracts the underlying paho implementation details.
type Client interface {
	// Start initiates the connection to the broker.
	// It is non-blocking and returns immediately. Use AwaitConnection to wait.
	Start(ctx context.Context) error

	// Disconnect cleanly closes the connection.
	Disconnect(ctx context.Context)

	// Publish sends a message to the specified topic.
	Publish(ctx context.Context, topic string, qos int, retain bool, payload []byte) error

	// Subscribe registers a handler for a specific topic filter.
	// It handles the underlying MQTT subscription packet sending.
	// If the connection is lost and restored, this client will automatically re-subscribe.
	Subscribe(ctx context.Context, topic string, qos int, handler MessageHandler) error

	// Unsubscribe removes the handler and sends an UNSUBSCRIBE packet.
	Unsubscribe(ctx context.Context, topic string) error

	// AwaitConnection blocks until the client is connected to the broker.
	AwaitConnection(ctx context.Context) error

	// IsConnected returns true if the client is currently connected.
	IsConnected() bool
}
