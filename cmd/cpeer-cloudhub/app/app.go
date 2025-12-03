package app

import (
	"context"
	"fmt"

	genericapiserver "k8s.io/apiserver/pkg/server"

	"cloupeer.io/cloupeer/cmd/cpeer-cloudhub/app/options"
	"cloupeer.io/cloupeer/pkg/app"
)

const (
	commandName = "cpeer-hub"
	commandDesc = `The Cloupeer Hub runs ...`
)

func NewApp() *app.App {
	opts := options.NewHubOptions()
	application := app.NewApp(
		commandName,
		"Launch a Cloupeer hub server",
		app.WithDescription(commandDesc),
		app.WithOptions(opts),
		app.WithDefaultValidArgs(),
		app.WithRunFunc(run(opts)),
		app.WithLoggerContextExtractor(map[string]func(context.Context) string{}),
	)
	return application
}

func run(opts *options.HubOptions) app.RunFunc {
	return func() error {
		ctx := genericapiserver.SetupSignalContext()

		cfg, err := opts.Config()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		server, err := cfg.NewHubServer()
		if err != nil {
			return fmt.Errorf("failed to create hub server: %w", err)
		}

		return server.Run(ctx)
	}
}
