## 番茄小说推荐 Demo（Ark + Tool）

这个示例演示如何用 **Eino + Ark ChatModel** 搭一个「番茄小说推荐」小应用，并接入一个 **NovelSearch Tool** 来模拟从番茄小说检索数据，再由大模型生成“标题 + 简介”的推荐结果。

### 目录结构

- `main.go`：交互式 CLI 入口，读取用户输入的阅读偏好，循环推荐小说
- `ark.go`：Ark Chat 模型初始化（`createArkChatModel`）
- `template.go`：番茄小说推荐的 `PromptTemplate`，包含系统提示 + 历史对话 + 用户偏好
- `generate.go`：封装一次性和流式生成的辅助函数
- `stream.go`：示例用的流式输出打印工具
- `novel_tool.go`：**NovelSearch Tool 实现 + 绑定 Ark ChatModel 的辅助函数**

### 0. 前置要求

1. 安装依赖（在仓库根目录）：

```bash
go mod tidy
```

2. 准备 Ark API Key：

```bash
export ARK_API_KEY=你的_ark_api_key
```

如需自定义模型或 BaseURL，可以在 `ark.go` 中调整 `ark.ChatModelConfig`。

### 1. 运行 Demo

在仓库根目录执行：

```bash
ARK_API_KEY=xxx go run ./tomato_novel_recommand
```

终端会进入一个简单的对话循环，每轮会提示：

> 请输入你当前想看的小说类型 / 心情 / 偏好（输入 exit 退出）：

你可以输入：

- `玄幻 爽文 系统流`
- `想看轻松搞笑一点的现代言情，最好是短篇的`
- `最近有点压力大，想看治愈一点的日常文`

Ark 会基于 `template.go` 中的 PromptTemplate 和历史对话，结合内部的推荐逻辑/Tool 结果，生成一段包含多本小说「标题 + 简短推荐语」的回复。

### 2. NovelSearch Tool（novel_tool.go）

`novel_tool.go` 中的核心内容：

- **数据结构**
  - `Novel`：单本小说（标题、作者、分类、简介、链接）
  - `NovelSearchParam`：搜索参数（`keyword` / `genre` / `top_n`）
- **Tool 实现**
  - `NovelSearchTool`：实现了 `Info` 和 `InvokableRun`，符合 Eino 的 `InvokableTool` 接口
  - `InvokableRun`：
    1. 解析大模型传入的 JSON 参数
    2. 调用 `callTomatoAPI` 获取小说列表（当前为 mock，实际可接 HTTP / RPC）
    3. 将结果序列化成 JSON 字符串返回给大模型
- **绑定函数**
  - `bindNovelSearchTool(ctx, cm)`：把 `NovelSearch` Tool 绑定到 Ark ChatModel 上，返回一个带 Tool 能力的新模型实例：

```go
cm := createArkChatModel(ctx)
cm, novelTool := bindNovelSearchTool(ctx, cm)
```

后续可以在 Graph / Agent 中，把 `novelTool` 放到 `compose.NewToolNode` 的 `Tools` 列表里，就能让 Ark 在 ReAct / Tool Call 流程中自动调用小说搜索能力。

### 3. 如何改成你自己的业务

1. **替换 Tool 实现**
   - 修改 `callTomatoAPI`，接入你自己的番茄小说服务 / 书库 API
   - 保持返回结构语义清晰（字段名尽量“可读”，便于大模型理解）

2. **调整 PromptTemplate**
   - 在 `template.go` 中修改系统提示和用户模板，使之更贴合你的推荐规则（比如增加标签、偏好、黑名单等）

3. **扩展为 Agent / Flow**
   - 如果你希望让模型自动决定「是否搜索」「搜索几次」「如何综合结果」，可以基于 `bindNovelSearchTool` 输出的模型，构建 ReAct Agent 或 Graph（参考仓库中的 `compose/graph/tool_call_agent` 等示例）。




