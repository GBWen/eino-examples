package workflow

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/cloudwego/eino-examples/tomato_novel_recommand/service/model"
)

type feedbackRecord struct {
	Timestamp  time.Time      `json:"ts"`
	Input      string         `json:"input"`
	Candidates []*model.Novel `json:"candidates,omitempty"`
	Reply      string         `json:"reply,omitempty"`
}

// RecordFeedback appends a simple JSON line to a local log file.
// Later this can be replaced by writing to a DB or updating a vector store.
// TODO: replace file-based logging with a persistent DB or vector-store update pipeline.
func RecordFeedback(path string, input string, candidates []*model.Novel, reply string) {
	if path == "" {
		path = "/tmp/novel_feedback.log"
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Printf("open feedback file failed: %v", err)
		return
	}
	defer f.Close()

	rec := feedbackRecord{
		Timestamp:  time.Now(),
		Input:      input,
		Candidates: candidates,
		Reply:      reply,
	}
	b, _ := json.Marshal(rec)
	b = append(b, '\n')
	if _, err := f.Write(b); err != nil {
		log.Printf("write feedback failed: %v", err)
	}
}
