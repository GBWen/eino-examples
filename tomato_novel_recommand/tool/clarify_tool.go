package tools

import (
	"context"
	"fmt"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type clarifyOptions struct {
	UserResponse *string
}

// WithUserResponse passes the user's response when calling the tool the second time.
func WithUserResponse(resp string) einotool.Option {
	return einotool.WrapImplSpecificOptFn(func(o *clarifyOptions) {
		o.UserResponse = &resp
	})
}

type ClarifyInput struct {
	Question string `json:"question" jsonschema_description:"question you want to ask the user to fill in missing key information"`
}

// NewClarifyTool creates a clarify tool:
// - first call (no user response): returns the question to ask the user
// - second call (with user response): returns the user's answer.
func NewClarifyTool() einotool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"clarify_missing_info",
		"When the user's request is incomplete, use this tool to ask a clarification question and wait for the user's answer.",
		func(ctx context.Context, input *ClarifyInput, opts ...einotool.Option) (output string, err error) {
			o := einotool.GetImplSpecificOptions[clarifyOptions](nil, opts...)
			if o.UserResponse == nil {
				// First call: return the question to ask the user.
				return input.Question, nil
			}
			// Second call: return the user's answer content.
			return *o.UserResponse, nil
		},
	)
	if err != nil {
		panic(fmt.Errorf("create clarify tool failed: %w", err))
	}
	return t
}
