package model

import "time"

// CommandType defines the type of command.
type CommandType string

const (
	CommandTypeOTA    CommandType = "OTA"
	CommandTypeReboot CommandType = "Reboot"
)

// CommandStatus defines the execution status of a command.
type CommandStatus string

const (
	CommandStatusPending   CommandStatus = "Pending"
	CommandStatusSent      CommandStatus = "Sent"
	CommandStatusReceived  CommandStatus = "Received"
	CommandStatusRunning   CommandStatus = "Running"
	CommandStatusSucceeded CommandStatus = "Succeeded"
	CommandStatusFailed    CommandStatus = "Failed"
)

// Command represents an instruction sent to a vehicle.
type Command struct {
	// ID is the unique trace ID (corresponds to K8s CRD Name).
	ID string

	// VehicleID is the target vehicle.
	VehicleID string

	// Type is the command type (OTA, Reboot).
	Type CommandType

	// Parameters contains specific arguments for the command.
	Parameters map[string]string

	// Status represents the current lifecycle phase.
	Status CommandStatus

	// CreatedAt is when the command was issued.
	CreatedAt time.Time
}
