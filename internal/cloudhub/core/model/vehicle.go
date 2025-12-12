package model

import "time"

// Vehicle represents the core business entity of a connected vehicle.
// It is decoupled from Kubernetes CRD definitions to maintain clean architecture.
type Vehicle struct {
	// VIN is the 17-character physical identifier.
	// This maps to Spec.VIN in the CRD.
	VIN string

	// ReportedVersion is the version currently reported by the vehicle.
	ReportedVersion string

	// Online indicates if the vehicle is currently connected to CloudHub.
	Online bool

	// LastHeartbeatTime is the timestamp when the vehicle last communicated.
	LastHeartbeatTime time.Time

	// DesiredVersion is the target version we want the vehicle to upgrade to.
	// This comes from the Spec.
	DesiredVersion string

	IsRegister bool
}

// VehicleStatusUpdate represents a partial update to the vehicle's status.
// Used for high-frequency updates (e.g. heartbeat) to avoid fetching the full object.
type VehicleStatusUpdate struct {
	VIN               string
	Online            bool
	LastHeartbeatTime time.Time
}
