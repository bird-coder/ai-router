package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Server    ServerConfig `json:"server"`
	Providers Providers    `json:"providers"`
	Routes    []RouteRule  `json:"routes"`
}

type ServerConfig struct {
	Address        string `json:"address"`
	DefaultWorkdir string `json:"default_workdir"`
}

type Providers struct {
	CLIs         map[string]CLIProviderConfig          `json:"cli"`
	OpenAICompat map[string]OpenAICompatProviderConfig `json:"openai_compat"`
}

type CLIProviderConfig struct {
	Binary         string   `json:"binary"`
	Args           []string `json:"args"`
	DefaultWorkdir string   `json:"default_workdir"`
	OutputMode     string   `json:"output_mode"`
}

type OpenAICompatProviderConfig struct {
	BaseURL             string            `json:"base_url"`
	ChatCompletionsPath string            `json:"chat_completions_path"`
	APIKey              string            `json:"api_key"`
	APIKeyEnv           string            `json:"api_key_env"`
	Headers             map[string]string `json:"headers"`
	TimeoutSeconds      int               `json:"timeout_seconds"`
}

type RouteRule struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Priority    int         `json:"priority"`
	Match       RouteMatch  `json:"match"`
	Target      RouteTarget `json:"target"`
}

type RouteMatch struct {
	TaskTypes       []string `json:"task_types"`
	PromptContains  []string `json:"prompt_contains"`
	PreferredModels []string `json:"preferred_models"`
	Clients         []string `json:"clients"`
	Providers       []string `json:"providers"`
}

type RouteTarget struct {
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	ReasoningEffort string `json:"reasoning_effort"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}

	if cfg.Server.Address == "" {
		cfg.Server.Address = ":8080"
	}
	for name, cli := range cfg.Providers.CLIs {
		if cli.OutputMode == "" {
			cli.OutputMode = "stdout"
		}
		cfg.Providers.CLIs[name] = cli
	}

	return &cfg, nil
}

func (c *Config) DefaultTimeout() time.Duration {
	return 2 * time.Minute
}
