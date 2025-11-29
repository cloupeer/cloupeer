package topic

import (
	"fmt"
)

// Constants defining the standard topic segments.
// These act as the "Protocol Contract" between Cloud (Hub) and Edge (Agent).
// Changing these values will break compatibility with existing agents.
const (
	// SuffixCommand represents the downstream command topic (Cloud -> Edge).
	// Structure: {root}/command/{vehicleID}
	SuffixCommand = "command"

	// SuffixCommandAck represents the upstream command acknowledgement/status topic (Edge -> Cloud).
	// By placing it under 'command/ack', we maintain logical grouping.
	// Structure: {root}/command/ack/{vehicleID}
	SuffixCommandAck = "command/ack"

	// SuffixFirmwareReq represents the upstream request for firmware URL (Edge -> Cloud).
	// Structure: {root}/firmware/url/req/{vehicleID}
	SuffixFirmwareReq = "firmware/url/req"

	// SuffixFirmwareResp represents the downstream response containing firmware URL (Cloud -> Edge).
	// Structure: {root}/firmware/url/resp/{vehicleID}
	SuffixFirmwareResp = "firmware/url/resp"

	// SuffixRegister represents the upstream registration topic (Edge -> Cloud).
	// Structure: {root}/register/{vehicleID}
	SuffixRegister = "register"
)

// TopicBuilder encapsulates the logic for constructing MQTT topic strings.
// It ensures type safety and consistency across the entire project.
type TopicBuilder struct {
	// root is the base namespace for all topics (e.g., "iov/v1", "cloupeer/prod").
	root string
}

// NewTopicBuilder creates a new instance of TopicBuilder with the specified root namespace.
func NewTopicBuilder(root string) *TopicBuilder {
	return &TopicBuilder{root: root}
}

// -----------------------------------------------------------------------------
// Topic Generation Methods
// -----------------------------------------------------------------------------

// Command returns the topic string for sending commands to a specific vehicle.
// Direction: Cloud -> Edge
func (b *TopicBuilder) Command(vehicleID string) string {
	return b.build(SuffixCommand, vehicleID)
}

// CommandAck returns the topic string for a vehicle to report command status.
// Direction: Edge -> Cloud
func (b *TopicBuilder) CommandAck(vehicleID string) string {
	return b.build(SuffixCommandAck, vehicleID)
}

// CommandAckWildcard returns the wildcard topic used by the Hub to subscribe to ALL acknowledgements.
// Result: {root}/command/ack/+
func (b *TopicBuilder) CommandAckWildcard() string {
	return b.build(SuffixCommandAck, "+")
}

// FirmwareURLReq returns the topic string for a vehicle to request a firmware download URL.
// Direction: Edge -> Cloud
func (b *TopicBuilder) FirmwareURLReq(vehicleID string) string {
	return b.build(SuffixFirmwareReq, vehicleID)
}

// FirmwareURLReqWildcard returns the wildcard topic used by the Hub to subscribe to ALL URL requests.
// Result: {root}/firmware/url/req/+
func (b *TopicBuilder) FirmwareURLReqWildcard() string {
	return b.build(SuffixFirmwareReq, "+")
}

// FirmwareURLResp returns the topic string for the Hub to send the firmware URL back to the vehicle.
// Direction: Cloud -> Edge
func (b *TopicBuilder) FirmwareURLResp(vehicleID string) string {
	return b.build(SuffixFirmwareResp, vehicleID)
}

// Register returns the topic string for a vehicle to register itself.
// Direction: Edge -> Cloud
func (b *TopicBuilder) Register(vehicleID string) string {
	return b.build(SuffixRegister, vehicleID)
}

// RegisterWildcard returns the wildcard topic used by the Hub to subscribe to ALL registrations.
// Result: {root}/register/+
func (b *TopicBuilder) RegisterWildcard() string {
	return b.build(SuffixRegister, "+")
}

// -----------------------------------------------------------------------------
// Helper Methods
// -----------------------------------------------------------------------------

// build is a private helper to construct the final topic string.
// Pattern: {root}/{suffix}/{identifier}
func (b *TopicBuilder) build(suffix, id string) string {
	return fmt.Sprintf("%s/%s/%s", b.root, suffix, id)
}
