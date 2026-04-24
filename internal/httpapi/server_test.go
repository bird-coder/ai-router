package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-router/internal/config"
	"ai-router/internal/provider"
	"ai-router/internal/router"
	"ai-router/internal/types"
)

type stubProvider struct {
	output string
}

func (s stubProvider) Run(context.Context, provider.Request) (string, error) {
	return s.output, nil
}

func TestHandleRouteDryRun(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			DefaultWorkdir: "/tmp/work",
		},
		Providers: config.Providers{
			CLIs: map[string]config.CLIProviderConfig{
				"codex": {
					Binary:         "codex",
					DefaultWorkdir: "/tmp/work",
				},
			},
		},
		Routes: []config.RouteRule{
			{
				Name:     "review-heavy",
				Priority: 100,
				Match: config.RouteMatch{
					TaskTypes: []string{"review"},
				},
				Target: config.RouteTarget{
					Provider:        "codex",
					Model:           "gpt-5.4",
					ReasoningEffort: "medium",
				},
			},
		},
	}

	server := New(cfg, router.New(cfg.Routes), provider.NewRegistry())

	body, err := json.Marshal(types.GenerateRequest{
		Prompt:   "review this code",
		TaskType: "review",
		DryRun:   true,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/route", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var resp types.GenerateResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Route.RuleName != "review-heavy" {
		t.Fatalf("rule_name = %q, want %q", resp.Route.RuleName, "review-heavy")
	}
	if resp.Route.Model != "gpt-5.4" {
		t.Fatalf("model = %q, want %q", resp.Route.Model, "gpt-5.4")
	}
	if resp.Route.ResolvedWorkdir != "/tmp/work" {
		t.Fatalf("resolved_workdir = %q, want %q", resp.Route.ResolvedWorkdir, "/tmp/work")
	}
}

func TestHandleOpenAIChatCompletions(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			DefaultWorkdir: "/tmp/work",
		},
		Routes: []config.RouteRule{
			{
				Name:     "fast-openai",
				Priority: 10,
				Match: config.RouteMatch{
					Clients: []string{"openai"},
				},
				Target: config.RouteTarget{
					Provider: "qwen",
					Model:    "qwen-plus",
				},
			},
		},
	}

	registry := provider.NewRegistry()
	registry.Register("qwen", stubProvider{output: "ok from qwen"})
	server := New(cfg, router.New(cfg.Routes), registry)

	body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"write a summary"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var resp struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Model != "gpt-5.4" {
		t.Fatalf("model = %q, want %q", resp.Model, "gpt-5.4")
	}
	if len(resp.Choices) != 1 || resp.Choices[0].Message.Content != "ok from qwen" {
		t.Fatalf("unexpected completion response: %s", recorder.Body.String())
	}
}

func TestHandleAnthropicMessages(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			DefaultWorkdir: "/tmp/work",
		},
		Routes: []config.RouteRule{
			{
				Name:     "anthropic-default",
				Priority: 10,
				Match: config.RouteMatch{
					Clients: []string{"anthropic"},
				},
				Target: config.RouteTarget{
					Provider: "claude-code",
					Model:    "claude-sonnet",
				},
			},
		},
	}

	registry := provider.NewRegistry()
	registry.Register("claude-code", stubProvider{output: "ok from claude"})
	server := New(cfg, router.New(cfg.Routes), registry)

	body := []byte(`{"model":"claude-sonnet-4","messages":[{"role":"user","content":"review this diff"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var resp struct {
		Model   string `json:"model"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Model != "claude-sonnet-4" {
		t.Fatalf("model = %q, want %q", resp.Model, "claude-sonnet-4")
	}
	if len(resp.Content) != 1 || resp.Content[0].Text != "ok from claude" {
		t.Fatalf("unexpected anthropic response: %s", recorder.Body.String())
	}
}
