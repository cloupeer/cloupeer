package options

import (
	"time"

	"github.com/spf13/pflag"
)

var _ IOptions = (*HttpOptions)(nil)

// HttpOptions contains configuration items related to HTTP server startup.
type HttpOptions struct {
	// Network with server network.
	Network string `json:"network" mapstructure:"network"`

	// Address with server address.
	Addr string `json:"addr" mapstructure:"addr"`

	// Timeout with server timeout. Used by http client side.
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`
}

// NewHttpOptions creates a HttpOptions object with default parameters.
func NewHttpOptions() *HttpOptions {
	return &HttpOptions{
		Network: "tcp",
		Addr:    "0.0.0.0:8443",
		Timeout: 30 * time.Second,
	}
}

// Validate is used to parse and validate the parameters entered by the user at
// the command line when the program starts.
func (o *HttpOptions) Validate() []error {
	if o == nil {
		return nil
	}

	errors := []error{}

	if err := ValidateAddress(o.Addr); err != nil {
		errors = append(errors, err)
	}

	return errors
}

// AddFlags adds flags related to HTTPS server for a specific APIServer to the
// specified FlagSet.
func (o *HttpOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.Network, "http.network", o.Network, "Specify the network for the HTTP server.")
	fs.StringVar(&o.Addr, "http.addr", o.Addr, "Specify the HTTP server bind address and port.")
	fs.DurationVar(&o.Timeout, "http.timeout", o.Timeout, "Timeout for server connections.")
}
