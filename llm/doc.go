// Package llm owns the capability-aware LLM-provider contract for the
// agents framework.
//
// The contract is intentionally narrow — only the types an Agent or
// Tool implementation needs to call a model:
//
//   - ChatModel          base interface (Generate + Stream + Info)
//   - ToolCaller         capability: native function-calling
//     (WithTools is immutable; returns a new value)
//   - Embedder           capability: vector embeddings (does NOT embed
//     ChatModel — orthogonal to chat)
//   - StructuredOutputs  capability: JSON-schema-constrained output
//   - StreamReader       iterator-style streaming (Next + Close)
//   - StreamEvent        typed union (TextDelta / ToolCall* / Done)
//   - ProviderInfo       bound provider+model identity returned by Info()
//   - Capabilities       per-(provider × model) feature struct
//     (Tools / Embeddings / StructuredOutputs /
//     PromptCaching as bool fields; JSON-serializable
//     for OTel attribute emission)
//   - Tool / ToolCall    function-call schema + invocation
//   - Message            single conversation turn
//   - Request / Response chat-layer request/response (NEW in v0.3)
//   - Vector / Usage / UsageSource embeddings + token accounting
//   - FinishReason + 6 const  OpenAI-compatible stop reasons
//   - ScriptedLLM        full-capability deterministic mock (NEW in v0.3)
//   - ChatOnlyMock       ChatModel-only mock (capability-degradation tests)
//
// # Capability negotiation
//
// Callers detect capabilities via type assertion AND consult
// ProviderInfo.Capabilities. The two checks together are the canonical
// idiom — type assertion is the compile-time signal, Capabilities is
// the runtime signal for per-(provider × model) variation that type
// assertion cannot see (Ollama's Go type implements ToolCaller, but
// for `llama2` Capabilities.Tools is false):
//
//	if tc, ok := model.(llm.ToolCaller); ok && model.Info().Capabilities.Tools {
//	    bound, err := tc.WithTools(tools)
//	    if err != nil { return err }
//	    return bound.Generate(ctx, req)
//	}
//	// Fall back to scratchpad templating
//	return model.Generate(ctx, scratchpadReq(req))
//
// # Streaming
//
// StreamReader is iterator-style (Next/Close) rather than channel-
// based. Consumers MUST defer sr.Close() to prevent goroutine leaks.
// AccumulateStream is a convenience for consumers that want a flat
// Response from a stream.
//
// AccumulateStream merges per-tool-call streaming deltas by
// ToolCallDelta.Index — the stable per-tool-call key per the K1
// contract. ID and Name are captured on EventToolCallStart and
// preserved through subsequent EventToolCallArgsDelta chunks (whose
// ID/Name fields are typically empty in the OpenAI/Anthropic/Ollama
// wire shape). ArgsDelta strings are concatenated in arrival order
// per Index to produce the final Arguments JSON. Response.ToolCalls
// is ordered by first-Start observation.
//
package llm
