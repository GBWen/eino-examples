package tools

import (
	"context"
	"crypto/sha1"
	"fmt"
	"log"

	"github.com/cloudwego/eino-examples/tomato_novel_recommand/qdrant"
)

// qdrantVectorSink writes novels into Qdrant so that subsequent vector searches can reuse them.
type qdrantVectorSink struct {
	client *qdrant.Client
	embed  EmbedFunc
	// cache the first successful vector dimension to avoid recreating collection repeatedly.
	dim int
}

// NewQdrantVectorSink builds a sink that persists novels into Qdrant using the provided embed function.
// If embed is nil, it returns nil to allow callers to keep existing behavior.
func NewQdrantVectorSink(embed EmbedFunc) NovelResultSink {
	if embed == nil {
		return nil
	}
	return &qdrantVectorSink{
		client: qdrant.NewFromEnv(),
		embed:  embed,
	}
}

func (s *qdrantVectorSink) StoreNovels(ctx context.Context, novels []*Novel) error {
	if len(novels) == 0 {
		return nil
	}

	points := make([]map[string]any, 0, len(novels))
	for _, n := range novels {
		vec, err := s.embed(ctx, n.Title+" "+n.Description)
		if err != nil {
			return fmt.Errorf("embed novel %q failed: %w", n.Title, err)
		}
		if s.dim == 0 {
			s.dim = len(vec)
			if err := s.client.EnsureCollection(s.dim); err != nil {
				return err
			}
		}
		points = append(points, map[string]any{
			"id":     makeStableID(n),
			"vector": vec,
			"payload": map[string]any{
				"title":       n.Title,
				"author":      n.Author,
				"genre":       n.Category,
				"description": n.Description,
				"link":        n.Link,
				"source":      "novel_search",
			},
		})
	}

	if err := s.client.UpsertPoints(points); err != nil {
		return fmt.Errorf("qdrant upsert failed: %w", err)
	}
	log.Printf("[qdrant] sink upserted %d novels into %s", len(points), s.client.Collection())
	return nil
}

func makeStableID(n *Novel) string {
	key := n.Title + "::" + n.Author
	sum := sha1.Sum([]byte(key))
	return fmt.Sprintf("%x", sum[:16])
}
