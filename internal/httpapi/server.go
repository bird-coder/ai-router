package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"ai-router/internal/config"
	"ai-router/internal/provider"
	"ai-router/internal/router"
	"ai-router/internal/types"
)

type Server struct {
	cfg      *config.Config
	router   *router.Engine
	registry *provider.Registry
	mux      *http.ServeMux
}

func New(cfg *config.Config, rt *router.Engine, registry *provider.Registry) *Server {
	s := &Server{
		cfg:      cfg,
		router:   rt,
		registry: registry,
		mux:      http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("/healthz", s.handleHealthz)
	s.mux.HandleFunc("/v1/route", s.handleRoute)
	s.mux.HandleFunc("/v1/chat/completions", s.handleOpenAIChatCompletions)
	s.mux.HandleFunc("/v1/messages", s.handleAnthropicMessages)
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req types.GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "prompt is required"})
		return
	}
	resp, status := s.execute(r.Context(), req)
	writeJSON(w, status, resp)
}

func (s *Server) resolveWorkdir(workdir string) string {
	switch {
	case strings.TrimSpace(workdir) != "":
		return workdir
	case strings.TrimSpace(s.cfg.Server.DefaultWorkdir) != "":
		return filepath.Clean(s.cfg.Server.DefaultWorkdir)
	default:
		for _, cli := range s.cfg.Providers.CLIs {
			if strings.TrimSpace(cli.DefaultWorkdir) != "" {
				return filepath.Clean(cli.DefaultWorkdir)
			}
		}
		return "."
	}
}

func (s *Server) handleOpenAIChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Model    string        `json:"model"`
		Messages []chatMessage `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	prompt := flattenMessages(req.Messages)
	generateReq := types.GenerateRequest{
		Prompt:         prompt,
		PreferredModel: req.Model,
		Client:         "openai",
		TaskType:       inferTaskType(prompt),
	}
	resp, status := s.execute(r.Context(), generateReq)
	if resp.Error != "" {
		writeJSON(w, status, map[string]any{"error": map[string]any{"message": resp.Error}})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":      "chatcmpl-ai-router",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   resp.Route.Model,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": resp.Output,
				},
				"finish_reason": "stop",
			},
		},
	})
}

func (s *Server) handleAnthropicMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Model    string        `json:"model"`
		Messages []chatMessage `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	prompt := flattenMessages(req.Messages)
	generateReq := types.GenerateRequest{
		Prompt:         prompt,
		PreferredModel: req.Model,
		Client:         "anthropic",
		TaskType:       inferTaskType(prompt),
	}
	resp, status := s.execute(r.Context(), generateReq)
	if resp.Error != "" {
		writeJSON(w, status, map[string]any{"error": map[string]any{"message": resp.Error}})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":    "msg_ai_router",
		"type":  "message",
		"role":  "assistant",
		"model": resp.Route.Model,
		"content": []map[string]any{
			{
				"type": "text",
				"text": resp.Output,
			},
		},
		"stop_reason": "end_turn",
	})
}

func (s *Server) execute(parent context.Context, req types.GenerateRequest) (types.GenerateResponse, int) {
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

	timeout := s.cfg.DefaultTimeout()
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

type chatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

func flattenMessages(messages []chatMessage) string {
	var parts []string
	for _, message := range messages {
		text := flattenContent(message.Content)
		if strings.TrimSpace(text) == "" {
			continue
		}
		parts = append(parts, message.Role+": "+text)
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func flattenContent(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		var parts []string
		for _, item := range typed {
			object, ok := item.(map[string]any)
			if !ok {
				continue
			}
			switch object["type"] {
			case "text":
				if text, ok := object["text"].(string); ok {
					parts = append(parts, text)
				}
			case "input_text":
				if text, ok := object["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return ""
	}
}

func inferTaskType(prompt string) string {
	lower := strings.ToLower(prompt)
	switch {
	case strings.Contains(lower, "review"):
		return "review"
	case strings.Contains(lower, "architecture"):
		return "architecture"
	case strings.Contains(lower, "fix"), strings.Contains(lower, "debug"):
		return "complex"
	default:
		return "fast"
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
