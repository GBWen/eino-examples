package workflow

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/components/tool"
)

// GetSystemPrompt returns the system prompt for the novel recommendation agent.
// It dynamically generates the prompt based on available tools to avoid mentioning tools that aren't bound.
func GetSystemPrompt(ctx context.Context, toolsList []tool.BaseTool) string {
	basePrompt := "你是一名中文网络小说平台的资深编辑，擅长根据用户的阅读喜好推荐中文网文。\n\n" +
		"如果你已经多次使用工具但仍无法显著提高推荐质量，应停止继续调用工具，基于当前已有的信息给出你能提供的最佳推荐结果。\n\n"

	// Build tool descriptions based on actually available tools
	var toolDescs []string
	hasClarify := false
	hasKeywordSearch := false
	hasVectorSearch := false

	for _, t := range toolsList {
		info, err := t.Info(ctx)
		if err != nil {
			continue
		}
		switch info.Name {
		case "clarify_missing_info":
			hasClarify = true
			toolDescs = append(toolDescs, "- clarify_missing_info: 当用户需求模糊或不完整时，使用此工具生成澄清问题询问用户，获取明确偏好后再搜索")
		case "novel_search":
			hasKeywordSearch = true
			toolDescs = append(toolDescs, "- novel_search: 基于关键词/类型搜索小说")
		case "novel_vector_search":
			hasVectorSearch = true
			toolDescs = append(toolDescs, "- novel_vector_search: 基于语义向量搜索小说（适合大规模书库）")
		}
	}

	if len(toolDescs) > 0 {
		basePrompt += "你可以使用以下工具：\n" + strings.Join(toolDescs, "\n") + "\n\n"
	}

	// Build workflow description based on available tools
	var workflowSteps []string
	if hasClarify {
		workflowSteps = append(workflowSteps, "1) **重要**：如果用户需求模糊（如只说'推荐'、'随便'、'好看的书'等），或缺少关键信息（题材、风格、关键词），必须先使用 clarify_missing_info 工具询问用户，获取明确偏好后再搜索")
	}
	if hasKeywordSearch || hasVectorSearch {
		workflowSteps = append(workflowSteps, "2) 在用户需求明确后，调用搜索工具获取候选")
	}
	workflowSteps = append(workflowSteps, "3) 从候选中选择3-5本最匹配的，给出推荐理由，并在每本书后面给出可点击的阅读链接（如果工具返回了 link 字段）")

	if len(workflowSteps) > 0 {
		basePrompt += "推荐流程：" + strings.Join(workflowSteps, " ") + "。\n\n"
	}

	basePrompt += "**重要规则**：\n" +
		"- 如果用户输入过于简单或模糊（少于10个字，或缺少具体偏好），必须先澄清，不要直接搜索\n" +
		"- 只有在用户提供了明确的题材、风格、关键词等信息后，才调用搜索工具\n" +
		"- 回答中推荐书目时，优先使用工具结果中的 title / author / category / description / link 字段，并明确写出“阅读链接：<URL>”\n" +
		"- 回答时使用自然、口语化的中文"

	return basePrompt
}
