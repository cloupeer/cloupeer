package hub

import (
	"net/http"
	"time"
)

type Config struct {
	Namespace string
	HttpAddr  string // HTTP Address (e.g., :8080)
	GrpcAddr  string // gRPC Address (e.g., :8081)
}

func (cfg *Config) NewHubServer() (*HubServer, error) {
	return &HubServer{
		namespace:  cfg.Namespace,
		httpAddr:   cfg.HttpAddr,
		grpcAddr:   cfg.GrpcAddr,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}, nil
}
