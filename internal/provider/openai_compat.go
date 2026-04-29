package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"ai-router/internal/config"
	"ai-router/internal/util"
)

type OpenAICompat struct {
	name string
	cfg  config.OpenAICompatProviderConfig
}

func NewOpenAICompat(name string, cfg config.OpenAICompatProviderConfig) *OpenAICompat {
	return &OpenAICompat{
		name: name,
		cfg:  cfg,
	}
}

func (o *OpenAICompat) Run(ctx context.Context, req Request) (string, error) {
	baseURL := strings.TrimRight(o.cfg.BaseURL, "/")
	if baseURL == "" {
		return "", fmt.Errorf("provider %q base_url is required", o.name)
	}
	path := o.cfg.ChatCompletionsPath
	if strings.TrimSpace(path) == "" {
		path = "/v1/chat/completions"
	}

	payload := map[string]any{
		"model": req.Model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": req.Prompt,
			},
		},
		"stream": false,
	}

	headers := map[string]string{}
	if token := resolveSecret(o.cfg.APIKey, o.cfg.APIKeyEnv); token != "" {
		headers["Authorization"] = "Bearer " + token
	}
	for key, value := range o.cfg.Headers {
		headers[key] = value
	}
	resp, err := util.HttpPostJSON(baseURL+path, payload, headers)
	if err != nil {
		return "", fmt.Errorf("request provider %q: %w", o.name, err)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content any `json:"content"`
			} `json:"message"`
			Text string `json:"text"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return "", fmt.Errorf("decode provider response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("provider %q returned no choices", o.name)
	}

	if parsed.Choices[0].Text != "" {
		return parsed.Choices[0].Text, nil
	}
	return flattenContent(parsed.Choices[0].Message.Content), nil
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
			if text, ok := object["text"].(string); ok {
				parts = append(parts, text)
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return ""
	}
}

func resolveSecret(inline, envName string) string {
	if strings.TrimSpace(inline) != "" {
		return inline
	}
	if strings.TrimSpace(envName) == "" {
		return ""
	}
	return os.Getenv(envName)
}
