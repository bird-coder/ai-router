package llm

type LLM interface {
	HealthCheck() bool
}

type Features struct {
	InputTokens     int
	HasTools        bool
	IsCodeTask      bool
	IsRefactor      bool
	IsSimpleQuery   bool
	RequiresLongCtx bool
	RequiresHighIQ  bool
}

type LLMConfig struct {
	BaseUrl string
}
