package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-router/internal/config"
	"ai-router/internal/httpapi"
	"ai-router/internal/provider"
	"ai-router/internal/router"
)

func main() {
	cfgPath := os.Getenv("AI_ROUTER_CONFIG")
	if cfgPath == "" {
		cfgPath = "config.example.json"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	registry := provider.NewRegistry()
	for name, cliCfg := range cfg.Providers.CLIs {
		registry.Register(name, provider.NewCLI(cliCfg))
	}
	for name, httpCfg := range cfg.Providers.OpenAICompat {
		registry.Register(name, provider.NewOpenAICompat(name, httpCfg))
	}

	engine := router.New(cfg.Routes)
	server := httpapi.New(cfg, engine, registry)

	httpServer := &http.Server{
		Addr:              cfg.Server.Address,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	log.Printf("ai-router listening on %s", cfg.Server.Address)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
}
