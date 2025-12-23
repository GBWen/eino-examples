package graph

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/tomato_novel_recommand/workflow"
)

// RunInteractiveLoop uses ReAct Agent to let the model automatically decide which tools to call.
// The model will:
// 1. Decide if clarification is needed (via clarify_missing_info tool)
// 2. Search for novels (via novel_search or novel_vector_search tool)
// 3. Rerank and generate recommendations based on search results
func RunInteractiveLoop(ctx context.Context, llm model.ToolCallingChatModel, toolsList []tool.BaseTool) {
	reader := bufio.NewReader(os.Stdin)
	var history []*schema.Message

	// Create ReAct Agent with all available tools
	// The model will automatically decide which tools to call
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: llm,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: toolsList,
		},
		MaxStep: 10, // Limit max tool-calling steps to avoid infinite loops
	})
	if err != nil {
		log.Fatalf("failed to create react agent: %v", err)
	}

	// System prompt: guide the model to use tools appropriately
	systemPrompt := "你是一名中文网络小说平台的资深编辑，擅长根据用户的阅读喜好推荐中文网文。" +
		"你可以使用以下工具：\n" +
		"- clarify_missing_info: 当用户需求不够明确时，先询问补充信息\n" +
		"- novel_search: 基于关键词/类型搜索小说（默认使用）\n" +
		"- novel_vector_search: 基于语义向量搜索小说（可选，适合大规模书库）\n\n" +
		"推荐流程：1) 如需要先澄清用户偏好 2) 调用搜索工具获取候选 3) 从候选中选择3-5本最匹配的，给出推荐理由。\n" +
		"回答时使用自然、口语化的中文。"

	for {
		fmt.Print("请输入你当前想看的小说类型 / 心情 / 偏好（输入 exit 退出）：")
		text, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("read input failed: %v", err)
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if strings.EqualFold(text, "exit") {
			fmt.Println("再见，期待下次帮你找书～")
			return
		}

		// Build messages with system prompt and history
		messages := []*schema.Message{
			schema.SystemMessage(systemPrompt),
		}
		messages = append(messages, history...)
		messages = append(messages, schema.UserMessage(text))

		// Let the agent automatically call tools and generate response
		fmt.Printf("\n=== LLM recommendation result (streaming) ===\n")
		sr, err := agent.Stream(ctx, messages)
		if err != nil {
			log.Fatalf("agent stream failed: %v", err)
		}

		full := &schema.Message{
			Role: schema.Assistant,
		}

		// Consume stream and print
		for {
			msg, err := sr.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				log.Fatalf("stream recv failed: %v", err)
			}
			if msg != nil && msg.Content != "" {
				fmt.Print(msg.Content)
				full.Content += msg.Content
			}
		}
		fmt.Println()

		// Update history
		history = append(history, schema.UserMessage(text))
		history = append(history, full)

		// Record feedback (simplified: we don't have candidates here since model handles it)
		// TODO: extract candidates from tool calls in the future
		workflow.RecordFeedback("", text, nil, full.Content)

		fmt.Println("\n----------------------")
	}
}
