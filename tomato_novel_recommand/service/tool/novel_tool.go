package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/cloudwego/eino-examples/tomato_novel_recommand/utils"
	einotool "github.com/cloudwego/eino/components/tool"
	einoutils "github.com/cloudwego/eino/components/tool/utils"
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

// NovelResultSink can persist/search results into a backing store (e.g. Qdrant).
type NovelResultSink interface {
	StoreNovels(ctx context.Context, novels []*Novel) error
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
	return NewNovelSearchToolWithSink(nil)
}

// NewNovelSearchToolWithSink allows the caller to plug in a sink to persist every search result
// (e.g. write into vector DB for later semantic search).
func NewNovelSearchToolWithSink(sink NovelResultSink) einotool.InvokableTool {
	novelSearchTool, err := einoutils.InferTool(
		"novel_search",
		"Search novels based on user-provided keyword or genre, and return suitable candidates.",
		func(ctx context.Context, input *NovelSearchInput) (output *NovelSearchOutput, err error) {
			log.Printf("[tool] invoke novel_search, input=%+v", input)
			if input.TopN == 0 {
				input.TopN = 5
			}

			novels, err := callNovelAPI(ctx, NovelSearchParam{
				Keyword: input.Keyword,
				Genre:   input.Genre,
				TopN:    input.TopN,
			})
			if err != nil {
				log.Printf("[tool] novel_search failed, err=%v", err)
				return nil, fmt.Errorf("call novel api failed: %w", err)
			}

			if sink != nil && len(novels) > 0 {
				if err := sink.StoreNovels(ctx, novels); err != nil {
					// Non-blocking persistence; log and continue.
					log.Printf("[tool] persist novels to sink failed, err=%v", err)
				}
			}

			out := &NovelSearchOutput{Novels: novels}
			log.Printf("[tool] novel_search ok, got %d novels", len(out.Novels))
			return out, nil
		},
	)
	if err != nil {
		log.Fatalf("failed to create novel search tool: %v", err)
	}
	return novelSearchTool
}

// novelAPIClient wraps HTTP client for novel search API.
// Uses 小尘API (https://api.xcvts.cn/api/xiaoshuo/fanqie).
type novelAPIClient struct {
	httpClient *http.Client
}

const (
	// novelAPIBaseURL is the base URL for 小尘API novel search.
	novelAPIBaseURL = "https://api.xcvts.cn/api/xiaoshuo/fanqie"
)

// newNovelAPIClient creates a client for novel search API.
func newNovelAPIClient() *novelAPIClient {
	return &novelAPIClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// xcvtsAPIResponse is the response format from 小尘API.
type xcvtsAPIResponse struct {
	APISource string      `json:"api_source"`
	Data      interface{} `json:"data"` // Can be array or object
}

// search calls the novel search API (小尘API by default).
// API format: GET {baseURL}?q={keyword}
func (c *novelAPIClient) search(ctx context.Context, p NovelSearchParam) ([]*Novel, error) {
	// Build search query: use keyword, fallback to genre if keyword is empty
	query := p.Keyword
	if query == "" {
		query = p.Genre
	}
	if query == "" {
		return nil, fmt.Errorf("search query is required (keyword or genre)")
	}

	// Build URL with query parameter
	reqURL := fmt.Sprintf("%s?q=%s", novelAPIBaseURL, url.QueryEscape(query))
	log.Printf("[novel_api] searching novels, query=%q, genre=%q, topN=%d, url=%s", p.Keyword, p.Genre, p.TopN, reqURL)

	// Create HTTP GET request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[novel_api] request failed, err=%v", err)
		return nil, fmt.Errorf("api call failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("[novel_api] non-200 status, code=%d", resp.StatusCode)
		return nil, fmt.Errorf("api returned status %d", resp.StatusCode)
	}

	// Parse response
	var apiResp xcvtsAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		log.Printf("[novel_api] decode response failed, err=%v", err)
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	// Check if data is an array (search results) or object (error/info)
	novels, err := c.parseResponseData(apiResp.Data, p)
	if err != nil {
		log.Printf("[novel_api] parse response data failed, err=%v", err)
		return nil, fmt.Errorf("parse response data failed: %w", err)
	}

	// Limit results to TopN
	if p.TopN > 0 && len(novels) > p.TopN {
		novels = novels[:p.TopN]
	}

	log.Printf("[novel_api] search ok, got %d novels (after topN limit)", len(novels))
	return novels, nil
}

// parseResponseData parses the data field from API response.
// It can be an array of novels or an object (error/info message).
func (c *novelAPIClient) parseResponseData(data interface{}, p NovelSearchParam) ([]*Novel, error) {
	// Try to parse as array of novels
	if arr, ok := data.([]interface{}); ok {
		var novels []*Novel
		for _, item := range arr {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			// Parse individual novel item
			novel := &Novel{
				Title:       getString(itemMap, "title"),
				Author:      getString(itemMap, "author"),
				Category:    utils.Coalesce(p.Genre, "未知"), // API doesn't provide category, use genre param or default
				Description: getString(itemMap, "abstract"),
			}

			// Build link from book_id
			bookID := getString(itemMap, "book_id")
			if bookID != "" {
				novel.Link = fmt.Sprintf("https://fanqienovel.com/page/%s", bookID)
			} else {
				novel.Link = ""
			}

			novels = append(novels, novel)
		}
		return novels, nil
	}

	// If data is an object, it might be an error or info message
	return nil, fmt.Errorf("unexpected response format: data is not an array")
}

// getString safely extracts string value from map.
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

var (
	// Global API client instance (lazy initialized)
	apiClient *novelAPIClient
)

// callNovelAPI calls the novel search API.
func callNovelAPI(ctx context.Context, p NovelSearchParam) ([]*Novel, error) {
	if apiClient == nil {
		apiClient = newNovelAPIClient()
	}
	return apiClient.search(ctx, p)
}
