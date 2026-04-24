package provider

import (
	"context"
	"fmt"

	"ai-router/internal/types"
)

type Request struct {
	Prompt          string
	Model           string
	ReasoningEffort string
	Workdir         string
	Client          string
}

type Provider interface {
	Run(context.Context, Request) (string, error)
}

type Registry struct {
	providers map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{providers: map[string]Provider{}}
}

func (r *Registry) Register(name string, provider Provider) {
	r.providers[name] = provider
}

func (r *Registry) MustGet(name string) (Provider, error) {
	provider, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %q not registered", name)
	}
	return provider, nil
}

func RouteSummary(req types.GenerateRequest, providerName, model, effort, workdir, ruleName string) types.SelectedRoute {
	return types.SelectedRoute{
		RuleName:        ruleName,
		Provider:        providerName,
		Model:           model,
		ReasoningEffort: effort,
		ResolvedWorkdir: workdir,
	}
}
