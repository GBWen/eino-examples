package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

// EmbedFunc 抽象化的文本向量化函数，调用方可接入任意 Embedding 服务。
type EmbedFunc func(ctx context.Context, text string) ([]float32, error)

// qdrantClient 负责调用 Qdrant 的 search 接口。
type qdrantClient struct {
	baseURL    string
	collection string
	httpClient *http.Client
}

func newQdrantClientFromEnv() *qdrantClient {
	base := os.Getenv("QDRANT_URL")
	if base == "" {
		base = "http://localhost:6333"
	}
	coll := os.Getenv("QDRANT_COLLECTION")
	if coll == "" {
		coll = "novels"
	}
	return &qdrantClient{
		baseURL:    base,
		collection: coll,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// qdrantSearchRequest 封装 Qdrant search 请求。
type qdrantSearchRequest struct {
	Vector []float32   `json:"vector"`
	Limit  int         `json:"limit"`
	Filter interface{} `json:"filter,omitempty"`
}

// qdrantSearchResponse 简化的响应结构。
type qdrantSearchResponse struct {
	Result []struct {
		ID      any                    `json:"id"`
		Score   float64                `json:"score"`
		Payload map[string]interface{} `json:"payload"`
	} `json:"result"`
	Status string `json:"status"`
	Time   int64  `json:"time"`
}

func (c *qdrantClient) search(ctx context.Context, vec []float32, topN int, genre string) ([]*Novel, error) {
	reqBody := qdrantSearchRequest{
		Vector: vec,
		Limit:  topN,
	}
	if genre != "" {
		reqBody.Filter = map[string]any{
			"must": []map[string]any{
				{
					"key":   "genre",
					"match": map[string]any{"value": genre},
				},
			},
		}
	}

	b, _ := json.Marshal(reqBody)
	url := fmt.Sprintf("%s/collections/%s/points/search", c.baseURL, c.collection)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("qdrant search status %d", resp.StatusCode)
	}

	var r qdrantSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}

	var novels []*Novel
	for _, item := range r.Result {
		payload := item.Payload
		novels = append(novels, &Novel{
			Title:       strFromPayload(payload, "title"),
			Author:      strFromPayload(payload, "author"),
			Category:    strFromPayload(payload, "genre"),
			Description: strFromPayload(payload, "description"),
			Link:        strFromPayload(payload, "link"),
		})
	}
	return novels, nil
}

func strFromPayload(p map[string]interface{}, key string) string {
	if v, ok := p[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// NovelVectorSearchInput 向量检索 Tool 的入参，兼容 keyword/genre/top_n。
type NovelVectorSearchInput struct {
	Keyword string `json:"keyword" jsonschema_description:"搜索关键词，如 甜宠/复仇/系统 等"`
	Genre   string `json:"genre" jsonschema_description:"可选的题材过滤，如 玄幻/都市/言情 等"`
	TopN    int    `json:"top_n" jsonschema_description:"返回前 N 本，默认 5 本"`
	// 兼容字段：如果调用方仍传 query，将优先使用 keyword 非空，否则 fallback 到 query。
	Query string `json:"query,omitempty" jsonschema_description:"兼容字段，同 keyword"`
}

// NewNovelVectorSearchTool 使用 Qdrant + EmbedFunc 的向量检索 Tool。
// embedFn：调用方提供的向量化函数；如果为空则返回错误提示。
func NewNovelVectorSearchTool(embedFn EmbedFunc) einotool.InvokableTool {
	qc := newQdrantClientFromEnv()

	toolImpl, err := utils.InferTool(
		"novel_vector_search",
		"基于向量检索的番茄小说搜索（Qdrant），支持 keyword/genre/top_n",
		func(ctx context.Context, input *NovelVectorSearchInput) (output *NovelSearchOutput, err error) {
			if embedFn == nil {
				return nil, fmt.Errorf("embedFn is nil: 请在创建 Tool 时传入向量化实现")
			}
			topN := input.TopN
			if topN == 0 {
				topN = 5
			}
			query := input.Keyword
			if query == "" {
				query = input.Query
			}
			vec, err := embedFn(ctx, query)
			if err != nil {
				return nil, fmt.Errorf("embed failed: %w", err)
			}
			novels, err := qc.search(ctx, vec, topN, input.Genre)
			if err != nil {
				// fallback: 返回空列表，让上层可决定是否继续走关键词检索
				return &NovelSearchOutput{Novels: []*Novel{}}, fmt.Errorf("qdrant search failed: %w", err)
			}
			return &NovelSearchOutput{Novels: novels}, nil
		},
	)
	if err != nil {
		log.Fatalf("failed to create novel_vector_search tool: %v", err)
	}
	return toolImpl
}

// BindNovelVectorSearchTool 将向量检索 Tool 绑定到模型。
func BindNovelVectorSearchTool(ctx context.Context, cm model.ToolCallingChatModel, embedFn EmbedFunc) (model.ToolCallingChatModel, einotool.InvokableTool) {
	vectorTool := NewNovelVectorSearchTool(embedFn)

	info, err := vectorTool.Info(ctx)
	if err != nil {
		log.Fatalf("get vector tool info failed: %v", err)
	}
	newCM, err := cm.WithTools([]*schema.ToolInfo{info})
	if err != nil {
		log.Fatalf("bind vector tool failed: %v", err)
	}
	return newCM, vectorTool
}
