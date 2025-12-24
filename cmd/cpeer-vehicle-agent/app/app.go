package app

import (
	"fmt"

	genericapiserver "k8s.io/apiserver/pkg/server"

	"github.com/autopeer-io/autopeer/cmd/cpeer-vehicle-agent/app/options"
	"github.com/autopeer-io/autopeer/pkg/app"
)

const (
	commandName = "cpeer-edge-agent"
	commandDesc = `The Autopeer Edge Agent runs on edge, reporting its status to the
cpeer-hub and executing tasks such as firmware upgrades.`
)

func NewApp() *app.App {
	opts := options.NewAgentOptions()
	application := app.NewApp(
		commandName,
		"Launch a Autopeer edge agent",
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
