package qdrant

import (
	"context"
	"crypto/sha1"
	"fmt"
	"log"
	"strings"

	"github.com/cloudwego/eino-examples/tomato_novel_recommand/service/model"
)

// EmbedFunc defines embedding function signature.
type EmbedFunc func(ctx context.Context, text string) ([]float32, error)

// VectorSink writes novel payloads into Qdrant.
type VectorSink struct {
	client *Client
	embed  EmbedFunc
	dim    int
}

// NewVectorSink builds a sink with the default client (env-configured).
// Returns nil if embed is nil to preserve caller behavior.
func NewVectorSink(embed EmbedFunc) *VectorSink {
	if embed == nil {
		return nil
	}
	return &VectorSink{
		client: NewFromEnv(),
		embed:  embed,
	}
}

// StoreNovels persists novel payloads with generated vectors.
func (s *VectorSink) StoreNovels(ctx context.Context, novels []*model.Novel) error {
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
			"id":     makeStableID(n.Title, n.Author),
			"vector": vec,
			"payload": map[string]any{
				"title":       n.Title,
				"author":      n.Author,
				"genre":       n.Category,
				"description": n.Description,
				"link":        n.Link,
				"source":      inferSource(n),
			},
		})
	}

	if err := s.client.UpsertPoints(points); err != nil {
		return fmt.Errorf("qdrant upsert failed: %w", err)
	}
	log.Printf("[qdrant] sink upserted %d novels into %s", len(points), s.client.Collection())
	return nil
}

func makeStableID(title, author string) string {
	key := title + "::" + author
	sum := sha1.Sum([]byte(key))
	return fmt.Sprintf("%x", sum[:16])
}

func inferSource(n *model.Novel) string {
	if strings.TrimSpace(n.Category) != "" {
		return n.Category
	}
	return "novel_search"
}
