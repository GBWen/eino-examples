package service

import (
	"context"
	"log"

	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/tomato_novel_recommand/config"
	"github.com/cloudwego/eino-examples/tomato_novel_recommand/service/graph"
	"github.com/cloudwego/eino-examples/tomato_novel_recommand/service/llm"
	"github.com/cloudwego/eino-examples/tomato_novel_recommand/service/qdrant"
	tools "github.com/cloudwego/eino-examples/tomato_novel_recommand/service/tool"
	einotool "github.com/cloudwego/eino/components/tool"
)

// Run starts the interactive novel recommendation loop.
// It wires model, embedding, tools, and then enters the CLI loop.
func Run(ctx context.Context) {
	log.Printf("=== 番茄小说推荐 Demo ===")
	log.Printf("提示：请先在环境变量中设置 ARK_API_KEY\n")

	cfg := config.LoadConfig()

	log.Printf("=== 正在创建 Ark Chat 模型 ===\n")
	cm := llm.CreateArkChatModel(ctx, cfg.ArkChat)
	var err error

	// Embedding function for both vector sink (store) and vector/hybrid search.
	embedFn, arkErr := llm.NewArkEmbedFunc(cfg.ArkEmbed)
	if arkErr != nil {
		log.Fatalf("Ark embedding is required: %v", arkErr)
	}
	log.Printf("Ark embedding enabled via EMBED_API_KEY")
	qc := qdrant.NewWithConfig(cfg.Qdrant)
	vectorSink := qdrant.NewVectorSinkWithClient(embedFn, qc)

	// Prepare tools: keyword search (default) + clarify tool
	toolsList := []einotool.BaseTool{
		tools.NewNovelSearchToolWithSink(vectorSink), // Keyword search + persist to vector DB
		tools.NewNovelHybridSearchTool(embedFn, qc),  // Hybrid: API + vector merge
		tools.NewClarifyTool(),                       // Clarification tool: ask user for more details when needed
	}

	// Optional: keep pure vector search tool if you want the agent to choose it explicitly.
	if embedFn != nil {
		vecTool := tools.NewNovelVectorSearchTool(embedFn, qc)
		toolsList = append(toolsList, vecTool)
		log.Printf("Vector search tool enabled (semantic retrieval available)\n")
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
