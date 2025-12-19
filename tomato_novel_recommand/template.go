/*
 * Tomato novel recommendation demo - prompt template.
 */

package main

import (
	"context"
	"log"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

// createTemplate creates a chat template for tomato novel recommendation.
func createTemplate() prompt.ChatTemplate {
	// use FString template to build system + history + user messages
	return prompt.FromMessages(schema.FString,
		// system message template
		schema.SystemMessage("你是一名中文网络小说平台的资深编辑，擅长根据用户的阅读喜好推荐中文网文。你可以虚构书名，但题材和风格必须贴合用户的偏好。每次推荐 3-5 本书，每本书都需要给出简短的推荐理由和适合人群。回答时请使用自然、口语化的中文。"+
			"如果用户提供的信息不够具体，请优先用中文向用户追问、澄清偏好，在获取足够信息之后再给出正式的推荐。后续多轮对话要充分利用之前的聊天内容进行个性化推荐。"),

		// conversation history placeholder (can be empty)
		schema.MessagesPlaceholder("chat_history", true),

		// user message template
		schema.UserMessage("用户当前的阅读偏好是：{preference}。请基于这个偏好推荐几本中文网络小说，用轻松、有趣、接地气的中文语气来回复。"),
	)
}

// createMessagesFromTemplate builds one round of messages from template.
func createMessagesFromTemplate(preference string, history []*schema.Message) []*schema.Message {
	template := createTemplate()

	messages, err := template.Format(context.Background(), map[string]any{
		"preference":   preference,
		"chat_history": history,
	})
	if err != nil {
		log.Fatalf("format template failed: %v", err)
	}
	return messages
}
