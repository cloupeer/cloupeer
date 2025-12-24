package server

import "github.com/autopeer-io/autopeer/pkg/options"

type Config struct {
	HttpOptions *options.HttpOptions
	GrpcOptions *options.GrpcOptions
	MqttOptions *options.MqttOptions
}
