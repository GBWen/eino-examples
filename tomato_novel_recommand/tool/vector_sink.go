package tools

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// qdrantVectorSink writes novels into Qdrant so that subsequent vector searches can reuse them.
type qdrantVectorSink struct {
	client *qdrantClient
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
		client: newQdrantClientFromEnv(),
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
			if err := s.client.ensureCollection(s.dim); err != nil {
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

	if err := s.client.upsertPoints(points); err != nil {
		return fmt.Errorf("qdrant upsert failed: %w", err)
	}
	log.Printf("[qdrant] sink upserted %d novels into %s", len(points), s.client.collection)
	return nil
}

func makeStableID(n *Novel) string {
	key := n.Title + "::" + n.Author
	sum := sha1.Sum([]byte(key))
	return fmt.Sprintf("%x", sum[:16])
}

// ensureCollection creates the collection if missing.
func (c *qdrantClient) ensureCollection(dim int) error {
	payload := map[string]any{
		"vectors": map[string]any{
			"size":     dim,
			"distance": "Cosine",
		},
	}
	b, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/collections/%s", c.baseURL, c.collection)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 200 OK or 409 Conflict (already exists) are both acceptable.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("ensure collection status %d", resp.StatusCode)
	}
	return nil
}

// upsertPoints writes points using the standard Qdrant /points endpoint.
func (c *qdrantClient) upsertPoints(points []map[string]any) error {
	body := map[string]any{
		"points": points,
	}
	b, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/collections/%s/points", c.baseURL, c.collection)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("upsert status %d", resp.StatusCode)
	}
	return nil
}

