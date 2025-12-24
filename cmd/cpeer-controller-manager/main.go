package main

import (
	"os"

	"k8s.io/apiserver/pkg/server"

	"github.com/autopeer-io/autopeer/cmd/cpeer-controller-manager/app"
)

func main() {
	ctx := server.SetupSignalContext()
	if err := app.NewControllerManagerCommand(ctx).Execute(); err != nil {
		os.Exit(1)
	}
}
