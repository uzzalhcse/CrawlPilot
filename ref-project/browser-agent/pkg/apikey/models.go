package apikey

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/genai"
)

// Model wraps a chat model with API key rotation support
type Model struct {
	provider   string
	manager    *Manager
	baseConfig interface{} // stores base configuration
	model      interface{} // can be ChatModel or ToolCallingChatModel
	currentKey *APIKey
}

// NewGeminiModel creates a Gemini model with API key rotation
func NewGeminiModel(
	ctx context.Context,
	manager *Manager,
	modelName string,
	thinkingConfig *genai.ThinkingConfig,
) (*Model, error) {
	// Get initial key
	key, err := manager.GetKey(ctx, "gemini")
	if err != nil {
		return nil, fmt.Errorf("failed to get Gemini API key: %w", err)
	}

	// Create Gemini client
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: key.Key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Create chat model
	model, err := gemini.NewChatModel(ctx, &gemini.Config{
		Client:         client,
		Model:          modelName,
		ThinkingConfig: thinkingConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini chat model: %w", err)
	}

	return &Model{
		provider: "gemini",
		manager:  manager,
		baseConfig: map[string]interface{}{
			"model":          modelName,
			"thinkingConfig": thinkingConfig,
		},
		model:      model,
		currentKey: key,
	}, nil
}

// NewOpenAIModel creates an OpenAI model with API key rotation
func NewOpenAIModel(
	ctx context.Context,
	manager *Manager,
	modelName string,
) (*Model, error) {
	// Get initial key
	key, err := manager.GetKey(ctx, "openai")
	if err != nil {
		return nil, fmt.Errorf("failed to get OpenAI API key: %w", err)
	}

	// Create OpenAI model
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:  modelName,
		APIKey: key.Key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI chat model: %w", err)
	}

	return &Model{
		provider: "openai",
		manager:  manager,
		baseConfig: map[string]interface{}{
			"model": modelName,
		},
		model:      model,
		currentKey: key,
	}, nil
}

// rotateKey rotates to a new API key and recreates the model
func (m *Model) rotateKey(ctx context.Context) error {
	// Get new key (excluding current one)
	key, err := m.manager.GetNextAvailableKey(ctx, m.provider, []primitive.ObjectID{m.currentKey.ID})
	if err != nil {
		return fmt.Errorf("failed to get next API key: %w", err)
	}

	// Recreate model based on provider
	switch m.provider {
	case "gemini":
		config := m.baseConfig.(map[string]interface{})
		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey: key.Key,
		})
		if err != nil {
			return fmt.Errorf("failed to create Gemini client: %w", err)
		}

		model, err := gemini.NewChatModel(ctx, &gemini.Config{
			Client:         client,
			Model:          config["model"].(string),
			ThinkingConfig: config["thinkingConfig"].(*genai.ThinkingConfig),
		})
		if err != nil {
			return fmt.Errorf("failed to create Gemini chat model: %w", err)
		}

		m.model = model
		m.currentKey = key

	case "openai":
		config := m.baseConfig.(map[string]interface{})
		model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
			Model:  config["model"].(string),
			APIKey: key.Key,
		})
		if err != nil {
			return fmt.Errorf("failed to create OpenAI chat model: %w", err)
		}

		m.model = model
		m.currentKey = key

	case "qwen":
		config := m.baseConfig.(map[string]interface{})
		model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
			Model:   config["model"].(string),
			APIKey:  key.Key,
			BaseURL: config["baseURL"].(string),
		})
		if err != nil {
			return fmt.Errorf("failed to create Qwen chat model: %w", err)
		}

		m.model = model
		m.currentKey = key

	default:
		return fmt.Errorf("unsupported provider: %s", m.provider)
	}

	fmt.Printf("🔄 Rotated to new API key: %s\n", maskKey(key.Key))
	return nil
}

// Generate wraps the model's Generate method with retry logic
func (m *Model) Generate(
	ctx context.Context,
	input []*schema.Message,
	opts ...model.Option,
) (*schema.Message, error) {
	// For local models (qwen), skip RetryWithRotation since they don't use MongoDB keys
	if m.provider == "qwen" {
		// Call the actual model directly (type assert to BaseChatModel)
		baseModel := m.model.(model.BaseChatModel)
		return baseModel.Generate(ctx, input, opts...)
	}

	// For cloud providers (openai, gemini), use retry with rotation
	config := DefaultRetryConfig()

	result, err := m.manager.RetryWithRotation(ctx, m.provider, config,
		func(ctx context.Context, apiKey string) (interface{}, error) {
			// Make sure we're using the right key
			if apiKey != m.currentKey.Key {
				if err := m.rotateKey(ctx); err != nil {
					return nil, err
				}
			}

			// Call the actual model (type assert to BaseChatModel)
			baseModel := m.model.(model.BaseChatModel)
			return baseModel.Generate(ctx, input, opts...)
		})

	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, result.Error
	}

	return result.Response.(*schema.Message), nil
}

// Stream wraps the model's Stream method with retry logic
func (m *Model) Stream(
	ctx context.Context,
	input []*schema.Message,
	opts ...model.Option,
) (*schema.StreamReader[*schema.Message], error) {
	// For streaming, we do a simple retry without rotation during stream
	// If rate limited, the next request will use a different key
	baseModel := m.model.(model.BaseChatModel)
	return baseModel.Stream(ctx, input, opts...)
}

// BindTools wraps the model's BindTools method
func (m *Model) BindTools(tools []*schema.ToolInfo) error {
	// Bind tools to the underlying model if it supports ChatModel interface
	if chatModel, ok := m.model.(model.ChatModel); ok {
		return chatModel.BindTools(tools)
	}
	// If not a ChatModel, we can't bind tools this way
	return fmt.Errorf("model does not support BindTools method")
}

// WithTools returns a new ToolCallingChatModel instance with tools bound
func (m *Model) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	// Check if the underlying model implements ToolCallingChatModel
	toolCallingModel, ok := m.model.(model.ToolCallingChatModel)
	if !ok {
		// Fallback: if the model doesn't support WithTools, try BindTools
		newModel := &Model{
			provider:   m.provider,
			manager:    m.manager,
			baseConfig: m.baseConfig,
			model:      m.model,
			currentKey: m.currentKey,
		}
		err := newModel.BindTools(tools)
		if err != nil {
			return nil, err
		}
		return newModel, nil
	}

	// Call the underlying model's WithTools method
	boundModel, err := toolCallingModel.WithTools(tools)
	if err != nil {
		return nil, err
	}

	// Return new wrapper with bound model
	return &Model{
		provider:   m.provider,
		manager:    m.manager,
		baseConfig: m.baseConfig,
		model:      boundModel,
		currentKey: m.currentKey,
	}, nil
}

// GetCurrentKey returns the currently active API key
func (m *Model) GetCurrentKey() *APIKey {
	return m.currentKey
}

// GetManager returns the API key manager
func (m *Model) GetManager() *Manager {
	return m.manager
}
