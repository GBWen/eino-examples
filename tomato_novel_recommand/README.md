## 番茄小说推荐 Demo（Ark + Tool）

这个示例演示如何用 **Eino + Ark ChatModel** 搭一个「番茄小说推荐」小应用，并接入一个 **NovelSearch Tool** 来模拟从番茄小说检索数据，再由大模型生成“标题 + 简介”的推荐结果。

### 目录结构

- `main.go`：交互式 CLI 入口（创建模型、绑定工具、启动 Agent 循环）
- `flow/`：流程底座，Ark ChatModel 初始化、生成/流式封装
- `workflow/`：PromptTemplate、候选拼接、用户反馈写回
- `tool/`：小说检索相关 Tool 封装（关键词检索 + 向量检索 + 澄清工具等）
- `graph/`：Agent Graph/循环模板（澄清 → 检索 → 精排生成 → 反馈）

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

Ark 会基于 `workflow/template.go` 中的 PromptTemplate、`graph/loop.go` 中的 Agent 流程，以及绑定的 Tool 结果，生成一段包含多本小说「标题 + 简短推荐语」的回复。

### 2. 当前流程说明（Flow / Graph / Workflow / Tool）

整体分层与职责：

- **Component 层（Tool / ChatModel / Embedding 等）**
  - `tool/novel_tool.go`：`novel_search` 关键词检索 Tool（用于 fallback），内部目前用 `callTomatoAPI` 返回 mock 数据。
  - `tool/vector_tool.go`：`novel_vector_search` 向量检索 Tool，基于 Qdrant + `EmbedFunc` 封装。
  - `tool/clarify_tool.go`：`clarify_missing_info` 澄清 Tool，在信息不足时向用户追问。
  - `flow/ark.go`：Ark ChatModel 初始化。
  - `flow/generate.go`：封装普通/流式生成。

- **Workflow Step 层**
  - `workflow/template.go`：将「用户需求 + 检索候选 + 历史对话」拼成提示词，供 ChatModel 精排生成。
  - `workflow/feedback.go`：将「用户输入 + 候选列表 + 模型回复」写入本地日志，作为后续更新兴趣向量/训练数据的入口。

- **Graph 层**
  - `graph/loop.go`：一个简单的 Agent 循环：
    1. 读取用户输入；
    2. `ensurePreference` 使用澄清 Tool 补齐关键信息；
    3. `retrieveCandidates` 先走向量检索 Tool，失败或为空时回退到关键词检索 Tool；
    4. 调用 `workflow.CreateMessagesFromTemplate` 将候选注入 prompt，流式输出推荐结果；
    5. 调用 `workflow.RecordFeedback` 记录反馈。

- **Flow 层**
  - `main.go`：组合上述组件，构造一个「澄清 → 检索 → 精排 → 反馈」的 ReAct 风格多轮推荐 Agent。

### 3. TODO / 后续扩展方向

代码里已经标了一些关键 TODO，可以按需实现成你的业务版本：

- **真实业务接入**
  - `tool/novel_tool.go`：`callTomatoAPI` 目前是 mock 数据。  
    - TODO：接入真实番茄小说或你自有书库的 HTTP/RPC API，替换硬编码结果。
  - `tool/vector_tool.go`：`newQdrantClientFromEnv` 里默认 `http://localhost:6333`。  
    - TODO：通过配置文件/启动参数管理 Qdrant endpoint 和 collection，而不是写死 localhost。
  - `main.go`：`newDemoEmbedFn` 只是一个伪向量生成。  
    - TODO：替换为真实 Embedding 服务（例如 Ark Embedding），并保证向量维度与 Qdrant collection 一致。

- **反馈闭环 & 兴趣向量**
  - `workflow/feedback.go`：当前只是将反馈 append 到 `/tmp/novel_feedback.log`。  
    - TODO：改为写入数据库或消息队列，异步更新用户兴趣向量 / 召回库权重。
  - `graph/loop.go`：在写完 feedback 后仅做了注释。  
    - TODO：实现一个离线/定时任务消费这些反馈日志，根据点击/满意度等信号更新用户画像。

- **工程化 & 可配置**
  - `tool/vector_tool.go`：  
    - TODO：通过依赖注入传入 `qdrantClient` 和 `EmbedFunc`，方便单测和多环境配置。
  - 可以进一步抽出配置结构体（如 `Config{ Ark, Qdrant, Embedding, FeedbackSink }`），统一管理所有外部依赖。







