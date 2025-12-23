package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// NewArkEmbedFuncFromEnv builds an EmbedFunc using Ark Embedding API.
// Requires EMBED_API_KEY; base/model have sane defaults.
func NewArkEmbedFuncFromEnv() (EmbedFunc, error) {
	apiKey := os.Getenv("EMBED_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("EMBED_API_KEY is required for Ark embedding")
	}
	baseURL := os.Getenv("EMBED_BASE_URL")
	if baseURL == "" {
		baseURL = "https://ark.cn-beijing.volces.com/api/v3/embeddings/multimodal"
	}
	model := os.Getenv("EMBED_MODEL")
	if model == "" {
		model = "doubao-embedding-vision-250615"
	}
	client := &http.Client{Timeout: 15 * time.Second}

	return func(ctx context.Context, text string) ([]float32, error) {
		body := map[string]any{
			"input": []map[string]any{{"type": "text", "text": text}},
			"model": model,
		}
		b, _ := json.Marshal(body)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			rb, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("embedding status %d, body=%s", resp.StatusCode, string(rb))
		}

		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var arrResp struct {
			Data []struct {
				Embedding []float32 `json:"embedding"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &arrResp); err == nil && len(arrResp.Data) > 0 && len(arrResp.Data[0].Embedding) > 0 {
			return arrResp.Data[0].Embedding, nil
		}

		var objResp struct {
			Data struct {
				Embedding []float32 `json:"embedding"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &objResp); err == nil && len(objResp.Data.Embedding) > 0 {
			return objResp.Data.Embedding, nil
		}

		return nil, fmt.Errorf("unexpected embedding response: %s", string(raw))
	}, nil
}
