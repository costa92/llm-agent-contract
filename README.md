[English](./README.md) | [简体中文](./README.zh-CN.md)

# llm-agent-contract

The capability-aware **LLM-provider contract** for the
[`llm-agent`](https://github.com/costa92/llm-agent) framework — extracted into a
standalone, **stdlib-only** module so every consumer depends on the *interface*
rather than on a concrete provider implementation or on the core framework.

```
module github.com/costa92/llm-agent-contract   (go 1.26, zero third-party requires)
```

## What it owns

A single package, `llm/`, holding the narrow set of types an Agent or Tool needs
to call a model — nothing more:

| Type | Role |
| --- | --- |
| `ChatModel` | base interface every provider implements: `Generate` (one-shot), `Stream` (iterator), `Info` |
| `ToolCaller` | capability: native function-calling. `WithTools` is **immutable** (returns a new value — rejects the mutate-in-place pattern that races) |
| `Embedder` | capability: vector embeddings. Deliberately does **not** embed `ChatModel` (orthogonal) |
| `StructuredOutputs` | capability: JSON-schema-constrained output (`WithSchema`, immutable) |
| `StreamReader` / `StreamEvent` | iterator-style streaming (`Next`/`Close`) + typed event union (text / tool-call / thinking / done) |
| `AccumulateStream` | drains a stream into a flat `Response`; merges tool-call deltas by **`Index`** (the stable K1 key) |
| `Request` / `Response` / `Message` | chat-layer request/response + a single conversation turn |
| `Tool` / `ToolCall` | function-call schema (raw JSON Schema) + invocation |
| `ProviderInfo` / `Capabilities` | bound provider+model identity; `Capabilities` is a JSON-serializable struct for OTel attribute emission |
| `Vector` / `Usage` / `UsageSource` | embeddings + token accounting (reported / estimated / unknown) |
| `FinishReason` | OpenAI-compatible stop reasons |
| error types | `ErrCapabilityNotSupported`, `ErrScriptExhausted` sentinels + typed `AuthError` / `RateLimitError` / `InvalidRequestError` / `TransientError` |
| `ScriptedLLM` / `ChatOnlyMock` | deterministic full-capability mock + ChatModel-only mock for capability-degradation tests |

## Capability negotiation

Callers detect capabilities via **type assertion** (the compile-time signal) **and**
consult `ProviderInfo.Capabilities` (the runtime signal for per-(provider × model)
variation type assertion can't see — e.g. Ollama's Go type implements `ToolCaller`,
but for `llama2` `Capabilities.Tools` is `false`):

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

## Streaming

`StreamReader` is iterator-style rather than channel-based, for explicit
cancellation, single-call error propagation, and no producer-goroutine leaks.
**Consumers MUST `defer sr.Close()`.** Use `AccumulateStream` when you don't care
about streaming granularity.

## Consumers

This module is a **leaf**; nothing in the ecosystem depends *on* a consumer of it
that it also depends back on. It is consumed by:

- `llm-agent` (core framework)
- `llm-agent-providers` (OpenAI / Anthropic / Ollama / DeepSeek / MiniMax adapters)
- `llm-agent-rag`
- `llm-agent-otel`
- `llm-agent-customer-support`
- `llm-agent-flow` (indirect)

## Versioning

Pre-release. While the contract is stabilizing, consumers wire it via a local
`replace github.com/costa92/llm-agent-contract => ../llm-agent-contract` with a
`v0.0.0` placeholder require. Tagging the first release (`v0.x.0`) and replacing
those `replace` directives with pinned `require`s across all consumers is the
remaining migration step (`replace` is rejected on tagged-release branches by the
`INFRA-04` gate).

## Development

```bash
GOWORK=off go vet ./...
GOWORK=off go build ./...
GOWORK=off go test ./... -count=1
```

Stays stdlib-only — adding a third-party `require` is a regression.
