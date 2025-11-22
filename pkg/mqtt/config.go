package mqtt

import (
	"errors"
	"net/url"
	"time"
)

// ClientConfig holds the configuration for creating a new MQTT Client.
type ClientConfig struct {
	BrokerURL string
	ClientID  string
	Username  string
	Password  string

	// KeepAlive in seconds. Default is 60.
	KeepAlive uint16

	// SessionExpiry in seconds. Default is 60.
	SessionExpiry uint32

	// ConnectTimeout for the initial connection. Default is 5s.
	ConnectTimeout time.Duration

	// CleanStart indicates whether to start a clean session.
	// For Cloupeer agents, this is usually false to receive missed messages.
	CleanStart bool

	// InsecureSkipVerify disables TLS certificate verification.
	// MUST be true for Cloupeer's self-signed certs environment.
	InsecureSkipVerify bool
}

// setDefaultConfig applies safe default values to the configuration.
func setDefaultConfig(cfg *ClientConfig) {
	if cfg.KeepAlive == 0 {
		cfg.KeepAlive = 60
	}

	if cfg.SessionExpiry == 0 {
		cfg.SessionExpiry = 60
	}

	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 5 * time.Second
	}
}

// Validate checks if the configuration is valid.
func (c *ClientConfig) Validate() error {
	if c.BrokerURL == "" {
		return errors.New("broker url is required")
	}
	if _, err := url.Parse(c.BrokerURL); err != nil {
		return err
	}
	return nil
}
