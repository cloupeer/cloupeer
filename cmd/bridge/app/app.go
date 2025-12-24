package app

import (
	"context"
	"fmt"

	genericapiserver "k8s.io/apiserver/pkg/server"

	"github.com/autopeer-io/autopeer/cmd/bridge/app/options"
	"github.com/autopeer-io/autopeer/pkg/app"
)

const (
	commandName = "autopeer-bridge"
	commandDesc = `The Autopeer bridge runs ...`
)

func NewApp() *app.App {
	opts := options.NewHubOptions()
	application := app.NewApp(
		commandName,
		"Launch a Autopeer hub server",
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
