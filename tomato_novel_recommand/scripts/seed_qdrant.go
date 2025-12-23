package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/cloudwego/eino-examples/tomato_novel_recommand/qdrant"
)

// Novel represents the minimal schema we ingest into Qdrant.
type Novel struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Link        string `json:"link"`
}

type xcvtsResponse struct {
	Data interface{} `json:"data"`
}

// EmbedClient calls a real embedding service (OpenAI/Ark compatible).
// Default: reuse the same Ark config as chat.
//   - API Key: EMBED_API_KEY; fallback ARK_API_KEY
//   - BaseURL: EMBED_BASE_URL; fallback ARK_BASE_URL; else default https://ark.cn-beijing.volces.com/api/v3
//   - Model:   EMBED_MODEL; fallback chat default (doubao-seed-1-6-251015)
type EmbedClient struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func newEmbedClientFromEnv() (*EmbedClient, error) {
	key := os.Getenv("EMBED_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("EMBED_API_KEY is required")
	}

	model := "doubao-embedding-vision-250615"

	base := "https://ark.cn-beijing.volces.com/api/v3/embeddings/multimodal"
	return &EmbedClient{
		apiKey:  key,
		model:   model,
		baseURL: base,
		client:  &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func (c *EmbedClient) Embed(ctx context.Context, text string) ([]float32, error) {
	body := map[string]any{
		// Ark multimodal embedding expects an array of typed inputs.
		"input": []map[string]any{
			{
				"type": "text",
				"text": text,
			},
		},
		"model": c.model,
	}
	b, _ := json.Marshal(body)
	// c.baseURL is already the full embeddings endpoint (Ark multimodal); do not append extra path.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding status %d, body=%s", resp.StatusCode, string(body))
	}

	// Read full body once to support multiple decode strategies.
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Strategy 1: data is an array
	var arrResp struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &arrResp); err == nil && len(arrResp.Data) > 0 && len(arrResp.Data[0].Embedding) > 0 {
		return arrResp.Data[0].Embedding, nil
	}

	// Strategy 2: data is an object with embedding field
	var objResp struct {
		Data struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &objResp); err == nil && len(objResp.Data.Embedding) > 0 {
		return objResp.Data.Embedding, nil
	}

	return nil, fmt.Errorf("unexpected embedding response format: %s", string(raw))
}

func fetchNovels(q string) ([]*Novel, error) {
	apiURL := fmt.Sprintf("https://api.xcvts.cn/api/xiaoshuo/fanqie?q=%s", url.QueryEscape(q))
	req, _ := http.NewRequest(http.MethodGet, apiURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api status %d", resp.StatusCode)
	}
	var r xcvtsResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}

	arr, ok := r.Data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	var novels []*Novel
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		title, _ := m["title"].(string)
		author, _ := m["author"].(string)
		abstract, _ := m["abstract"].(string)
		bookID, _ := m["book_id"].(string)
		link := ""
		if bookID != "" {
			link = fmt.Sprintf("https://fanqienovel.com/page/%s", bookID)
		}
		novels = append(novels, &Novel{
			ID:          bookID,
			Title:       title,
			Author:      author,
			Category:    q, // use query term as a coarse genre tag
			Description: abstract,
			Link:        link,
		})
	}
	return novels, nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	embedClient, err := newEmbedClientFromEnv()
	if err != nil {
		log.Fatalf("init embed client failed: %v", err)
	}

	client := qdrant.NewFromEnv()

	queries := []string{"玄幻", "都市", "言情", "科幻", "悬疑", "历史"}
	target := 30
	var all []*Novel

	for _, q := range queries {
		ns, err := fetchNovels(q)
		if err != nil {
			log.Printf("fetch %s failed: %v", q, err)
			continue
		}
		all = append(all, ns...)
		if len(all) >= target {
			break
		}
		time.Sleep(300 * time.Millisecond) // avoid hitting API too fast
	}

	if len(all) == 0 {
		log.Fatalf("no novels fetched, abort")
	}
	if len(all) > target {
		all = all[:target]
	}

	// Build points with real embeddings
	points := make([]map[string]interface{}, 0, len(all))
	for _, n := range all {
		vec, err := embedClient.Embed(context.Background(), n.Title+" "+n.Description)
		if err != nil {
			log.Fatalf("embed failed for %s: %v", n.Title, err)
		}
		id := makeQdrantID(n)
		points = append(points, map[string]interface{}{
			"id":     id,
			"vector": vec,
			"payload": map[string]interface{}{
				"id":          n.ID,
				"title":       n.Title,
				"author":      n.Author,
				"genre":       n.Category,
				"description": n.Description,
				"link":        n.Link,
				"seed_query":  n.Category,
				"source":      "xcvts_seed",
			},
		})
	}

	// Create/overwrite collection using the first vector's dimension
	dim := len(points[0]["vector"].([]float32))
	if err := client.EnsureCollection(dim); err != nil {
		log.Fatalf("create collection failed: %v", err)
	}
	log.Printf("collection ready: %s (dim=%d)", client.Collection(), dim)

	if err := client.UpsertPoints(points); err != nil {
		log.Fatalf("upsert failed: %v", err)
	}
	log.Printf("done: upserted %d points into %s", len(points), client.Collection())
}

// deterministicUUID returns a RFC4122-compliant UUID derived from input string (stable across runs).
func deterministicUUID(s string) string {
	sum := sha1.Sum([]byte(s))
	var u [16]byte
	copy(u[:], sum[:16])
	u[6] = (u[6] & 0x0f) | 0x40 // version 4
	u[8] = (u[8] & 0x3f) | 0x80 // variant RFC4122
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(u[0:4]),
		binary.BigEndian.Uint16(u[4:6]),
		binary.BigEndian.Uint16(u[6:8]),
		binary.BigEndian.Uint16(u[8:10]),
		u[10:16],
	)
}

// makeQdrantID prefers the source book_id; if it is a positive integer, use it directly.
// Otherwise, fall back to a deterministic UUID.
func makeQdrantID(n *Novel) any {
	if n.ID != "" {
		if v, err := strconv.ParseUint(n.ID, 10, 64); err == nil {
			return v
		}
	}
	return deterministicUUID(n.Title + "::" + n.Author)
}
