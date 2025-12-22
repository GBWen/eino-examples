package flow

import (
	"context"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
)

// CreateArkChatModel creates an Ark chat model with env config.
func CreateArkChatModel(ctx context.Context) model.ToolCallingChatModel {
	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		log.Fatalf("ARK_API_KEY is not set")
	}

	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		BaseURL: "https://ark.cn-beijing.volces.com/api/v3",
		Region:  "cn-beijing",
		APIKey:  apiKey,
		Model:   "doubao-seed-1-6-251015",
	})
	if err != nil {
		log.Fatalf("create ark chat model failed: %v", err)
	}
	return chatModel
}
