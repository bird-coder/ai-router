package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"ai-router/internal/config"
)

type OpenAICompat struct {
	name   string
	cfg    config.OpenAICompatProviderConfig
	client *http.Client
}

func NewOpenAICompat(name string, cfg config.OpenAICompatProviderConfig) *OpenAICompat {
	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = 120
	}
	return &OpenAICompat{
		name: name,
		cfg:  cfg,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
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

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal provider request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build provider request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if token := resolveSecret(o.cfg.APIKey, o.cfg.APIKeyEnv); token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}
	for key, value := range o.cfg.Headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request provider %q: %w", o.name, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read provider response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("provider %q returned %d: %s", o.name, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content any `json:"content"`
			} `json:"message"`
			Text string `json:"text"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
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
