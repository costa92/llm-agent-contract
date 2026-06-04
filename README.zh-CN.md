[English](./README.md) | [简体中文](./README.zh-CN.md)

# llm-agent-contract

面向 [`llm-agent`](https://github.com/costa92/llm-agent) 框架的、具备能力感知的
**LLM 提供方契约（LLM-provider contract）**——它被抽取为一个独立的、**仅标准库**
的 module，使每个消费方都依赖于*接口*，而非依赖某个具体的 provider 实现或核心框架。

```
module github.com/costa92/llm-agent-contract   (go 1.26, zero third-party requires)
```

## 它拥有什么

一个单独的包 `llm/`，仅持有 Agent 或 Tool 调用模型所需的那一小组类型——别无其他：

| 类型 | 角色 |
| --- | --- |
| `ChatModel` | 每个 provider 都要实现的基础接口：`Generate`（一次性）、`Stream`（迭代器）、`Info` |
| `ToolCaller` | 能力：原生函数调用。`WithTools` 是**不可变的**（返回一个新值——拒绝那种会产生竞态的就地修改模式） |
| `Embedder` | 能力：向量嵌入。刻意**不**内嵌 `ChatModel`（两者正交） |
| `StructuredOutputs` | 能力：受 JSON-schema 约束的输出（`WithSchema`，不可变） |
| `StreamReader` / `StreamEvent` | 迭代器风格的流式（`Next`/`Close`）+ 类型化的事件联合（text / tool-call / thinking / done） |
| `AccumulateStream` | 将一个流抽干为扁平的 `Response`；按 **`Index`**（稳定的 K1 键）合并 tool-call 增量 |
| `Request` / `Response` / `Message` | 聊天层的请求/响应 + 单个对话轮次 |
| `Tool` / `ToolCall` | 函数调用 schema（原始 JSON Schema）+ 调用 |
| `ProviderInfo` / `Capabilities` | 绑定的 provider+model 身份；`Capabilities` 是一个可 JSON 序列化的结构体，用于 OTel 属性发射 |
| `Vector` / `Usage` / `UsageSource` | 嵌入 + token 计量（reported / estimated / unknown） |
| `FinishReason` | OpenAI 兼容的停止原因 |
| error 类型 | `ErrCapabilityNotSupported`、`ErrScriptExhausted` 哨兵 + 类型化的 `AuthError` / `RateLimitError` / `InvalidRequestError` / `TransientError` |
| `ScriptedLLM` / `ChatOnlyMock` | 确定性的全能力模拟（mock）+ 仅 ChatModel 的模拟，用于能力降级测试 |

## 能力协商

调用方通过**类型断言**（编译期信号）**并**查询 `ProviderInfo.Capabilities`（运行时信号，
用于类型断言看不到的逐 (provider × model) 差异——例如 Ollama 的 Go 类型实现了 `ToolCaller`，
但对 `llama2` 而言 `Capabilities.Tools` 为 `false`）来检测能力：

```go
if tc, ok := model.(llm.ToolCaller); ok && model.Info().Capabilities.Tools {
    bound, err := tc.WithTools(tools)
    if err != nil {
        return err
    }
    return bound.Generate(ctx, req)
}
// Fall back to scratchpad templating
return model.Generate(ctx, scratchpadReq(req))
```

## 流式

`StreamReader` 采用迭代器风格而非基于 channel，以实现显式取消、单次调用的错误传播，
以及没有生产者 goroutine 泄漏。**消费方必须 `defer sr.Close()`。** 当你不关心流式粒度时，
请使用 `AccumulateStream`。

## 消费方

本 module 是一片**叶子**；生态中没有任何它所消费的、同时又反过来依赖它的消费方。它被以下方消费：

- `llm-agent`（核心框架）
- `llm-agent-providers`（OpenAI / Anthropic / Ollama / DeepSeek / MiniMax 适配器）
- `llm-agent-rag`
- `llm-agent-otel`
- `llm-agent-customer-support`
- `llm-agent-flow`（间接）

## 版本管理

预发布阶段。在契约趋于稳定期间，消费方通过本地的
`replace github.com/costa92/llm-agent-contract => ../llm-agent-contract` 配合一个
`v0.0.0` 占位 require 来接入。打出首个发布 tag（`v0.x.0`）、并在所有消费方中将这些
`replace` 指令替换为锚定的 `require`，是剩余的迁移步骤（`replace` 会被 `INFRA-04`
门禁在已打 tag 的发布分支上拒绝）。

## 开发

```bash
GOWORK=off go vet ./...
GOWORK=off go build ./...
GOWORK=off go test ./... -count=1
```

保持仅标准库——新增第三方 `require` 即为一次回归。
