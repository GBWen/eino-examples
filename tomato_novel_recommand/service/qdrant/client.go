package qdrant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cloudwego/eino-examples/tomato_novel_recommand/config"
)

// Client wraps minimal Qdrant HTTP operations used in this demo.
// It is extracted from the seed script so both runtime sinks and seeders share the same logic.
type Client struct {
	baseURL    string
	collection string
	httpClient *http.Client
}

// New builds a Client with explicit URL/collection.
func New(baseURL, collection string) *Client {
	return &Client{
		baseURL:    baseURL,
		collection: collection,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// NewWithConfig builds Client from QdrantConfig.
func NewWithConfig(cfg config.QdrantConfig) *Client {
	return New(cfg.URL, cfg.Collection)
}

// NewFromEnv builds a Client using env/config defaults.
func NewFromEnv() *Client {
	cfg := config.LoadQdrantConfig()
	return NewWithConfig(cfg)
}

// Collection returns the configured collection name.
func (c *Client) Collection() string {
	return c.collection
}

// BaseURL returns the configured base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// HTTPClient returns the underlying HTTP client (used by search tool).
func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

// EnsureCollection creates the collection if missing. 200 OK or 409 Conflict are accepted.
// Distance uses Cosine to align with seed script defaults.
func (c *Client) EnsureCollection(dim int) error {
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

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusConflict {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("ensure collection status %d, body=%s", resp.StatusCode, string(body))
}

// UpsertPoints writes points using Qdrant /points endpoint. Returns error on any non-2xx status.
func (c *Client) UpsertPoints(points []map[string]any) error {
	body := map[string]any{
		"points": points,
	}
	b, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/collections/%s/points", c.baseURL, c.collection)
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

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upsert status %d, body=%s", resp.StatusCode, string(respBody))
	}
	return nil
}
