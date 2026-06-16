package types

type GenerateRequest struct {
	Prompt            string            `json:"prompt"`
	TaskType          string            `json:"task_type"`
	PreferredModel    string            `json:"preferred_model"`
	PreferredProvider string            `json:"preferred_provider"`
	Client            string            `json:"client"`
	Workdir           string            `json:"workdir"`
	TimeoutSeconds    int               `json:"timeout_seconds"`
	DryRun            bool              `json:"dry_run"`
	Metadata          map[string]string `json:"metadata"`
}

type GenerateResponse struct {
	Route  SelectedRoute `json:"route"`
	Output string        `json:"output,omitempty"`
	Error  string        `json:"error,omitempty"`
}

type SelectedRoute struct {
	RuleName        string `json:"rule_name"`
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	ResolvedWorkdir string `json:"resolved_workdir"`
}

type OpenAIResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type AnthropicResponse struct {
	ID         string        `json:"id"`
	Type       string        `json:"type"`
	Role       string        `json:"role"`
	Model      string        `json:"model"`
	Content    []ChatContent `json:"content"`
	StopReason string        `json:"stop_reason"`
}

type ChatContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
