package edgeagent

import (
	"net/http"
	"time"

	"cloupeer.io/cloupeer/pkg/log"
)

type Config struct {
	DeviceID          string
	GatewayURL        string
	HeartbeatInterval time.Duration
	VersionFile       string
}

func (cfg *Config) NewAgent() (*Agent, error) {
	initialVersion, err := readVersionFromFile(cfg.VersionFile)
	if err != nil {
		log.Warn("Failed to read initial version, defaulting to v1.0.0", "error", err)
		initialVersion = "v1.0.0"
	}

	return &Agent{
		deviceID: cfg.DeviceID,
		heartbeat: &heartbeat{
			url:      cfg.GatewayURL,
			interval: cfg.HeartbeatInterval,
		},
		state:       &state{firmwareVersion: initialVersion},
		versionFile: cfg.VersionFile,
		client:      &http.Client{Timeout: 10 * time.Second},
	}, nil
}
