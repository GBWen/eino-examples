/*
 * Tomato novel recommendation demo - Ark chat model initialization.
 */

package main

import (
	"context"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
)

// createArkChatModel creates an Ark chat model.
func createArkChatModel(ctx context.Context) model.ToolCallingChatModel {
	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		log.Fatalf("ARK_API_KEY is not set")
	}

	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		// use default public Ark inference endpoint and region, change if needed
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


