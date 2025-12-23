package tools

import (
	"context"
	"fmt"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type ClarifyInput struct {
	Question string `json:"question" jsonschema_description:"The clarification question to ask the user. Should be specific and helpful, e.g. '请告诉我你想看的题材（如玄幻、都市、言情等）和风格偏好（如爽文、慢热、甜宠等）'"`
}

// NewClarifyTool creates a clarify tool for ReAct Agent.
// When called, it returns the question that should be asked to the user.
// The agent will use this question as its response, and wait for the user's next input.
func NewClarifyTool() einotool.InvokableTool {
	t, err := utils.InferTool(
		"clarify_missing_info",
		"Use this tool when the user's request is too vague or incomplete (e.g., just says 'recommend', '随便', or lacks genre/style preferences). "+
			"This tool returns a clarification question that you should ask the user. "+
			"After asking, wait for the user's response in the next turn before calling search tools.",
		func(ctx context.Context, input *ClarifyInput) (output string, err error) {
			if input.Question == "" {
				// Default question if not provided
				return "请告诉我你想看的题材（如玄幻、都市、言情、悬疑等）和风格偏好（如爽文、慢热、甜宠、系统流等），这样我才能为你推荐合适的小说。", nil
			}
			return input.Question, nil
		},
	)
	if err != nil {
		panic(fmt.Errorf("create clarify tool failed: %w", err))
	}
	return t
}
