/*
 * Tomato novel recommendation demo - NovelSearch Tool & binding helpers.
 *
 * 本文件演示如何把"番茄小说搜索"封装成 Eino Tool，
 * 让 Ark ChatModel 可以在 ReAct / ToolCall 场景下自动调用搜索能力。
 *
 * 参考了 adk/intro/chatmodel 中的 booksearch.go 实现方式，
 * 使用 utils.InferTool 简化 Tool 创建，通过结构体标签定义参数。
 */

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

// Novel 表示单本小说的关键信息。
type Novel struct {
	Title       string `json:"title"`
	Author      string `json:"author"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Link        string `json:"link"`
}

// NovelSearchInput 是 NovelSearch Tool 接收的参数结构。
// 使用 jsonschema_description 标签定义参数描述，enum 标签定义可选值。
type NovelSearchInput struct {
	Keyword string `json:"keyword" jsonschema_description:"搜索关键词，比如：重生、甜宠、复仇、系统等"`
	Genre   string `json:"genre" jsonschema_description:"小说类型/题材，比如：玄幻、都市、言情、悬疑等" enum:"玄幻,enum:都市,enum:言情,enum:悬疑,enum:科幻,enum:历史,enum:军事"`
	TopN    int    `json:"top_n" jsonschema_description:"返回前 N 本命中的小说，默认 5 本"`
}

// NovelSearchOutput 是 NovelSearch Tool 的返回结果结构。
type NovelSearchOutput struct {
	Novels []*Novel `json:"novels"`
}

// NewNovelSearchTool 创建一个 NovelSearch Tool 实例。
// 使用 utils.InferTool 简化 Tool 创建，自动从函数签名和结构体标签推断 Tool 元信息。
func NewNovelSearchTool() tool.InvokableTool {
	novelSearchTool, err := utils.InferTool(
		"novel_search",
		"根据用户给定的关键词或类型，从番茄小说中检索合适的小说列表",
		func(ctx context.Context, input *NovelSearchInput) (output *NovelSearchOutput, err error) {
			// 设置默认值
			if input.TopN == 0 {
				input.TopN = 5
			}

			// 调用实际的番茄小说后端（此处使用 mock，方便本仓库直接运行）
			novels, err := callTomatoAPI(ctx, NovelSearchParam{
				Keyword: input.Keyword,
				Genre:   input.Genre,
				TopN:    input.TopN,
			})
			if err != nil {
				return nil, fmt.Errorf("call tomato api failed: %w", err)
			}

			return &NovelSearchOutput{Novels: novels}, nil
		},
	)
	if err != nil {
		log.Fatalf("failed to create novel search tool: %v", err)
	}
	return novelSearchTool
}

// NovelSearchParam 用于内部 API 调用的参数结构（保持向后兼容）。
type NovelSearchParam struct {
	Keyword string
	Genre   string
	TopN    int
}

// callTomatoAPI 是对番茄小说检索服务的封装。
// 实际接入时请替换为真实 HTTP / RPC 调用，这里仅做演示。
func callTomatoAPI(_ context.Context, p NovelSearchParam) ([]*Novel, error) {
	// Demo：返回一些 mock 数据，字段语义清晰便于大模型理解。
	mock := []*Novel{
		{
			Title:       "重生成顶级爽文女主",
			Author:      "番茄·示例",
			Category:    coalesce(p.Genre, "都市言情"),
			Description: "轻松爽文向，女主重生后一路逆袭，适合想放松、看爽点的读者。",
			Link:        "https://tomato.example.com/book/10001",
		},
		{
			Title:       "异世签到一千年",
			Author:      "番茄·种田",
			Category:    coalesce(p.Genre, "玄幻"),
			Description: "慢热种田流，主角靠系统稳扎稳打发育，适合喜欢节奏舒缓的读者。",
			Link:        "https://tomato.example.com/book/10002",
		},
	}
	if p.TopN < len(mock) && p.TopN > 0 {
		return mock[:p.TopN], nil
	}
	return mock, nil
}

// coalesce 返回第一个非空字符串。
func coalesce(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// bindNovelSearchTool 将 NovelSearch Tool 绑定到 Ark ChatModel 上，
// 返回带 Tool 能力的新模型实例和 Tool 本身，方便在 Graph / Agent 中继续使用。
func bindNovelSearchTool(ctx context.Context, cm model.ToolCallingChatModel) (model.ToolCallingChatModel, tool.BaseTool) {
	novelTool := NewNovelSearchTool()

	info, err := novelTool.Info(ctx)
	if err != nil {
		log.Fatalf("get novel tool info failed: %v", err)
	}

	newCM, err := cm.WithTools([]*schema.ToolInfo{info})
	if err != nil {
		log.Fatalf("bind tools failed: %v", err)
	}
	return newCM, novelTool
}
