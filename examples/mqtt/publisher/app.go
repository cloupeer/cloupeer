package main

import (
	"context"
	"crypto/tls"
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

type Publisher struct {
	cfg    Config
	cm     *autopaho.ConnectionManager
	cancel context.CancelFunc
}

func NewPublisher(cfg Config) *Publisher {
	return &Publisher{cfg: cfg}
}

func (p *Publisher) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	defer cancel()

	serverUrl, err := url.Parse(p.cfg.Broker)
	if err != nil {
		return fmt.Errorf("failed to parse broker URL: %w", err)
	}

	cfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{serverUrl},
		TlsCfg:                        &tls.Config{InsecureSkipVerify: true},
		KeepAlive:                     60,
		CleanStartOnInitialConnection: false,
		SessionExpiryInterval:         60,
		ReconnectBackoff:              autopaho.NewConstantBackoff(5 * time.Second),
		// Set callbacks to methods on our Publisher struct
		OnConnectionUp: p.OnConnectionUp,
		OnConnectError: p.OnConnectError,
		ClientConfig: paho.ClientConfig{
			ClientID:           p.cfg.ClientID,
			Session:            state.NewInMemory(),
			OnClientError:      func(err error) { log.Printf("client error: %s\n", err) },
			OnServerDisconnect: p.OnServerDisconnect,
		},
	}

	if p.cfg.Debug {
		cfg.Debug = logger{prefix: "autoPaho"}
		cfg.PahoDebug = logger{prefix: "paho"}
	}

	// Connect to the server
	log.Println("Attempting to connect to broker...")
	p.cm, err = autopaho.NewConnection(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create connection manager: %w", err)
	}

	// Start the publisher loop in a separate goroutine
	go p.startPublishLoop(ctx)

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
	_ = p.cm.Disconnect(shutdownCtx)

	log.Println("Shutdown complete.")
	return nil
}

func (p *Publisher) startPublishLoop(ctx context.Context) {
	var count uint64
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Wait for the connection to be up before publishing
			err := p.cm.AwaitConnection(ctx)
			if err != nil { // Should only happen when context is cancelled
				log.Printf("Publisher exiting (AwaitConnection: %s)\n", err)
				return
			}

			count++
			msg, err := json.Marshal(struct {
				Count uint64
			}{Count: count})
			if err != nil {
				log.Printf("Failed to marshal JSON: %v", err)
				continue
			}

			// Publish the message
			pr, err := p.cm.Publish(ctx, &paho.Publish{
				QoS:     byte(p.cfg.QoS),
				Topic:   p.cfg.Topic,
				Payload: msg,
				Retain:  false,
			})
			if err != nil {
				log.Printf("Error publishing message %s: %s\n", msg, err)
			} else if pr.ReasonCode != 0 && pr.ReasonCode != 16 { // 16 = Server received message but there are no subscribers
				log.Printf("Reason code %d received for message %s\n", pr.ReasonCode, msg)
			} else {
				log.Printf("Sent message: %s\n", msg)
			}

		case <-ctx.Done():
			log.Println("Publisher loop stopping.")
			return
		}
	}
}

// --- Callbacks ---

func (p *Publisher) OnConnectionUp(cm *autopaho.ConnectionManager, c *paho.Connack) {
	log.Println("MQTT connection up")
	// Publisher doesn't need to subscribe, but you could log state here
}

func (p *Publisher) OnConnectError(err error) {
	log.Printf("Error whilst attempting connection: %s\n", err)
}

func (p *Publisher) OnServerDisconnect(d *paho.Disconnect) {
	log.Printf("Server requested disconnect: %d - %s\n", d.ReasonCode, d.Properties.ReasonString)
	if d.ReasonCode == 0x8E { // 142 - Session taken over
		log.Println("Session taken over, initiating shutdown.")
		p.cancel() // Trigger graceful shutdown
	}
}
