package hub

type Config struct {
	Namespace string
	Addr      string
}

func (cfg *Config) NewHubServer() (*HubServer, error) {
	return &HubServer{
		namespace: cfg.Namespace,
		addr:      cfg.Addr,
	}, nil
}
