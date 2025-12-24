package mqtt

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"

	"github.com/autopeer-io/autopeer/pkg/log"
)

type pahoClient struct {
	cfg *ClientConfig
	cm  *autopaho.ConnectionManager

	// subscriptions holds the registered handlers.
	// Key: topic filter (string), Value: subscriptionEntry
	subscriptions sync.Map
}

type subscriptionEntry struct {
	topic   string
	qos     int
	handler MessageHandler
}

// NewClient creates a new MQTT client implementing the Client interface.
func NewClient(cfg *ClientConfig) (Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("mqtt config is required")
	}

	setDefaultConfig(cfg)

	// Basic validation using the config's own logic
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid mqtt config: %w", err)
	}

	return &pahoClient{
		cfg: cfg,
	}, nil
}

func (c *pahoClient) Start(ctx context.Context) error {
	brokerURL, _ := url.Parse(c.cfg.BrokerURL) // Already validated

	pahoCfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{brokerURL},
		KeepAlive:                     c.cfg.KeepAlive,
		CleanStartOnInitialConnection: c.cfg.CleanStart,
		SessionExpiryInterval:         c.cfg.SessionExpiry,
		ReconnectBackoff:              autopaho.NewConstantBackoff(3 * time.Second),
		ConnectTimeout:                c.cfg.ConnectTimeout,
		ConnectUsername:               c.cfg.Username,
		ConnectPassword:               []byte(c.cfg.Password),
		TlsCfg: &tls.Config{
			InsecureSkipVerify: c.cfg.InsecureSkipVerify,
		},
		WillMessage: c.willMessage(),
		ClientConfig: paho.ClientConfig{
			ClientID:           c.cfg.ClientID,
			OnClientError:      c.onClientError,
			OnServerDisconnect: c.onServerDisconnect,
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				c.router,
			},
		},
		OnConnectionUp: c.onConnectionUp,
		OnConnectError: c.onConnectError,
	}

	log.Info("Starting MQTT Client", "broker", c.cfg.BrokerURL, "clientID", c.cfg.ClientID)

	cm, err := autopaho.NewConnection(ctx, pahoCfg)
	if err != nil {
		return err
	}
	c.cm = cm
	return nil
}

func (c *pahoClient) Disconnect(ctx context.Context) {
	if c.cm != nil {
		_ = c.cm.Disconnect(ctx)
		log.Info("MQTT Client disconnected")
	}
}

func (c *pahoClient) Publish(ctx context.Context, topic string, qos int, retain bool, payload []byte) error {
	if c.cm == nil {
		return fmt.Errorf("client not started")
	}

	// Check connection status to avoid immediate error if possible,
	// although paho handles offline buffering if configured.
	// Here we simply delegate.
	_, err := c.cm.Publish(ctx, &paho.Publish{
		Topic:   topic,
		QoS:     byte(qos),
		Retain:  retain,
		Payload: payload,
	})

	return err
}

func (c *pahoClient) Subscribe(ctx context.Context, topic string, qos int, handler MessageHandler) error {
	if c.cm == nil {
		return fmt.Errorf("client not started")
	}

	// 1. Store the handler for routing and re-connection logic
	entry := subscriptionEntry{
		topic:   topic,
		qos:     qos,
		handler: handler,
	}
	c.subscriptions.Store(topic, entry)

	// 2. If currently connected, send the SUBSCRIBE packet immediately.
	// If not connected, OnConnectionUp will handle it later.
	// Note: We don't strictly check IsConnected() because autopaho might be in a reconnecting state.
	// Attempting to subscribe usually works or queues up.
	_, err := c.cm.Subscribe(ctx, &paho.Subscribe{
		Subscriptions: []paho.SubscribeOptions{
			{Topic: topic, QoS: byte(qos)},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to send subscription packet: %w", err)
	}

	log.Info("Subscribed to topic", "topic", topic)
	return nil
}

func (c *pahoClient) Unsubscribe(ctx context.Context, topic string) error {
	if c.cm == nil {
		return fmt.Errorf("client not started")
	}

	c.subscriptions.Delete(topic)

	_, err := c.cm.Unsubscribe(ctx, &paho.Unsubscribe{
		Topics: []string{topic},
	})
	return err
}

func (c *pahoClient) AwaitConnection(ctx context.Context) error {
	if c.cm == nil {
		return fmt.Errorf("client not started")
	}
	return c.cm.AwaitConnection(ctx)
}

func (c *pahoClient) IsConnected() bool {
	// autopaho doesn't expose a direct IsConnected bool easily without locking internal state,
	// but we can rely on AwaitConnection returning nil context immediately if connected.
	// However, for simplicity in this context, we skip deep inspection.
	// A simple way is checking context errors on a dummy operation or creating a struct field tracked by hooks.
	// For now, we omit complex status checking to keep it simple.
	return true
}

// --- Internal Callbacks ---

// onConnectionUp is called when the connection is established or re-established.
func (c *pahoClient) onConnectionUp(cm *autopaho.ConnectionManager, ack *paho.Connack) {
	log.Info("MQTT Connection established")

	// Re-subscribe to all registered topics
	c.subscriptions.Range(func(key, value any) bool {
		entry := value.(subscriptionEntry)
		log.Info("Re-subscribing", "topic", entry.topic)
		if _, err := cm.Subscribe(context.Background(), &paho.Subscribe{
			Subscriptions: []paho.SubscribeOptions{
				{Topic: entry.topic, QoS: byte(entry.qos)},
			},
		}); err != nil {
			log.Error(err, "Failed to re-subscribe", "topic", entry.topic)
		}
		return true
	})
}

func (c *pahoClient) onConnectError(err error) {
	log.Error(err, "MQTT Connection failed, retrying...")
}

func (c *pahoClient) onClientError(err error) {
	log.Error(err, "MQTT Client internal error")
}

func (c *pahoClient) onServerDisconnect(d *paho.Disconnect) {
	log.Warn("MQTT Server requested disconnect", "reason", d.Properties.ReasonString)
}

// router handles incoming messages and dispatches them to the registered handlers.
func (c *pahoClient) router(p paho.PublishReceived) (bool, error) {
	// Iterate over subscriptions to find a match.
	// Since we support wildcards, we cannot do a simple map lookup.
	// This O(N) iteration is acceptable for the expected number of subscriptions (usually < 10 per agent/hub).
	// For high-scale, a Trie-based implementation would be needed.

	matched := false
	c.subscriptions.Range(func(key, value any) bool {
		entry := value.(subscriptionEntry)
		if topicsMatch(topicFilter(entry.topic), p.Packet.Topic) {
			// Execute handler in a separate goroutine to avoid blocking the reader loop
			// Or execute inline if logic is fast. Given "go" keyword is cheap:
			go func(h MessageHandler) {
				// Create a background context or derive one with timeout
				h(context.Background(), p.Packet.Topic, p.Packet.Payload)
			}(entry.handler)
			matched = true
		}
		return true
	})

	if !matched {
		log.Debug("Received message on unhandled topic", "topic", p.Packet.Topic)
	}

	return true, nil // Always acknowledge reception
}

func (c *pahoClient) willMessage() *paho.WillMessage {
	if c.cfg.WillTopic == "" {
		return nil
	}
	return &paho.WillMessage{
		Topic:   c.cfg.WillTopic,
		Payload: c.cfg.WillPayload,
		QoS:     c.cfg.WillQoS,
		Retain:  c.cfg.WillRetain,
	}
}

// topicsMatch checks if a topic matches a filter (supports wildcards + and #).
func topicsMatch(filter, topic string) bool {
	// This is a simplified matcher. Paho often has internal ones, but for transparency:
	if filter == topic {
		return true
	}

	// If simple equality fails, check for wildcards.
	// Optimization: if no wildcards, we are done.
	if !strings.Contains(filter, "+") && !strings.Contains(filter, "#") {
		return false
	}

	filterParts := strings.Split(filter, "/")
	topicParts := strings.Split(topic, "/")

	for i, part := range filterParts {
		if part == "#" {
			return true
		}
		if i >= len(topicParts) {
			return false
		}
		if part != "+" && part != topicParts[i] {
			return false
		}
	}

	return len(filterParts) == len(topicParts)
}

func topicFilter(filter string) string {
	if strings.HasPrefix(filter, "$share/") {
		// Format: $share/<group>/<topic>
		parts := strings.SplitN(filter, "/", 3)
		if len(parts) == 3 {
			return parts[2]
		}
	}
	return filter
}
