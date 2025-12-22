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

// newDemoEmbedFn 为演示提供一个简单的占位向量化函数。
// 真正接入时请替换为你自己的 Embedding 服务（如 Ark Embedding）。
func newDemoEmbedFn() tools.EmbedFunc {
	// 默认返回一个固定维度的伪向量（适合 Demo / 空跑）。
	// 真实场景请用模型输出的向量且与 Qdrant collection 的维度匹配。
	const dim = 8
	return func(_ context.Context, text string) ([]float32, error) {
		vec := make([]float32, dim)
		// 简单哈希，把字符转成数值，便于演示（非真实 embedding）。
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

	// 关键词检索 Tool 作为 fallback（不强制绑定给模型）
	keywordTool := tools.NewNovelSearchTool()

	// 可选：绑定向量检索 Tool（需要提供 embedFn 并确保 Qdrant 可用）
	var vecTool einotool.InvokableTool
	if embedFn := newDemoEmbedFn(); embedFn != nil {
		cm, vecTool = tools.BindNovelVectorSearchTool(ctx, cm, embedFn)
		log.Printf("已绑定 NovelVectorSearch Tool（Qdrant）\n\n")
	} else {
		log.Printf("未绑定 NovelVectorSearch Tool：embedFn 未配置\n\n")
	}

	graph.RunInteractiveLoop(ctx, cm, vecTool, keywordTool)
}
