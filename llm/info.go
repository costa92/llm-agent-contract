package llm

// ProviderInfo describes a bound provider+model combination. Returned
// by ChatModel.Info(). Capabilities reflect THIS bound model, not the
// provider type generically (Pitfall 6). Provider instances bind a
// model at construction time — `openai.New(openai.WithModel("gpt-4o"))`
// — so Info() is constant for the lifetime of the value.
type ProviderInfo struct {
	Provider     string       `json:"provider"`     // "openai", "anthropic", "ollama", "deepseek", "minimax"
	Model        string       `json:"model"`        // "gpt-4o-mini", "claude-3-5-haiku", "llama3.1:8b"
	Capabilities Capabilities `json:"capabilities"`
}

// Capabilities is a value type — JSON-serializable for OTel attribute
// emission (gen_ai.provider.capabilities.* in Phase 5). Per D-02, this
// is a struct (NOT methods, NOT a bitmask): self-documenting in test
// failures, extensible with non-bool fields later (e.g.,
// MaxToolsPerCall int) without breaking JSON consumers.
//
// Type assertion remains the PRIMARY signal at compile time
// (`if tc, ok := model.(ToolCaller); ok { ... }`); Capabilities is the
// RUNTIME signal for per-(provider × model) variation that type
// assertion cannot see — Ollama's Go type implements ToolCaller, but
// for `llama2` the Capabilities.Tools bool is false.
type Capabilities struct {
	Tools             bool `json:"tools"`               // Native function-calling supported by the bound model
	Embeddings        bool `json:"embeddings"`          // Embed() returns vectors (NOT ErrCapabilityNotSupported)
	StructuredOutputs bool `json:"structured_outputs"`  // WithSchema() applies a JSON schema constraint
	PromptCaching     bool `json:"prompt_caching"`      // Anthropic explicit / OpenAI auto (consumed Phase 5+)
}
