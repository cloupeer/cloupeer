package paths

// Topic segments for the Autopeer IoV protocol.
// These constants define the routing topology contract between Cloud (Hub) and Edge (Agent).

// Downstream: Cloud -> Edge (Directives & Responses)
const (
	// Command is the topic segment for downstream control directives.
	// Pattern: {root}/command/{vehicleID}
	Command = "command"

	// OTAResponse is the topic segment for delivering firmware update artifacts.
	// Payload: { "requestID": "...", "downloadURL": "..." }
	// Pattern: {root}/ota/response/{vehicleID}
	OTAResponse = "ota/response"
)

// Upstream: Edge -> Cloud (Requests & Status Reports)
const (
	// Register is the topic segment for vehicle registration.
	// Pattern: {root}/register/{vehicleID}
	Register = "register"

	// Online is the topic segment for reporting vehicle online/offline status.
	// Payload: { "online": true/false, "timestamp": ... }
	// Pattern: {root}/online/{vehicleID}
	Online = "online"

	// CommandAck is the topic segment for command execution status updates.
	// Pattern: {root}/command/ack/{vehicleID}
	CommandAck = "command/ack"

	// OTARequest is the topic segment for requesting firmware update information.
	// Payload: { "currentVersion": "v1.0", "requestID": "..." }
	// Pattern: {root}/ota/request/{vehicleID}
	OTARequest = "ota/request"

	// OTAProgress is the topic segment for reporting installation progress.
	// Payload: { "percentage": 50, "status": "installing", "message": "..." }
	// Pattern: {root}/ota/progress/{vehicleID}
	OTAProgress = "ota/progress"
)
