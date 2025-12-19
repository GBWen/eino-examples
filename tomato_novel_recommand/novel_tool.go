/*
 * Tomato novel recommendation demo - NovelSearch Tool & binding helpers.
 *
 * 本文件演示如何把“番茄小说搜索”封装成 Eino Tool，
 * 让 Ark ChatModel 可以在 ReAct / ToolCall 场景下自动调用搜索能力。
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
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

// NovelSearchParam 是 NovelSearch Tool 接收的参数结构。
type NovelSearchParam struct {
	Keyword string `json:"keyword"` // 关键字：如“重生、甜宠、系统”
	Genre   string `json:"genre"`   // 类型：如“玄幻、都市、言情”
	TopN    int    `json:"top_n"`   // 返回前 N 本，默认 5
}

// NovelSearchTool 将“番茄小说搜索能力”封装为一个 InvokableTool。
type NovelSearchTool struct{}

// NewNovelSearchTool 创建一个 NovelSearch Tool 实例。
func NewNovelSearchTool() *NovelSearchTool {
	return &NovelSearchTool{}
}

// Info 返回 Tool 的元信息，用于暴露给大模型做 tool calling。
func (t *NovelSearchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "novel_search",
		Desc: "根据用户给定的关键词或类型，从番茄小说中检索合适的小说列表",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"keyword": {
				Type:     "string",
				Desc:     "搜索关键词，比如：重生、甜宠、复仇、系统 等",
				Required: false,
			},
			"genre": {
				Type:     "string",
				Desc:     "小说类型/题材，比如：玄幻、都市、言情、悬疑等",
				Required: false,
			},
			"top_n": {
				Type: "number",
				Desc: "返回前 N 本命中的小说，默认 5 本",
			},
		}),
	}, nil
}

// InvokableRun 是 Tool 的真正执行逻辑。
// - argumentsInJSON：来自大模型 Tool Call 的 JSON 字符串参数
// - 返回值：给大模型的字符串，一般设计成结构化 JSON，字段含义要清晰
func (t *NovelSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 1. 解析参数
	var p NovelSearchParam
	if err := json.Unmarshal([]byte(argumentsInJSON), &p); err != nil {
		return "", fmt.Errorf("unmarshal novel search param failed: %w", err)
	}
	if p.TopN == 0 {
		p.TopN = 5
	}

	// 2. 调用实际的番茄小说后端（此处使用 mock，方便本仓库直接运行）
	novels, err := callTomatoAPI(ctx, p)
	if err != nil {
		return "", fmt.Errorf("call tomato api failed: %w", err)
	}

	// 3. 序列化为 JSON 字符串返回给大模型
	out, err := json.Marshal(novels)
	if err != nil {
		return "", fmt.Errorf("marshal novels failed: %w", err)
	}
	return string(out), nil
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
