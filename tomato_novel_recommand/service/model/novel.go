package model

import "context"

// Novel represents the key information of a single novel.
type Novel struct {
	Title       string `json:"title"`
	Author      string `json:"author"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Link        string `json:"link"`
}

// NovelResultSink persists/searches novel results to a backing store (e.g., vector DB).
type NovelResultSink interface {
	StoreNovels(ctx context.Context, novels []*Novel) error
}
