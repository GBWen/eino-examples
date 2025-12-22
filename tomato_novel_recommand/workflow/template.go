package workflow

import (
	"context"
	"encoding/json"
	"strings"

	tools "github.com/cloudwego/eino-examples/tomato_novel_recommand/tool"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

// CreateTemplate builds the chat template for the novel recommender.
func CreateTemplate() prompt.ChatTemplate {
	return prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一名中文网络小说平台的资深编辑，擅长根据用户的阅读喜好推荐中文网文。你可以虚构书名，但题材和风格必须贴合用户的偏好。每次推荐 3-5 本书，每本书都需要给出简短的推荐理由和适合人群，优先使用已检索到的候选书目。回答时请使用自然、口语化的中文。"+
			"如果信息不足，请先澄清再给正式推荐。多轮对话要利用历史偏好。"),
		schema.MessagesPlaceholder("chat_history", true),
		schema.UserMessage("用户当前需求：{preference}\n\n"+
			"已检索候选（可直接引用/精排）：\n{candidates}\n\n"+
			"请基于上述信息生成推荐。若候选为空，先向用户确认或给出探索性提问。"),
	)
}

// CreateMessagesFromTemplate renders messages with user preference, candidates, and history.
func CreateMessagesFromTemplate(preference string, candidates []*tools.Novel, history []*schema.Message) ([]*schema.Message, error) {
	template := CreateTemplate()
	cands := formatCandidates(candidates)
	return template.Format(context.Background(), map[string]any{
		"preference":   preference,
		"candidates":   cands,
		"chat_history": history,
	})
}

func formatCandidates(candidates []*tools.Novel) string {
	if len(candidates) == 0 {
		return "（暂无检索结果）"
	}
	var b strings.Builder
	for i, n := range candidates {
		_ = json.NewEncoder(&b).Encode(map[string]any{
			"rank":        i + 1,
			"title":       n.Title,
			"author":      n.Author,
			"genre":       n.Category,
			"description": n.Description,
			"link":        n.Link,
		})
	}
	return b.String()
}
