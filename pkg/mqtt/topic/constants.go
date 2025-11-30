package topic

// Standard MQTT wildcard definitions.
const (
	// Wildcard is the single-level wildcard "+".
	// It matches exactly one topic level.
	// Example: "sensors/+/temperature" matches "sensors/room1/temperature".
	Wildcard = "+"

	// MultiWildcard is the multi-level wildcard "#".
	// It matches the current level and all subsequent levels.
	// It must be the last character in the topic filter.
	// Example: "sensors/tesla/#" matches "sensors/tesla/engine/status".
	MultiWildcard = "#"
)
