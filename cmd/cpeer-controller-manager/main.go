package main

import (
	"os"

	"cloupeer.io/cloupeer/cmd/cpeer-controller-manager/app"
	"k8s.io/apiserver/pkg/server"
)

func main() {
	ctx := server.SetupSignalContext()
	if err := app.NewControllerManagerCommand(ctx).Execute(); err != nil {
		os.Exit(1)
	}
}
