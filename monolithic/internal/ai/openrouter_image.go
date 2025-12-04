package ai

import "context"

// GenerateWithImage is not yet supported by OpenRouter free models (text-only fallback)
func (o *OpenRouterClient) GenerateWithImage(ctx context.Context, prompt string, imageData []byte, mimeType string) (string, error) {
	// OpenRouter free models don't support vision, fall back to text-only
	o.logger.Warn("Image analysis requested but not supported by OpenRouter free models, using text-only")
	return o.GenerateText(ctx, prompt)
}
