package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloupeer.io/cloupeer/internal/cloudhub/core/model"
	pkgmqtt "cloupeer.io/cloupeer/pkg/mqtt"
	"cloupeer.io/cloupeer/pkg/options"
)

type MQTTNotifier struct {
	client pkgmqtt.Client
}

func NewMQTTNotifier(opts *options.MqttOptions) (*MQTTNotifier, error) {
	// Create a dedicated client for outgoing messages (Egress)
	// This separates ingress and egress connections, which is good for reliability.
	cfg := &pkgmqtt.ClientConfig{
		BrokerURL:          opts.Broker,
		ClientID:           opts.ClientID + "-notifier", // Distinct ClientID
		Username:           opts.Username,
		Password:           opts.Password,
		CleanStart:         true,
		KeepAlive:          60,
		ConnectTimeout:     5 * time.Second,
		InsecureSkipVerify: true,
	}

	client, err := pkgmqtt.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	// Start the client immediately
	if err := client.Start(context.Background()); err != nil {
		return nil, err
	}
	// Note: We might want to AwaitConnection here or lazily wait on first Publish

	return &MQTTNotifier{client: client}, nil
}

func (n *MQTTNotifier) Notify(ctx context.Context, cmd *model.Command) error {
	// Topic: devices/{vehicleID}/command
	topic := fmt.Sprintf("devices/%s/command", cmd.VehicleID)

	payload, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	return n.client.Publish(ctx, topic, 1, false, payload)
}
