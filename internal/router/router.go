package router

import (
	"slices"
	"strings"

	"ai-router/internal/config"
	"ai-router/internal/types"
)

type Engine struct {
	rules []config.RouteRule
}

type Decision struct {
	Rule   config.RouteRule
	Target config.RouteTarget
}

func New(rules []config.RouteRule) *Engine {
	sorted := append([]config.RouteRule(nil), rules...)
	slices.SortFunc(sorted, func(a, b config.RouteRule) int {
		switch {
		case a.Priority > b.Priority:
			return -1
		case a.Priority < b.Priority:
			return 1
		default:
			return strings.Compare(a.Name, b.Name)
		}
	})
	return &Engine{rules: sorted}
}

func (e *Engine) Resolve(req types.GenerateRequest) Decision {
	for _, rule := range e.rules {
		if matches(rule.Match, req) {
			return Decision{Rule: rule, Target: overrideTarget(rule.Target, req)}
		}
	}

	fallback := config.RouteRule{
		Name: "implicit-default",
		Target: config.RouteTarget{
			Provider: defaultProvider(req),
			Model:    defaultModel(req),
		},
	}
	return Decision{Rule: fallback, Target: fallback.Target}
}

func overrideTarget(target config.RouteTarget, req types.GenerateRequest) config.RouteTarget {
	if req.PreferredProvider != "" {
		target.Provider = req.PreferredProvider
	}
	if req.PreferredModel != "" {
		target.Model = req.PreferredModel
	}
	if target.Provider == "" {
		target.Provider = defaultProvider(req)
	}
	if target.Model == "" {
		target.Model = defaultModel(req)
	}
	return target
}

func defaultProvider(req types.GenerateRequest) string {
	if req.PreferredProvider != "" {
		return req.PreferredProvider
	}
	switch req.TaskType {
	case "review", "architecture", "complex":
		return "codex"
	default:
		return "qwen"
	}
}

func defaultModel(req types.GenerateRequest) string {
	switch req.TaskType {
	case "review", "architecture", "complex":
		return "gpt-5.4"
	case "fast", "triage":
		return "gpt-5.4-mini"
	default:
		return "gpt-5.4-mini"
	}
}

func matches(match config.RouteMatch, req types.GenerateRequest) bool {
	if len(match.TaskTypes) > 0 && !containsFold(match.TaskTypes, req.TaskType) {
		return false
	}
	if len(match.PreferredModels) > 0 && !containsFold(match.PreferredModels, req.PreferredModel) {
		return false
	}
	if len(match.Clients) > 0 && !containsFold(match.Clients, req.Client) {
		return false
	}
	if len(match.Providers) > 0 && !containsFold(match.Providers, req.PreferredProvider) {
		return false
	}
	for _, fragment := range match.PromptContains {
		if !strings.Contains(strings.ToLower(req.Prompt), strings.ToLower(fragment)) {
			return false
		}
	}
	return true
}

func containsFold(values []string, needle string) bool {
	for _, value := range values {
		if strings.EqualFold(value, needle) {
			return true
		}
	}
	return false
}
