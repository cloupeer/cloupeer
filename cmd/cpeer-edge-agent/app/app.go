package app

import (
	"fmt"

	genericapiserver "k8s.io/apiserver/pkg/server"

	"cloupeer.io/cloupeer/cmd/cpeer-edge-agent/app/options"
	"cloupeer.io/cloupeer/pkg/app"
)

const (
	commandName = "cpeer-edge-agent"
	commandDesc = `The Cloupeer Edge Agent runs on edge devices, reporting its status to the
cpeer-hub and executing tasks such as firmware upgrades.`
)

func NewApp() *app.App {
	opts := options.NewAgentOptions()
	application := app.NewApp(
		commandName,
		"Launch a Cloupeer edge agent",
		app.WithDescription(commandDesc),
		app.WithOptions(opts),
		app.WithDefaultValidArgs(),
		app.WithRunFunc(run(opts)),
	)
	return application
}

func run(opts *options.AgentOptions) app.RunFunc {
	return func() error {
		ctx := genericapiserver.SetupSignalContext()

		cfg, err := opts.Config()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		agent, err := cfg.NewAgent()
		if err != nil {
			return fmt.Errorf("failed to create agent: %w", err)
		}

		return agent.Run(ctx)
	}
}
