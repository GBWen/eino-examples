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

// WithUserResponse 在第二次调用时提供用户回答。
func WithUserResponse(resp string) einotool.Option {
	return einotool.WrapImplSpecificOptFn(func(o *clarifyOptions) {
		o.UserResponse = &resp
	})
}

type ClarifyInput struct {
	Question string `json:"question" jsonschema_description:"想问用户以补足关键信息的问题"`
}

// NewClarifyTool 生成澄清工具：若未提供用户回答，则返回问题文本；提供后直接返回回答。
func NewClarifyTool() einotool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"clarify_missing_info",
		"当用户提供的信息不完整时，提出澄清问题并等待用户回答。",
		func(ctx context.Context, input *ClarifyInput, opts ...einotool.Option) (output string, err error) {
			o := einotool.GetImplSpecificOptions[clarifyOptions](nil, opts...)
			if o.UserResponse == nil {
				// 第一次调用：返回要询问的问题
				return input.Question, nil
			}
			// 第二次调用：返回用户回答内容
			return *o.UserResponse, nil
		},
	)
	if err != nil {
		panic(fmt.Errorf("create clarify tool failed: %w", err))
	}
	return t
}
