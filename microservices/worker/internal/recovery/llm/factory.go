package llm

import (
	"fmt"
	"time"
)

// ProviderFactory creates LLM providers based on configuration
type ProviderFactory struct{}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

// Create creates an LLM provider based on the config
func (f *ProviderFactory) Create(cfg Config) (Provider, error) {
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	switch cfg.Provider {
	case "ollama":
		return NewOllamaClient(OllamaConfig{
			Endpoint: cfg.Endpoint,
			Model:    cfg.Model,
			Timeout:  timeout,
		}), nil

	case "openai":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("OpenAI API key is required")
		}
		return NewOpenAIClient(OpenAIConfig{
			Endpoint: cfg.Endpoint,
			APIKey:   cfg.APIKey,
			Model:    cfg.Model,
			Timeout:  timeout,
		}), nil

	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", cfg.Provider)
	}
}

// CreateDefault creates a default provider (Ollama with Qwen2.5)
func (f *ProviderFactory) CreateDefault() Provider {
	return NewOllamaClient(OllamaConfig{
		Endpoint: "http://localhost:11434",
		Model:    "qwen2.5",
		Timeout:  60 * time.Second,
	})
}
