package service

import (
	"ai-router/internal/provider"
	"ai-router/internal/router"
	"ai-router/internal/types"
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

var (
	defaultWorkdir = "/Users/jiajie.yu/go/src"
	defaultTimeout = 2 * time.Minute
)

type TransferService struct {
	router   *router.Engine
	registry *provider.Registry
}

func NewTransferService(rt *router.Engine, registry *provider.Registry) *TransferService {
	return &TransferService{
		router:   rt,
		registry: registry,
	}
}

func (s *TransferService) Transfer(parent context.Context, req types.GenerateRequest) (types.GenerateResponse, int) {
	decision := s.router.Resolve(req)
	workdir := s.resolveWorkdir(req.Workdir)
	selected := provider.RouteSummary(
		req,
		decision.Target.Provider,
		decision.Target.Model,
		decision.Target.ReasoningEffort,
		workdir,
		decision.Rule.Name,
	)

	if req.DryRun {
		return types.GenerateResponse{Route: selected}, http.StatusOK
	}

	executor, err := s.registry.MustGet(decision.Target.Provider)
	if err != nil {
		return types.GenerateResponse{Route: selected, Error: err.Error()}, http.StatusInternalServerError
	}

	timeout := defaultTimeout
	if req.TimeoutSeconds > 0 {
		timeout = time.Duration(req.TimeoutSeconds) * time.Second
	}

	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	output, err := executor.Run(ctx, provider.Request{
		Prompt:          req.Prompt,
		Model:           decision.Target.Model,
		ReasoningEffort: decision.Target.ReasoningEffort,
		Workdir:         workdir,
		Client:          req.Client,
	})
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			status = http.StatusGatewayTimeout
		}
		return types.GenerateResponse{Route: selected, Error: err.Error()}, status
	}

	return types.GenerateResponse{Route: selected, Output: output}, http.StatusOK
}

func (s *TransferService) resolveWorkdir(workdir string) string {
	if strings.TrimSpace(workdir) != "" {
		return workdir
	}
	if strings.TrimSpace(defaultWorkdir) != "" {
		return filepath.Clean(defaultWorkdir)
	}
	return "."
}
