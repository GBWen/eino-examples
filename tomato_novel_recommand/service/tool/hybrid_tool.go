package tools

import (
	"context"
	"fmt"
	"log"
	"strings"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"github.com/cloudwego/eino-examples/tomato_novel_recommand/qdrant"
)

// NovelHybridSearchInput allows combined API + vector search.
type NovelHybridSearchInput struct {
	Keyword string `json:"keyword" jsonschema_description:"search keyword, e.g. sweet-pet / revenge / system"`
	Genre   string `json:"genre" jsonschema_description:"optional genre filter"`
	TopN    int    `json:"top_n" jsonschema_description:"return top N merged results, default 6"`
}

// NewNovelHybridSearchTool runs keyword API search and vector search, then merges results.
// It needs an embedding function for the vector part; if nil, creation will fail.
func NewNovelHybridSearchTool(embedFn EmbedFunc) einotool.InvokableTool {
	if embedFn == nil {
		log.Fatalf("embedFn is required for hybrid search")
	}
	qc := qdrant.NewFromEnv()
	// Reuse the same embed + qdrant client to persist fresh API results into vectors.
	sink := &qdrantVectorSink{client: qc, embed: embedFn}

	toolImpl, err := utils.InferTool(
		"novel_hybrid_search",
		"Hybrid search that combines API keyword results with vector DB matches, merges and deduplicates to return the best candidates.",
		func(ctx context.Context, input *NovelHybridSearchInput) (output *NovelSearchOutput, err error) {
			log.Printf("[tool] invoke novel_hybrid_search, input=%+v", input)
			topN := input.TopN
			if topN == 0 {
				topN = 6
			}
			query := input.Keyword
			if query == "" {
				query = input.Genre
			}
			if strings.TrimSpace(query) == "" {
				return nil, fmt.Errorf("keyword or genre is required")
			}

			// 1) API keyword search (fast, freshness)
			apiNovels, apiErr := callNovelAPI(ctx, NovelSearchParam{
				Keyword: query,
				Genre:   input.Genre,
				TopN:    topN,
			})
			if apiErr != nil {
				log.Printf("[tool] hybrid api search failed: %v", apiErr)
			} else if len(apiNovels) > 0 {
				if err := sink.StoreNovels(ctx, apiNovels); err != nil {
					log.Printf("[tool] hybrid persist api results failed: %v", err)
				}
			}

			// 2) Vector semantic search (relevance)
			vecNovels := []*Novel{}
			if vec, err := embedFn(ctx, query); err != nil {
				log.Printf("[tool] hybrid embed failed: %v", err)
			} else if novels, err := qdrantSearch(ctx, qc, vec, topN, input.Genre); err != nil {
				log.Printf("[tool] hybrid vector search failed: %v", err)
			} else {
				vecNovels = novels
			}

			merged := mergeNovels(topN, apiNovels, vecNovels)
			return &NovelSearchOutput{Novels: merged}, nil
		},
	)
	if err != nil {
		log.Fatalf("failed to create novel_hybrid_search tool: %v", err)
	}
	return toolImpl
}

// mergeNovels deduplicates by title+author, and keeps order: vector-first, then API.
func mergeNovels(limit int, vec []*Novel, api []*Novel) []*Novel {
	key := func(n *Novel) string {
		return strings.ToLower(n.Title + "::" + n.Author)
	}
	seen := make(map[string]struct{})
	out := make([]*Novel, 0, limit)

	for _, list := range [][]*Novel{vec, api} {
		for _, n := range list {
			if len(out) >= limit {
				return out
			}
			k := key(n)
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, n)
		}
	}
	return out
}

