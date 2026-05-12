package api

import (
	"ai-router/internal/hooks"
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bird-coder/manyo/config"
	"github.com/bird-coder/manyo/lib/stage"
	"github.com/bird-coder/manyo/pkg/core"
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
)

func init() {
	StartCmd.PersistentFlags().StringVarP(&configYml, "config", "c", "config/server.yaml", "Start server with provided configuration file")
}

func setup() {
	if _, err := core.BuildDefault(configYml); err != nil {
		panic(err)
	}
	fmt.Println("starting api server...")
}

func run() error {
	defer core.Default().SyncLogger()
	log := core.Default().GetLogger(core.DEFAULT_KEY)
	log.Info("ai-router server start")

	ctx := context.Background()
	httpConfig, _ := core.GetCustomConfig[*config.HttpConfig](core.Default(), "http")
	httpServer := httpx.NewHttpServer(ctx, httpConfig)
	app := stage.NewApp(
		stage.WithContext(ctx),
		stage.WithServer(httpServer),
		stage.BeforeStart(hooks.BeforeStart),
		stage.AfterStart(hooks.AfterStart),
		stage.BeforeStop(hooks.BeforeStop),
		stage.AfterStop(hooks.AfterStop),
	)
	return app.Run()
}
