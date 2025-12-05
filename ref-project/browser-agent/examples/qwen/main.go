/*
 * Example: Qwen 3 with Browser Automation (FREE via Ollama)
 *
 * This example demonstrates using Qwen 3 running on Google Colab
 * via Ollama for browser automation tasks.
 *
 * ✅ Tool/function calling support confirmed
 * ✅ Browser automation ready
 * ✅ FREE - runs on Colab
 */

package main

import (
	"context"
	"crawer-agent/exp/v2/pkg/agent"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/nerdface-ai/browser-use-go/pkg/dotenv"

	"github.com/charmbracelet/log"
)

func main() {
	// Set log level to debug for more detailed output
	log.SetLevel(log.DebugLevel)

	dotenv.LoadEnv(".env")

	ctx := context.Background()

	log.Info("🔄 Initializing Qwen 3 via Ollama for Browser Automation (FREE!)")

	// Get Qwen configuration from environment
	qwenBaseURL := os.Getenv("QWEN_BASE_URL")
	if qwenBaseURL == "" {
		log.Fatal("QWEN_BASE_URL not set. Please set it in your .env file")
	}

	// Create Qwen 3 model via Ollama
	modelName := "qwen3:latest" // Ollama model name
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   modelName,
		APIKey:  "sk-dummy-key",
		BaseURL: qwenBaseURL, // e.g., https://allegedly-hopeful-stallion.ngrok-free.app/v1
	})
	if err != nil {
		log.Fatalf("Failed to create Qwen model: %v", err)
	}

	log.Infof("✅ Using Qwen 3 via Ollama")
	log.Infof("   Endpoint: %s", qwenBaseURL)
	log.Infof("   Model: %s", modelName)
	log.Info("   🆓 FREE via Google Colab!")

	// Define the browser automation task
	task := "Go to https://merrell.jp then hover over the men's category then click a cheap product and extract the product name, price,Model number, and description."

	log.Infof("Starting browser automation task: %s", task)

	// Create agent with the Qwen model
	ag := agent.NewAgent(
		task,
		model,
	)

	// Run the agent
	result, err := ag.Run()
	if err != nil {
		log.Fatalf("Agent execution failed: %v", err)
	}

	log.Infof("✅ Task completed successfully!")
	log.Infof("Result: %v", result)

	log.Info("\n💰 Cost: $0 (FREE!) vs OpenAI: ~$0.05-0.10 per task")
	log.Info("🚀 Qwen 3 supports tool calling for browser automation!")
}
