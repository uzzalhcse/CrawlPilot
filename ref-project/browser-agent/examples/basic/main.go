package main

import (
	"context"
	"crawer-agent/exp/v2/pkg/agent"
	"os"

	"github.com/charmbracelet/log"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/joho/godotenv"
)

func main() {
	log.SetLevel(log.InfoLevel)

	if err := godotenv.Load(".env"); err != nil {
		log.Debug("No .env file found")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	ctx := context.Background()
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatal("Failed to create chat model:", err)
	}

	task := "Do a Google search to find who Elon Musk's wife is"
	ag := agent.NewAgent(task, model)

	historyResult, err := ag.Run(agent.WithMaxSteps(15))
	if err != nil {
		log.Fatal("Agent run failed:", err)
	}

	if historyResult != nil && historyResult.LastResult() != nil && historyResult.LastResult().ExtractedContent != nil {
		log.Infof("Agent output: %s", *historyResult.LastResult().ExtractedContent)
	} else {
		log.Info("Agent did not produce an extractable result")
	}
}
