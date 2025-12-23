package tools

import (
	"context"

	"github.com/cloudwego/eino-examples/tomato_novel_recommand/qdrant"
)

// NewQdrantVectorSink adapts qdrant.VectorSink to NovelResultSink.
func NewQdrantVectorSink(embed EmbedFunc) NovelResultSink {
	sink := qdrant.NewVectorSink(qdrant.EmbedFunc(embed))
	if sink == nil {
		return nil
	}
	return &qdrantSinkAdapter{sink: sink}
}

type qdrantSinkAdapter struct {
	sink *qdrant.VectorSink
}

func (a *qdrantSinkAdapter) StoreNovels(ctx context.Context, novels []*Novel) error {
	payloads := make([]qdrant.NovelPayload, 0, len(novels))
	for _, n := range novels {
		payloads = append(payloads, qdrant.NovelPayload{
			Title:       n.Title,
			Author:      n.Author,
			Genre:       n.Category,
			Description: n.Description,
			Link:        n.Link,
			Source:      "novel_search",
		})
	}
	return a.sink.Store(ctx, payloads)
}

