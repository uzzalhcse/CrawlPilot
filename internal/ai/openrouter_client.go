package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// OpenRouterClient handles OpenRouter API requests
type OpenRouterClient struct {
	keyManager *AIKeyManager
	model      string
	logger     *zap.Logger
	httpClient *http.Client
}

// NewOpenRouterClient creates a new OpenRouter client
func NewOpenRouterClient(keyManager *AIKeyManager, model string, logger *zap.Logger) (*OpenRouterClient, error) {
	if model == "" {
		model = "meta-llama/llama-3.1-8b-instruct:free" // Free Llama 3.1 8B
	}

	logger.Info("OpenRouter client initialized with key rotation", zap.String("model", model))

	return &OpenRouterClient{
		keyManager: keyManager,
		model:      model,
		logger:     logger,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}, nil
}

// OpenRouterRequest represents the API request
type OpenRouterRequest struct {
	Model    string              `json:"model"`
	Messages []OpenRouterMessage `json:"messages"`
}

type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouterResponse represents the API response
type OpenRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

// GenerateText sends a text prompt to OpenRouter
func (o *OpenRouterClient) GenerateText(ctx context.Context, prompt string) (string, error) {
	var lastErr error

	maxAttempts := 100
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Get API key from manager
		apiKey, err := o.keyManager.GetAPIKey(ctx, "openrouter")
		if err != nil {
			if attempt == 0 {
				return "", fmt.Errorf("no OpenRouter API keys available: %w", err)
			}
			return "", fmt.Errorf("failed after %d attempts, no more keys available: %w", attempt, lastErr)
		}

		request := OpenRouterRequest{
			Model: o.model,
			Messages: []OpenRouterMessage{
				{Role: "user", Content: prompt},
			},
		}

		jsonData, err := json.Marshal(request)
		if err != nil {
			return "", fmt.Errorf("failed to marshal request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("HTTP-Referer", "https://crawlify.app")
		req.Header.Set("X-Title", "Crawlify Health Check AI")

		o.logger.Debug("Sending prompt to OpenRouter",
			zap.String("model", o.model),
			zap.Int("prompt_length", len(prompt)),
			zap.String("key_id", o.keyManager.GetCurrentKeyID()),
			zap.Int("attempt", attempt+1))

		resp, err := o.httpClient.Do(req)
		if err != nil {
			lastErr = err
			o.logger.Warn("OpenRouter API call failed",
				zap.Error(err),
				zap.String("key_id", o.keyManager.GetCurrentKeyID()),
				zap.Int("attempt", attempt+1))

			o.keyManager.RecordFailure(ctx, err)

			if attempt < maxAttempts-1 {
				delayMs := 1000 + (attempt % 5000)
				time.Sleep(time.Duration(delayMs) * time.Millisecond)
			}
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = err
			o.keyManager.RecordFailure(ctx, err)
			continue
		}

		// Check for HTTP errors
		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("OpenRouter API error (status %d): %s", resp.StatusCode, string(body))

			if resp.StatusCode == 429 || resp.StatusCode == 402 || resp.StatusCode == 403 {
				o.logger.Warn("OpenRouter key rate limited or invalid",
					zap.Int("status_code", resp.StatusCode),
					zap.String("key_id", o.keyManager.GetCurrentKeyID()))
			}

			o.keyManager.RecordFailure(ctx, lastErr)

			if attempt < maxAttempts-1 {
				delayMs := 1000 + (attempt % 5000)
				time.Sleep(time.Duration(delayMs) * time.Millisecond)
			}
			continue
		}

		var apiResp OpenRouterResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			lastErr = fmt.Errorf("failed to parse response: %w", err)
			o.keyManager.RecordFailure(ctx, lastErr)
			continue
		}

		if apiResp.Error != nil {
			lastErr = fmt.Errorf("OpenRouter API error: %s", apiResp.Error.Message)
			o.keyManager.RecordFailure(ctx, lastErr)

			if attempt < maxAttempts-1 {
				delayMs := 1000 + (attempt % 5000)
				time.Sleep(time.Duration(delayMs) * time.Millisecond)
			}
			continue
		}

		if len(apiResp.Choices) == 0 {
			lastErr = fmt.Errorf("no response from OpenRouter")
			o.keyManager.RecordFailure(ctx, lastErr)
			continue
		}

		result := strings.TrimSpace(apiResp.Choices[0].Message.Content)

		// Record success
		o.keyManager.RecordSuccess(ctx)

		o.logger.Info("OpenRouter API request successful",
			zap.Int("response_length", len(result)),
			zap.String("key_id", o.keyManager.GetCurrentKeyID()),
			zap.Int("attempts_used", attempt+1))

		return result, nil
	}

	return "", fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}

// Close is a no-op
func (o *OpenRouterClient) Close() error {
	return nil
}
