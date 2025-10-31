package edgeagent

import (
	"net/http"
	"time"
)

type Config struct {
}

func (cfg *Config) NewAgent() (*Agent, error) {
	return &Agent{
		client: &http.Client{Timeout: 10 * time.Second},
	}, nil
}
