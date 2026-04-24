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
