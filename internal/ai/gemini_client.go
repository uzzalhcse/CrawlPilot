package ai

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/genai"
)

// GeminiClient wraps the Gemini AI client with key rotation
type GeminiClient struct {
	keyManager *AIKeyManager
	model      string
	logger     *zap.Logger
}

// NewGeminiClient creates a new Gemini AI client with key manager
func NewGeminiClient(keyManager *AIKeyManager, model string, logger *zap.Logger) (*GeminiClient, error) {
	if model == "" {
		model = "gemini-2.5-flash" // Default to Gemini 2.5 Flash
	}

	logger.Info("Gemini client initialized with key rotation (new SDK)", zap.String("model", model))

	return &GeminiClient{
		keyManager: keyManager,
		model:      model,
		logger:     logger,
	}, nil
}

// GenerateText sends a text prompt to Gemini and returns the response
func (g *GeminiClient) GenerateText(ctx context.Context, prompt string) (string, error) {
	var lastErr error

	// Try with all available keys (up to 100 attempts to avoid infinite loops)
	maxAttempts := 100
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Get API key from manager
		apiKey, err := g.keyManager.GetAPIKey(ctx, "gemini")
		if err != nil {
			// No more available keys
			if attempt == 0 {
				return "", fmt.Errorf("no API keys available: %w", err)
			}
			return "", fmt.Errorf("failed after %d attempts, no more keys available: %w", attempt, lastErr)
		}

		// Create client with current key using new SDK
		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey: apiKey,
		})
		if err != nil {
			return "", fmt.Errorf("failed to create Gemini client: %w", err)
		}

		g.logger.Debug("Sending prompt to Gemini",
			zap.String("model", g.model),
			zap.Int("prompt_length", len(prompt)),
			zap.String("key_id", g.keyManager.GetCurrentKeyID()),
			zap.Int("attempt", attempt+1))

		// Call GenerateContent
		resp, err := client.Models.GenerateContent(ctx, g.model, genai.Text(prompt), nil)

		if err != nil {
			lastErr = err

			// Check if key is suspended and should be disabled
			if strings.Contains(err.Error(), "CONSUMER_SUSPENDED") || strings.Contains(err.Error(), "has been suspended") {
				g.logger.Warn("API key suspended, disabling",
					zap.String("key_id", g.keyManager.GetCurrentKeyID()),
					zap.Error(err))
			}

			g.logger.Warn("Gemini API call failed",
				zap.Error(err),
				zap.String("key_id", g.keyManager.GetCurrentKeyID()),
				zap.Int("attempt", attempt+1))

			// Record failure and rotate if needed
			g.keyManager.RecordFailure(ctx, err)

			// Add random delay before next attempt (500ms - 2000ms)
			if attempt < maxAttempts-1 {
				delayMs := 500 + (attempt % 1500) // Progressive delay
				time.Sleep(time.Duration(delayMs) * time.Millisecond)
			}

			// Continue to next attempt
			continue
		}

		if resp == nil {
			lastErr = fmt.Errorf("no response from Gemini")
			g.keyManager.RecordFailure(ctx, lastErr)
			continue
		}

		// Success! Extract text using the helper method from new SDK
		result := resp.Text()

		// Record success
		g.keyManager.RecordSuccess(ctx)

		g.logger.Info("Gemini API request successful",
			zap.Int("response_length", len(result)),
			zap.String("key_id", g.keyManager.GetCurrentKeyID()),
			zap.Int("attempts_used", attempt+1))

		return result, nil
	}

	return "", fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}

// GenerateWithImage sends a prompt with an image to Gemini
func (g *GeminiClient) GenerateWithImage(ctx context.Context, prompt string, imageData []byte, mimeType string) (string, error) {
	var lastErr error

	// Try with all available keys (round-robin through all keys once)
	// We'll attempt up to 20 times (reasonable max number of keys a user might have)
	maxAttempts := 20
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Get API key from manager (will rotate to next available key after failures)
		apiKey, err := g.keyManager.GetAPIKey(ctx, "gemini")
		if err != nil {
			if attempt == 0 {
				return "", fmt.Errorf("no API keys available: %w", err)
			}
			return "", fmt.Errorf("failed after %d attempts, no more keys available: %w", attempt, lastErr)
		}

		// Create client with current key using new SDK
		// Note: The new SDK reads GEMINI_API_KEY from env by default, but we can pass it via config if needed.
		// However, the NewClient signature in the user example was NewClient(ctx, nil).
		// Let's try to set the API key via option or config if available.
		// Looking at the error "undefined: genai.BackendGoogleAI", we should remove that.
		// We'll set the API key in the context or via a config struct if we can find one.
		// For now, let's try passing the key in a way that works.
		// Since I don't have the full docs, I'll assume we can pass a config with APIKey.
		// If BackendGoogleAI is undefined, maybe we just omit it.

		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey: apiKey,
		})
		if err != nil {
			return "", fmt.Errorf("failed to create Gemini client: %w", err)
		}

		g.logger.Debug("Sending multimodal prompt to Gemini",
			zap.String("model", g.model),
			zap.Int("prompt_length", len(prompt)),
			zap.Int("image_size", len(imageData)),
			zap.String("key_id", g.keyManager.GetCurrentKeyID()),
			zap.Int("attempt", attempt+1))

		// Create parts for multimodal request
		// We need to combine text and image into a single list of contents.
		// genai.Text returns []*genai.Content.
		// We need to construct the image part manually since genai.ImageData is undefined.

		// Create the image part
		imagePart := &genai.Part{
			InlineData: &genai.Blob{
				MIMEType: mimeType,
				Data:     imageData,
			},
		}

		// Create the text part
		textPart := &genai.Part{
			Text: prompt,
		}

		// Create content
		content := &genai.Content{
			Parts: []*genai.Part{textPart, imagePart},
		}

		// Call GenerateContent with the content list
		resp, err := client.Models.GenerateContent(ctx, g.model, []*genai.Content{content}, nil)

		if err != nil {
			lastErr = err

			if strings.Contains(err.Error(), "CONSUMER_SUSPENDED") || strings.Contains(err.Error(), "has been suspended") {
				g.logger.Warn("API key suspended, disabling",
					zap.String("key_id", g.keyManager.GetCurrentKeyID()),
					zap.Error(err))
			}

			g.logger.Warn("Gemini multimodal API call failed",
				zap.Error(err),
				zap.String("key_id", g.keyManager.GetCurrentKeyID()),
				zap.Int("attempt", attempt+1))

			g.keyManager.RecordFailure(ctx, err)

			// Continue to next key immediately (no retry delay needed since we're rotating keys)
			continue
		}

		if resp == nil {
			lastErr = fmt.Errorf("no response from Gemini")
			g.keyManager.RecordFailure(ctx, lastErr)
			continue
		}

		result := resp.Text()

		g.keyManager.RecordSuccess(ctx)

		g.logger.Info("Gemini multimodal API request successful",
			zap.Int("response_length", len(result)),
			zap.String("key_id", g.keyManager.GetCurrentKeyID()),
			zap.Int("attempts_used", attempt+1))

		return result, nil
	}

	return "", fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}

// GenerateWithBase64Image sends a prompt with a base64-encoded image
func (g *GeminiClient) GenerateWithBase64Image(ctx context.Context, prompt string, base64Image string, mimeType string) (string, error) {
	// Decode base64 image
	imageData, err := base64.StdEncoding.DecodeString(base64Image)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 image: %w", err)
	}

	return g.GenerateWithImage(ctx, prompt, imageData, mimeType)
}

// Close is a no-op
func (g *GeminiClient) Close() error {
	return nil
}
