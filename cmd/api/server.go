package api

import (
	"ai-router/internal/config"
	"ai-router/internal/hooks"

	"github.com/spf13/cobra"

	"github.com/bird-coder/manyo/lib/stage"
	"github.com/bird-coder/manyo/pkg/core"
	"github.com/bird-coder/manyo/pkg/logger"
	"github.com/bird-coder/manyo/pkg/server/httpx"
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

	appConfig *config.AppConfig
)

func init() {
	StartCmd.PersistentFlags().StringVarP(&configYml, "config", "c", "config/server.yaml", "Start server with provided configuration file")
}

func setup() {
	appConfig = new(config.AppConfig)
	if _, err := core.BuildWithProvider(configYml, appConfig); err != nil {
		panic(err)
	}
	logger.Info("starting api server...")
}

func run() error {
	logger.Info("ai-router server start")

	httpServer := httpx.NewHttpServer(&appConfig.Http)
	app := stage.NewApp(
		stage.WithServer(httpServer),
		stage.BeforeStart(hooks.BeforeStart),
		stage.AfterStart(hooks.AfterStart),
		stage.BeforeStop(hooks.BeforeStop),
		stage.AfterStop(hooks.AfterStop),
	)
	return app.Run()
}
