package options

import (
	"time"

	"cloupeer.io/cloupeer/pkg/mqtt"
	"github.com/spf13/pflag"
)

var _ IOptions = (*MqttOptions)(nil)

// MqttOptions contains configuration for MQTT client and topics.
type MqttOptions struct {
	Broker   string `json:"broker" mapstructure:"broker"`
	Username string `json:"username" mapstructure:"username"`
	Password string `json:"password" mapstructure:"password"`
	ClientID string `json:"client-id" mapstructure:"client-id"`

	// Client behavior
	KeepAlive      time.Duration `json:"keep-alive" mapstructure:"keep-alive"`
	ConnectTimeout time.Duration `json:"connect-timeout" mapstructure:"connect-timeout"`
	SessionExpiry  uint32        `json:"session-expiry" mapstructure:"session-expiry"`
	CleanStart     bool          `json:"clean-start" mapstructure:"clean-start"`

	// InsecureSkipVerify controls whether a client verifies the server's certificate chain and host name.
	// If true, TLS accepts any certificate presented by the server and any host name in that certificate.
	// In this mode, TLS is susceptible to man-in-the-middle attacks. This should be used only for testing.
	InsecureSkipVerify bool `json:"insecure-skip-verify" mapstructure:"insecure-skip-verify"`

	// Topic Topology definition
	// Using prefixes allows us to construct topics like: {TopicRoot}/{XXX}
	TopicRoot string `json:"topic-root" mapstructure:"topic-root"`
}

// NewMqttOptions creates a new MqttOptions with default values.
func NewMqttOptions() *MqttOptions {
	return &MqttOptions{
		Broker:             "wss://mqtt.cloupeer.io/mqtt",
		Username:           "admin",
		Password:           "public",
		KeepAlive:          60 * time.Second,
		ConnectTimeout:     5 * time.Second,
		SessionExpiry:      60,
		CleanStart:         true,
		InsecureSkipVerify: true,
		TopicRoot:          "iov/v1",
	}
}

// Validate is used to parse and validate the parameters entered by the user at
// the command line when the program starts.
func (o *MqttOptions) Validate() []error {
	if o == nil {
		return nil
	}

	errors := []error{}

	return errors
}

// AddFlags adds flags for MqttOptions to the specified FlagSet.
func (o *MqttOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.Broker, "mqtt.broker", o.Broker, "The URL of the MQTT broker.")
	fs.StringVar(&o.Username, "mqtt.username", o.Username, "The username for MQTT authentication.")
	fs.StringVar(&o.Password, "mqtt.password", o.Password, "The password for MQTT authentication.")
	fs.StringVar(&o.ClientID, "mqtt.client-id", o.ClientID, "Explicit Client ID (optional, usually generated).")

	fs.DurationVar(&o.KeepAlive, "mqtt.keep-alive", o.KeepAlive, "MQTT Keep Alive interval.")
	fs.DurationVar(&o.ConnectTimeout, "mqtt.connect-timeout", o.ConnectTimeout, "Timeout for establishing MQTT connection.")
	fs.Uint32Var(&o.SessionExpiry, "mqtt.session-expiry", o.SessionExpiry, "MQTT Session Expiry Interval in seconds.")
	fs.BoolVar(&o.InsecureSkipVerify, "mqtt.insecure-skip-verify", o.InsecureSkipVerify, "If true, skips the TLS certificate verification.")

	// Topics
	fs.StringVar(&o.TopicRoot, "mqtt.topic-root", o.TopicRoot, "Topic prefix for sending commands.")
}

func (o *MqttOptions) ToClientConfig() *mqtt.ClientConfig {
	return &mqtt.ClientConfig{
		BrokerURL:          o.Broker,
		Username:           o.Username,
		Password:           o.Password,
		ClientID:           o.ClientID,
		KeepAlive:          uint16(o.KeepAlive.Seconds()),
		SessionExpiry:      o.SessionExpiry,
		ConnectTimeout:     o.ConnectTimeout,
		CleanStart:         o.CleanStart,
		InsecureSkipVerify: o.InsecureSkipVerify,
	}
}
