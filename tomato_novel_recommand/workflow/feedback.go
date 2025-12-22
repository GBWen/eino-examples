package workflow

import (
	"encoding/json"
	"log"
	"os"
	"time"

	tools "github.com/cloudwego/eino-examples/tomato_novel_recommand/tool"
)

type feedbackRecord struct {
	Timestamp  time.Time      `json:"ts"`
	Input      string         `json:"input"`
	Candidates []*tools.Novel `json:"candidates,omitempty"`
	Reply      string         `json:"reply,omitempty"`
}

// RecordFeedback 写入简单的日志文件，后续可替换为 DB/向量库更新。
func RecordFeedback(path string, input string, candidates []*tools.Novel, reply string) {
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
