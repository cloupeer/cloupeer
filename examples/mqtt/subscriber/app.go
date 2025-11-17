package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/eclipse/paho.golang/paho/session/state"
)

type Config struct {
	Debug    bool
	Broker   string
	Topic    string
	ClientID string
	QoS      int
}

type Subscriber struct {
	cfg    Config
	cm     *autopaho.ConnectionManager
	cancel context.CancelFunc
}

func NewSubscriber(cfg Config) *Subscriber {
	return &Subscriber{cfg: cfg}
}

func (s *Subscriber) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	defer cancel()

	serverUrl, err := url.Parse(s.cfg.Broker)
	if err != nil {
		return fmt.Errorf("failed to parse broker URL: %w", err)
	}

	cfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{serverUrl},
		KeepAlive:                     60,
		CleanStartOnInitialConnection: false,
		SessionExpiryInterval:         60,
		ReconnectBackoff:              autopaho.NewConstantBackoff(5 * time.Second),
		// Set callbacks to methods on our Subscriber struct
		OnConnectionUp: s.OnConnectionUp,
		OnConnectError: s.OnConnectError,
		ClientConfig: paho.ClientConfig{
			ClientID: s.cfg.ClientID,
			Session:  state.NewInMemory(),
			// Set the OnPublishReceived callback
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				s.OnPublishReceived,
			},
			OnClientError:      func(err error) { log.Printf("client error: %s\n", err) },
			OnServerDisconnect: s.OnServerDisconnect,
		},
	}

	if s.cfg.Debug {
		cfg.Debug = logger{prefix: "autoPaho"}
		cfg.PahoDebug = logger{prefix: "paho"}
	}

	// Connect to the server
	log.Println("Attempting to connect to broker...")
	s.cm, err = autopaho.NewConnection(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create connection manager: %w", err)
	}

	// Wait for OS signal or context cancellation
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	signal.Notify(sig, syscall.SIGTERM)

	select {
	case <-sig:
		log.Println("OS signal caught, initiating shutdown...")
		cancel() // Trigger context cancellation
	case <-ctx.Done():
		log.Println("Context cancelled, initiating shutdown...")
	}

	// Perform graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	log.Println("Disconnecting from broker...")
	_ = s.cm.Disconnect(shutdownCtx)

	log.Println("Shutdown complete.")
	return nil
}

func (s *Subscriber) OnConnectionUp(cm *autopaho.ConnectionManager, c *paho.Connack) {
	log.Println("MQTT connection up")
	if _, err := cm.Subscribe(context.Background(), &paho.Subscribe{
		Subscriptions: []paho.SubscribeOptions{
			{Topic: s.cfg.Topic, QoS: byte(s.cfg.QoS)},
		},
	}); err != nil {
		log.Printf("failed to subscribe (topic: %s): %s\n", s.cfg.Topic, err)
		return
	}
	log.Printf("Subscribed to topic: %s\n", s.cfg.Topic)
}

func (s *Subscriber) OnConnectError(err error) {
	log.Printf("Error whilst attempting connection: %s\n", err)
}

func (s *Subscriber) OnServerDisconnect(d *paho.Disconnect) {
	log.Printf("Server requested disconnect: %d - %s\n", d.ReasonCode, d.Properties.ReasonString)
	if d.ReasonCode == 0x8E { // 142 - Session taken over
		log.Println("Session taken over, initiating shutdown.")
		s.cancel() // Trigger graceful shutdown
	}
}

func (s *Subscriber) OnPublishReceived(pr paho.PublishReceived) (bool, error) {
	var m struct {
		Count uint64
	}

	if err := json.Unmarshal(pr.Packet.Payload, &m); err != nil {
		log.Printf("Failed to parse message payload (%s): %s\n", pr.Packet.Payload, err)
		return true, nil // Acknowledge message even if parse fails
	}

	log.Printf("Received message (Count: %d) on topic: %s\n", m.Count, pr.Packet.Topic)
	return true, nil
}
