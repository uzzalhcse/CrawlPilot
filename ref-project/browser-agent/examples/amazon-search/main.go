package main

import (
	"context"
	"crawer-agent/exp/v2/pkg/agent"
	"os"

	"github.com/charmbracelet/log"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/nerdface-ai/browser-use-go/pkg/dotenv"
)

func main() {
	log.SetLevel(log.DebugLevel)

	dotenv.LoadEnv(".env")
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set GEMINI_API_KEY environment variable.")
	}

	modelName := "qwen3:latest" // Ollama model name
	ctx := context.Background()

	qwenBaseURL := os.Getenv("QWEN_BASE_URL")
	if qwenBaseURL == "" {
		log.Fatal("QWEN_BASE_URL not set. Please set it in your .env file")
	}
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   modelName,
		APIKey:  "sk-dummy-key",
		BaseURL: qwenBaseURL, // e.g., https://allegedly-hopeful-stallion.ngrok-free.app/v1
	})
	if err != nil {
		log.Fatalf("Failed to create Gemini chat model: %v", err)
	}

	// You can customize the search keyword
	searchKeyword := os.Getenv("SEARCH_KEYWORD")
	if searchKeyword == "" {
		searchKeyword = "wireless headphones"
	}

	task := `Go to amazon.com and search for "` + searchKeyword + `". 
Then click on the first product in the search results.
Extract and report:
1. Product name
2. Price
3. Rating (if available)
4. Number of reviews (if available)`

	log.Infof("Starting Amazon product search for: %s", searchKeyword)

	ag := agent.NewAgent(
		task,
		model,
		agent.WithAgentSettings(agent.AgentSettingsConfig{
			"use_vision":   false,
			"max_failures": 5,
		}),
	)

	historyResult, err := ag.Run(
		agent.WithMaxSteps(20),
	)

	if err != nil {
		log.Fatalf("Agent execution failed: %v", err)
	}

	if historyResult != nil && historyResult.LastResult() != nil {
		if historyResult.LastResult().ExtractedContent != nil {
			log.Infof("✅ Product Details:\n%s", *historyResult.LastResult().ExtractedContent)
		} else {
			log.Warn("Agent completed but no content was extracted")
		}
	} else {
		log.Warn("Agent completed but no history result available")
	}
}
