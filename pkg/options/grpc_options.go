package options

import (
	"time"

	"github.com/spf13/pflag"
)

var _ IOptions = (*GrpcOptions)(nil)

// GrpcOptions are for creating an unauthenticated, unauthorized, insecure port.
type GrpcOptions struct {
	// Network with server network.
	Network string `json:"network" mapstructure:"network"`

	// Address with server address.
	Addr string `json:"addr" mapstructure:"addr"`

	// Timeout with server timeout. Used by grpc client side.
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`
}

// NewGrpcOptions is for creating an unauthenticated, unauthorized, insecure port.
func NewGrpcOptions() *GrpcOptions {
	return &GrpcOptions{
		Network: "tcp",
		Addr:    "0.0.0.0:8091",
		Timeout: 30 * time.Second,
	}
}

// Validate is used to parse and validate the parameters entered by the user at
// the command line when the program starts.
func (o *GrpcOptions) Validate() []error {
	var errors []error

	if err := ValidateAddress(o.Addr); err != nil {
		errors = append(errors, err)
	}

	return errors
}

// AddFlags adds flags related to features for a specific api server to the
// specified FlagSet.
func (o *GrpcOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.Network, "grpc.network", o.Network, "Specify the network for the gRPC server.")
	fs.StringVar(&o.Addr, "grpc.addr", o.Addr, "Specify the gRPC server bind address and port.")
	fs.DurationVar(&o.Timeout, "grpc.timeout", o.Timeout, "Timeout for server connections.")
}
