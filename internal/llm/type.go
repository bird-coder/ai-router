package llm

import (
	"sync"
	"time"

	"github.com/bird-coder/manyo/util"
)

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
	ApiKey  string
}

type LLMBase struct {
	LLMConfig

	Available bool
	LastCheck time.Time

	mu sync.RWMutex
}

func (llm *LLMBase) HealthCheck() bool {
	headers := map[string]string{
		"Authorization": "Bearer " + llm.ApiKey,
	}
	if err := util.HttpHead(llm.BaseUrl, headers); err != nil {
		return false
	}
	llm.mu.Lock()
	defer llm.mu.Unlock()
	llm.Available = true
	llm.LastCheck = time.Now()
	return true
}
