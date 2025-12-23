package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"github.com/cloudwego/eino-examples/tomato_novel_recommand/config"
)

// EmbedFunc abstracts a text-embedding function.
// Caller can plug in any embedding service.
type EmbedFunc func(ctx context.Context, text string) ([]float32, error)

// qdrantClient wraps Qdrant search API.
type qdrantClient struct {
	baseURL    string
	collection string
	httpClient *http.Client
}

func newQdrantClientFromEnv() *qdrantClient {
	qc := config.LoadQdrantConfig()
	return &qdrantClient{
		baseURL:    qc.URL,
		collection: qc.Collection,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// qdrantSearchRequest is a minimal search request payload for Qdrant.
type qdrantSearchRequest struct {
	Vector []float32   `json:"vector"`
	Limit  int         `json:"limit"`
	Filter interface{} `json:"filter,omitempty"`
}

// qdrantSearchResponse is a simplified search response from Qdrant.
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
	log.Printf("[qdrant] vector search, collection=%s, topN=%d, genre=%q, url=%s", c.collection, topN, genre, url)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		log.Printf("[qdrant] build request failed, err=%v", err)
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		log.Printf("[qdrant] request failed, err=%v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("[qdrant] non-2xx status, code=%d", resp.StatusCode)
		return nil, fmt.Errorf("qdrant search status %d", resp.StatusCode)
	}

	var r qdrantSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		log.Printf("[qdrant] decode response failed, err=%v", err)
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
	log.Printf("[qdrant] search ok, got %d novels", len(novels))
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

// NovelVectorSearchInput is the input for the vector-search tool, compatible with keyword/genre/top_n.
type NovelVectorSearchInput struct {
	Keyword string `json:"keyword" jsonschema_description:"search keyword, e.g. sweet-pet / revenge / system"`
	Genre   string `json:"genre" jsonschema_description:"optional genre filter, e.g. fantasy / urban / romance"`
	TopN    int    `json:"top_n" jsonschema_description:"return top N results, default 5"`
	// Compatible field: if caller still uses 'query', we fallback to it when keyword is empty.
	Query string `json:"query,omitempty" jsonschema_description:"compatible field, same meaning as keyword"`
}

// NewNovelVectorSearchTool builds a Qdrant-based vector-search tool using the given EmbedFunc.
// embedFn must be provided by the caller; if nil, an error is returned.
func NewNovelVectorSearchTool(embedFn EmbedFunc) einotool.InvokableTool {
	// TODO: inject qdrantClient and EmbedFunc via DI/config to make this tool easier to test and configure.
	qc := newQdrantClientFromEnv()

	toolImpl, err := utils.InferTool(
		"novel_vector_search",
		"Vector-based novel search using Qdrant, supports keyword/genre/top_n.",
		func(ctx context.Context, input *NovelVectorSearchInput) (output *NovelSearchOutput, err error) {
			log.Printf("[tool] invoke novel_vector_search, input=%+v", input)
			if embedFn == nil {
				return nil, fmt.Errorf("embedFn is nil: please provide an embedding implementation when creating the tool")
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
				log.Printf("[tool] novel_vector_search embed failed, err=%v", err)
				return nil, fmt.Errorf("embed failed: %w", err)
			}
			novels, err := qc.search(ctx, vec, topN, input.Genre)
			if err != nil {
				// Fallback: return empty list and let upper layer decide whether to fall back to keyword search.
				log.Printf("[tool] novel_vector_search qdrant search failed, err=%v", err)
				return &NovelSearchOutput{Novels: []*Novel{}}, fmt.Errorf("qdrant search failed: %w", err)
			}
			out := &NovelSearchOutput{Novels: novels}
			log.Printf("[tool] novel_vector_search ok, got %d novels", len(out.Novels))
			return out, nil
		},
	)
	if err != nil {
		log.Fatalf("failed to create novel_vector_search tool: %v", err)
	}
	return toolImpl
}
