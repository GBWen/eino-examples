package tools

import (
	"context"
	"fmt"
	"log"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// Novel represents the key information of a single novel.
type Novel struct {
	Title       string `json:"title"`
	Author      string `json:"author"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Link        string `json:"link"`
}

// NovelSearchInput is the input schema of the NovelSearch tool.
// It uses jsonschema_description and enum tags for better tool documentation.
type NovelSearchInput struct {
	Keyword string `json:"keyword" jsonschema_description:"search keyword, e.g. rebirth / sweet-pet / revenge / system"`
	Genre   string `json:"genre" jsonschema_description:"novel genre, e.g. fantasy / urban / romance / mystery" enum:"fantasy,enum:urban,enum:romance,enum:mystery,enum:sci-fi,enum:history,enum:military"`
	TopN    int    `json:"top_n" jsonschema_description:"return top N results, default 5"`
}

// NovelSearchOutput is the output schema of the NovelSearch tool.
type NovelSearchOutput struct {
	Novels []*Novel `json:"novels"`
}

// NovelSearchParam is the internal parameter struct for API calls (kept for backwards compatibility).
type NovelSearchParam struct {
	Keyword string
	Genre   string
	TopN    int
}

// NewNovelSearchTool creates a NovelSearch tool.
// It uses utils.InferTool to infer tool metadata from function signature and struct tags.
func NewNovelSearchTool() einotool.InvokableTool {
	novelSearchTool, err := utils.InferTool(
		"novel_search",
		"Search tomato novels based on user-provided keyword or genre, and return suitable candidates.",
		func(ctx context.Context, input *NovelSearchInput) (output *NovelSearchOutput, err error) {
			if input.TopN == 0 {
				input.TopN = 5
			}

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

// callTomatoAPI wraps the tomato novel search backend.
// In real usage, replace this with real HTTP / RPC calls; this is only a demo mock.
// TODO: plug in real tomato novel search API (HTTP/RPC) and remove the hard-coded mock data below.
func callTomatoAPI(_ context.Context, p NovelSearchParam) ([]*Novel, error) {
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
