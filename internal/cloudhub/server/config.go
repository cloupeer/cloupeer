package server

import "cloupeer.io/cloupeer/pkg/options"

type Config struct {
	HttpOptions *options.HttpOptions
	GrpcOptions *options.GrpcOptions
	MqttOptions *options.MqttOptions
}
