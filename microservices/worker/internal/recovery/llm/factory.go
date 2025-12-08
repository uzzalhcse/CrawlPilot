package llm

import (
	"fmt"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
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

// CreateForRecovery creates a provider for recovery agent using MultiConfig
// Falls back to probe config if recovery is empty
func (f *ProviderFactory) CreateForRecovery(mc MultiConfig) (Provider, error) {
	cfg := mc.GetRecoveryConfig()
	if cfg.IsEmpty() {
		logger.Warn("No LLM config for recovery agent, using default Ollama")
		return f.CreateDefault(), nil
	}

	logger.Info("Creating LLM provider for recovery agent",
		zap.String("provider", cfg.Provider),
		zap.String("model", cfg.Model),
	)
	return f.Create(cfg)
}

// CreateForProbe creates a provider for probe agent using MultiConfig
// Falls back to recovery config if probe is empty
func (f *ProviderFactory) CreateForProbe(mc MultiConfig) (Provider, error) {
	cfg := mc.GetProbeConfig()
	if cfg.IsEmpty() {
		logger.Warn("No LLM config for probe agent, using default Ollama")
		return f.CreateDefault(), nil
	}

	logger.Info("Creating LLM provider for probe agent",
		zap.String("provider", cfg.Provider),
		zap.String("model", cfg.Model),
	)
	return f.Create(cfg)
}

// CreateBoth creates both recovery and probe providers
// Returns (recoveryProvider, probeProvider, error)
// If both configs point to the same model, returns the same provider instance for efficiency
func (f *ProviderFactory) CreateBoth(mc MultiConfig) (Provider, Provider, error) {
	recoveryCfg := mc.GetRecoveryConfig()
	probeCfg := mc.GetProbeConfig()

	// Check if they're the same config (share provider)
	if recoveryCfg.Provider == probeCfg.Provider &&
		recoveryCfg.Model == probeCfg.Model &&
		recoveryCfg.Endpoint == probeCfg.Endpoint {
		provider, err := f.Create(recoveryCfg)
		if err != nil {
			return nil, nil, err
		}
		logger.Info("Using single LLM provider for both agents",
			zap.String("provider", recoveryCfg.Provider),
			zap.String("model", recoveryCfg.Model),
		)
		return provider, provider, nil
	}

	// Create separate providers
	recoveryProvider, err := f.CreateForRecovery(mc)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create recovery provider: %w", err)
	}

	probeProvider, err := f.CreateForProbe(mc)
	if err != nil {
		recoveryProvider.Close()
		return nil, nil, fmt.Errorf("failed to create probe provider: %w", err)
	}

	logger.Info("Using separate LLM providers",
		zap.String("recovery_provider", recoveryCfg.Provider),
		zap.String("recovery_model", recoveryCfg.Model),
		zap.String("probe_provider", probeCfg.Provider),
		zap.String("probe_model", probeCfg.Model),
	)

	return recoveryProvider, probeProvider, nil
}
