## 番茄小说推荐 Demo（Ark + ReAct Agent）

这个示例演示如何用 **Eino + Ark ChatModel + ReAct Agent** 搭一个「番茄小说推荐」小应用。

### 架构设计

**核心思路：关键词搜索 + LLM 精排**
- 使用 **ReAct Agent** 让模型自动决定何时调用哪个 Tool
- 默认使用 **关键词搜索**（适合中小规模书库，几千到几万本）
- 预留 **向量检索** 作为可选扩展（适合大规模书库，百万级以上）

**工作流程：**
1. 用户输入偏好 → ReAct Agent 接收
2. 模型自动判断是否需要澄清（调用 `clarify_missing_info` Tool）
3. 模型自动调用搜索工具（`novel_search` 或 `novel_vector_search`）
4. 模型基于搜索结果精排，生成 3-5 本推荐并给出理由

### 目录结构

- `main.go`：交互式 CLI 入口，初始化 ReAct Agent 和 Tools
- `flow/`：流程底座，Ark ChatModel 初始化、生成/流式封装
- `workflow/`：反馈记录（可选，用于后续用户画像更新）
- `tool/`：Tool 封装
  - `novel_tool.go`：关键词搜索 Tool（默认使用）
  - `vector_tool.go`：向量检索 Tool（可选扩展，需 Qdrant + Embedding）
  - `clarify_tool.go`：澄清工具
- `graph/`：ReAct Agent 循环封装（CLI 交互）

### 0. 前置要求

1. 安装依赖（在仓库根目录）：

```bash
go mod tidy
```

2. 准备 Ark API Key：

```bash
export ARK_API_KEY=你的_ark_api_key
```


如需自定义模型或 BaseURL，可以在 `flow/ark.go` 中调整 `ark.ChatModelConfig`。

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

模型会自动调用工具并生成推荐。例如：
- 用户说"我想看甜宠文" → 模型自动调用 `novel_search` → 基于结果生成推荐
- 用户说"推荐"（信息不足）→ 模型自动调用 `clarify_missing_info` → 询问后搜索 → 生成推荐

### 2. 架构说明（ReAct Agent + Tools）

**核心组件：**

- **ReAct Agent** (`graph/loop.go`)
  - 使用 `react.NewAgent` 创建，让模型自动决定调用哪个 Tool
  - 模型会自动进行多轮 Tool Calling，直到生成最终推荐
  - 无需硬编码流程，模型自己决定：是否需要澄清 → 用哪个搜索工具 → 如何精排

- **Tools** (`tool/`)
  - `novel_search`：关键词搜索（默认，适合中小规模书库）
  - `clarify_missing_info`：澄清工具（模型自动调用）
  - `novel_vector_search`：向量检索（可选，需在 `main.go` 中启用）

- **Workflow** (`workflow/`)
  - `feedback.go`：记录用户反馈，用于后续用户画像更新

**启用向量检索（可选）：**

在 `main.go` 中取消注释以下代码：
```go
if embedFn := newDemoEmbedFn(); embedFn != nil {
    vecTool := tools.NewNovelVectorSearchTool(embedFn)
    toolsList = append(toolsList, vecTool)
}
```

### 3. TODO / 后续扩展方向

代码里已经标了一些关键 TODO，可以按需实现成你的业务版本：

- **向量检索扩展（可选）**
  - `tool/vector_tool.go`：`newQdrantClientFromEnv` 里默认 `http://localhost:6333`。  
    - TODO：通过配置文件/启动参数管理 Qdrant endpoint 和 collection，而不是写死 localhost。
  - `main.go`：`newDemoEmbedFn` 只是一个伪向量生成。  
    - TODO：替换为真实 Embedding 服务（例如 Ark Embedding），并保证向量维度与 Qdrant collection 一致。

- **反馈闭环 & 兴趣向量**
  - `workflow/feedback.go`：当前只是将反馈 append 到 `/tmp/novel_feedback.log`。  
    - TODO：改为写入数据库或消息队列，异步更新用户兴趣向量 / 召回库权重。
  - `graph/loop.go`：当前反馈记录简化了候选列表（因为模型自动处理）。  
    - TODO：从 Tool Calling 历史中提取候选列表，用于更完整的反馈记录。

- **工程化 & 可配置**
  - `tool/vector_tool.go`：  
    - TODO：通过依赖注入传入 `qdrantClient` 和 `EmbedFunc`，方便单测和多环境配置。
  - 可以进一步抽出配置结构体（如 `Config{ Ark, Qdrant, Embedding, FeedbackSink }`），统一管理所有外部依赖。







