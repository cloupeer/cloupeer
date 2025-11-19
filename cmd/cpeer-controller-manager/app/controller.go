package app

import (
	"context"
	"flag"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/component-base/featuregate"
	controllerruntime "sigs.k8s.io/controller-runtime"

	"cloupeer.io/cloupeer/cmd/cpeer-controller-manager/app/options"
	"cloupeer.io/cloupeer/internal/controller"
	"cloupeer.io/cloupeer/pkg/log"
)

func NewControllerManagerCommand(ctx context.Context) *cobra.Command {
	opts := options.NewControllerManagerOptions()
	cmd := &cobra.Command{
		Use:  "cpeer-controller-manager",
		Long: "The Cloupeer Controller Manager is a daemon that embeds the core control loops for the Cloupeer platform.",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Init(opts.LogOptions)
			controllerruntime.SetLogger(log.Std().Logr())

			gate := featuregate.NewFeatureGate()
			for _, fg := range opts.FeatureGates {
				if err := gate.Set(fmt.Sprintf("%s=true", fg)); err != nil {
					log.Error(err, "failed to set feature gate", "featureGate", fg)
				}
			}

			kubeconfig := controllerruntime.GetConfigOrDie()
			mgr, err := controller.NewControllerManager(ctx, kubeconfig, opts.HealthProbeBindAddress, opts.HubAddr)
			if err != nil {
				log.Error(err, "failed to new controller manager")
				return err
			}

			if err = mgr.Start(ctx); err != nil {
				log.Error(err, "failed to start controller manager")
				return err
			}

			return nil
		},
	}

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	fs := cmd.Flags()
	namedfs := opts.Flags()
	globalflag.AddGlobalFlags(namedfs.FlagSet("global"), cmd.Name())
	for _, f := range namedfs.FlagSets {
		fs.AddFlagSet(f)
	}

	return cmd
}
