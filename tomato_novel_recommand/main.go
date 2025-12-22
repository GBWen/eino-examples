/*
 * Tomato novel recommendation demo - simple chat loop entry.
 *
 * Usage (from repo root):
 *   ARK_API_KEY=xxx go run ./tomato_novel_recommand
 *
 * This will start an interactive CLI that asks for your reading preference
 * and returns a list of recommended Chinese web novels each round.
 */

package main

import (
	"context"
	"log"

	"github.com/cloudwego/eino-examples/tomato_novel_recommand/flow"
	"github.com/cloudwego/eino-examples/tomato_novel_recommand/graph"
	tools "github.com/cloudwego/eino-examples/tomato_novel_recommand/tool"
	einotool "github.com/cloudwego/eino/components/tool"
)

// newDemoEmbedFn provides a simple fake embedding function for demo purposes.
// In real scenarios, replace this with your own embedding service (e.g. Ark Embedding).
func newDemoEmbedFn() tools.EmbedFunc {
	// Returns a fixed-dimension pseudo vector (good enough for local demo).
	// In production, use real model outputs and ensure the dim matches Qdrant collection.
	// TODO: replace this demo embedding with a real embedding model (e.g. Ark Embedding API).
	const dim = 8
	return func(_ context.Context, text string) ([]float32, error) {
		vec := make([]float32, dim)
		// Simple hash-based projection to float values, only for demo (not a real embedding).
		for i, r := range []rune(text) {
			vec[i%dim] += float32(r%113) / 100.0
		}
		return vec, nil
	}
}

func main() {
	ctx := context.Background()

	log.Printf("=== 番茄小说推荐 Demo ===")
	log.Printf("提示：请先在环境变量中设置 ARK_API_KEY\n")

	log.Printf("=== 正在创建 Ark Chat 模型 ===\n")
	cm := flow.CreateArkChatModel(ctx)

	// Keyword-search tool as a fallback (not necessarily bound to the model).
	keywordTool := tools.NewNovelSearchTool()

	// Optionally bind vector-search tool (requires embedFn and a running Qdrant).
	var vecTool einotool.InvokableTool
	if embedFn := newDemoEmbedFn(); embedFn != nil {
		cm, vecTool = tools.BindNovelVectorSearchTool(ctx, cm, embedFn)
		log.Printf("NovelVectorSearch tool (Qdrant) is bound\n\n")
	} else {
		log.Printf("NovelVectorSearch tool is NOT bound: embedFn is nil\n\n")
	}

	graph.RunInteractiveLoop(ctx, cm, vecTool, keywordTool)
}
