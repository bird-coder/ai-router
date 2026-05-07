package api

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bird-coder/manyo/lib/stage"
	"github.com/bird-coder/manyo/pkg/core"
)

var (
	configYml string
	StartCmd  = &cobra.Command{
		Use:          "server",
		Short:        "Start router server",
		Example:      "ai-router server -c config/server.yaml",
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			setup()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return run()
		},
	}
)

func init() {
	StartCmd.PersistentFlags().StringVarP(&configYml, "config", "c", "config/server.yaml", "Start server with provided configuration file")
}

func setup() {
	if err := core.Kernal.Init(configYml); err != nil {
		panic(err)
	}
	fmt.Println("starting api server...")
}

func run() error {
	defer core.Kernal.SyncLogger()
	log := core.Kernal.GetLogger("default")
	log.Info("ai-router server start")

	app := stage.NewApp(
		stage.WithServer(),
		stage.BeforeStart(),
		stage.AfterStart(),
		stage.BeforeStop(),
		stage.AfterStop(),
	)
	return app.Run()
}
