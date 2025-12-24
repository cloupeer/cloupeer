package notifier

import (
	"context"
	"encoding/json"

	pb "github.com/autopeer-io/autopeer/api/proto/v1"
	"github.com/autopeer-io/autopeer/internal/bridge/core/model"
	"github.com/autopeer-io/autopeer/internal/pkg/mqtt/paths"
	pkgmqtt "github.com/autopeer-io/autopeer/pkg/mqtt"
	"github.com/autopeer-io/autopeer/pkg/mqtt/topic"
)

type MQTTNotifier struct {
	client pkgmqtt.Client
	topics *topic.Builder
}

func NewMQTTNotifier(client pkgmqtt.Client, builder *topic.Builder) (*MQTTNotifier, error) {
	return &MQTTNotifier{
		client: client,
		topics: builder,
	}, nil
}

func (n *MQTTNotifier) Notify(ctx context.Context, cmd *model.Command) error {

	agentCmd := &pb.AgentCommand{
		CommandName: cmd.ID,
		CommandType: string(cmd.Type),
		Parameters:  cmd.Parameters,
		Timestamp:   cmd.CreatedAt.Unix(),
	}

	payload, err := json.Marshal(agentCmd)
	if err != nil {
		return err
	}

	qos := 1
	retain := true
	t := n.topics.Build(paths.Command, cmd.VehicleID)

	return n.client.Publish(ctx, t, qos, retain, payload)
}
