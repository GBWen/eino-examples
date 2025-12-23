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

	"github.com/cloudwego/eino/schema"

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
	var err error

	// Embedding function for both vector sink (store) and vector/hybrid search.
	embedFn, arkErr := tools.NewArkEmbedFuncFromEnv()
	if arkErr != nil {
		log.Fatalf("Ark embedding is required: %v", arkErr)
	}
	log.Printf("Ark embedding enabled via EMBED_API_KEY")
	vectorSink := tools.NewQdrantVectorSink(embedFn)

	// Prepare tools: keyword search (default) + clarify tool
	toolsList := []einotool.BaseTool{
		tools.NewNovelSearchToolWithSink(vectorSink), // Keyword search + persist to vector DB
		tools.NewNovelHybridSearchTool(embedFn),      // Hybrid: API + vector merge
		tools.NewClarifyTool(),                       // Clarification tool: ask user for more details when needed
	}

	// Optional: keep pure vector search tool if you want the agent to choose it explicitly.
	if embedFn != nil {
		vecTool := tools.NewNovelVectorSearchTool(embedFn)
		toolsList = append(toolsList, vecTool)
		log.Printf("Vector search tool enabled (for large-scale book library)\n")
	}

	// Bind all tools to the model
	// It will automatically decide which tool to use
	var toolInfos []*schema.ToolInfo
	for _, t := range toolsList {
		info, err := t.Info(ctx)
		if err != nil {
			log.Fatalf("get tool info failed: %v", err)
		}
		toolInfos = append(toolInfos, info)
	}
	cm, err = cm.WithTools(toolInfos)
	if err != nil {
		log.Fatalf("bind tools failed: %v", err)
	}

	log.Printf("Tools bound: keyword search + clarify (vector search disabled by default)\n\n")

	graph.RunInteractiveLoop(ctx, cm, toolsList)
}
