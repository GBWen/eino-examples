package graph

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/tomato_novel_recommand/flow"
	tools "github.com/cloudwego/eino-examples/tomato_novel_recommand/tool"
	"github.com/cloudwego/eino-examples/tomato_novel_recommand/workflow"
)

// RunInteractiveLoop is a demo agent loop:
// read user preference -> build messages -> (optionally call tools) -> LLM stream -> maintain history.
func RunInteractiveLoop(ctx context.Context, llm model.ToolCallingChatModel, vectorTool einotool.InvokableTool, keywordTool einotool.InvokableTool) {
	reader := bufio.NewReader(os.Stdin)
	var history []*schema.Message

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

		// Step 1: clarify missing key information if the query is too short.
		preference := ensurePreference(ctx, text, reader)

		// Step 2: call vector-search / keyword-search tools to get candidate novels.
		candidates := retrieveCandidates(ctx, vectorTool, keywordTool, preference)

		// Step 3: build prompt with candidates and stream the final recommendation.
		messages, err := workflow.CreateMessagesFromTemplate(preference, candidates, history)
		if err != nil {
			log.Fatalf("format template failed: %v", err)
		}

		fmt.Printf("\n=== LLM recommendation result (streaming) ===\n")
		sr := flow.Stream(ctx, llm, messages)
		full := flow.ConsumeStream(sr, flow.StdoutWriter)

		// Update history: user messages + assistant reply.
		history = append(history, messages...)
		history = append(history, full)

		// Step 4: write feedback for later offline use (e.g. DB / vector store update).
		// TODO: consume feedback logs to update user preference vectors or training data.
		workflow.RecordFeedback("", preference, candidates, full.Content)

		fmt.Println("\n----------------------")
	}
}

// ensurePreference uses the clarify tool to complete user preference if it's too vague.
func ensurePreference(ctx context.Context, pref string, reader *bufio.Reader) string {
	if len([]rune(pref)) >= 4 {
		return pref
	}
	clarify := tools.NewClarifyTool()
	question, err := clarify.InvokableRun(ctx, `{"question":"请补充你想看的题材/风格/关键词，例如：甜宠、系统、爽文、慢热、悬疑等"}`)
	if err != nil {
		log.Printf("clarify tool failed: %v", err)
		return pref
	}
	fmt.Printf("信息不足，澄清问题：%s\n你的回答：", question)
	resp, _ := reader.ReadString('\n')
	resp = strings.TrimSpace(resp)
	if resp == "" {
		return pref
	}
	answer, err := clarify.InvokableRun(ctx, `{"question":"ok"}`, tools.WithUserResponse(resp))
	if err != nil {
		log.Printf("clarify tool failed: %v", err)
		return resp
	}
	return answer
}

// retrieveCandidates calls the vector-search tool, and falls back to keyword-search tool on error or empty result.
func retrieveCandidates(ctx context.Context, vectorTool einotool.InvokableTool, keywordTool einotool.InvokableTool, pref string) []*tools.Novel {
	args, _ := json.Marshal(map[string]any{
		"keyword": pref,
		"top_n":   5,
	})

	// First try vector search.
	if vectorTool != nil {
		if res := invokeTool(ctx, vectorTool, args); len(res) > 0 {
			return res
		}
		log.Printf("vector search fallback to keyword search")
	}

	// Fallback to keyword search.
	if keywordTool != nil {
		if res := invokeTool(ctx, keywordTool, args); len(res) > 0 {
			return res
		}
	}
	return nil
}

func invokeTool(ctx context.Context, t einotool.InvokableTool, args []byte) []*tools.Novel {
	out, err := t.InvokableRun(ctx, string(args))
	if err != nil {
		log.Printf("invoke tool failed: %v", err)
		return nil
	}
	var res tools.NovelSearchOutput
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		log.Printf("decode tool output failed: %v", err)
		return nil
	}
	return res.Novels
}
