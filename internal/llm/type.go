package llm

type Tier int

const (
	Cheap Tier = iota
	Balanced
	Premium
)

type LLM interface {
	HealthCheck() bool
}

type LLMConfig struct {
	BaseUrl string
}
