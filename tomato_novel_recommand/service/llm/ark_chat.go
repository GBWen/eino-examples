package llm

import (
	"context"
	"log"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"

	"github.com/cloudwego/eino-examples/tomato_novel_recommand/config"
)

// CreateArkChatModel creates an Ark chat model with env config.
func CreateArkChatModel(ctx context.Context) model.ToolCallingChatModel {
	cfg := config.LoadArkChatConfig()
	if cfg.APIKey == "" {
		log.Fatalf("ARK_API_KEY is not set")
	}

	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		BaseURL: cfg.Base,
		Region:  cfg.Region,
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
	})
	if err != nil {
		log.Fatalf("create ark chat model failed: %v", err)
	}
	return chatModel
}
