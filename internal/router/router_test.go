package router

import (
	"testing"

	"ai-router/internal/config"
	"ai-router/internal/types"
)

func TestResolvePrefersHigherPriorityRule(t *testing.T) {
	engine := New([]config.RouteRule{
		{
			Name:     "default",
			Priority: 10,
			Target: config.RouteTarget{
				Provider: "codex",
				Model:    "gpt-5.4-mini",
			},
		},
		{
			Name:     "review",
			Priority: 100,
			Match: config.RouteMatch{
				TaskTypes: []string{"review"},
			},
			Target: config.RouteTarget{
				Provider: "codex",
				Model:    "gpt-5.4",
			},
		},
	})

	decision := engine.Resolve(types.GenerateRequest{
		Prompt:   "review this diff",
		TaskType: "review",
	})

	if decision.Rule.Name != "review" {
		t.Fatalf("rule = %q, want %q", decision.Rule.Name, "review")
	}
	if decision.Target.Model != "gpt-5.4" {
		t.Fatalf("model = %q, want %q", decision.Target.Model, "gpt-5.4")
	}
}

func TestResolveAllowsPreferredModelOverride(t *testing.T) {
	engine := New([]config.RouteRule{
		{
			Name:     "default",
			Priority: 1,
			Target: config.RouteTarget{
				Provider: "codex",
				Model:    "gpt-5.4-mini",
			},
		},
	})

	decision := engine.Resolve(types.GenerateRequest{
		Prompt:         "write a quick summary",
		PreferredModel: "gpt-5.4",
	})

	if decision.Target.Model != "gpt-5.4" {
		t.Fatalf("model = %q, want preferred model", decision.Target.Model)
	}
}

func TestResolveMatchesClientSpecificRule(t *testing.T) {
	engine := New([]config.RouteRule{
		{
			Name:     "generic",
			Priority: 1,
			Target: config.RouteTarget{
				Provider: "qwen",
				Model:    "qwen-plus",
			},
		},
		{
			Name:     "anthropic-client",
			Priority: 10,
			Match: config.RouteMatch{
				Clients: []string{"anthropic"},
			},
			Target: config.RouteTarget{
				Provider: "claude-code",
				Model:    "claude-sonnet",
			},
		},
	})

	decision := engine.Resolve(types.GenerateRequest{
		Prompt: "review this",
		Client: "anthropic",
	})

	if decision.Rule.Name != "anthropic-client" {
		t.Fatalf("rule = %q, want %q", decision.Rule.Name, "anthropic-client")
	}
	if decision.Target.Provider != "claude-code" {
		t.Fatalf("provider = %q, want %q", decision.Target.Provider, "claude-code")
	}
}
