package llm

import "context"

// ToolCaller is the capability for native tool/function-calling.
// WithTools is IMMUTABLE: it returns a new ToolCaller bound to the
// given tools; the receiver is unchanged. This rejects Eino's
// deprecated BindTools mutation pattern (concurrent calls on the same
// model with different tool sets would otherwise race).
//
// Implementations MUST satisfy ChatModel — a tool-bound model is still
// a ChatModel that can Generate / Stream.
type ToolCaller interface {
	ChatModel
	WithTools(tools []Tool) (ToolCaller, error)
}

// Embedder is the capability for vector embeddings. Returns vectors in
// input order with len(vectors) == len(texts). Providers without
// embedding endpoints (Anthropic, in v0.3) do NOT implement this
// interface; callers detect via type assertion AND consult
// Capabilities.Embeddings on the bound ProviderInfo.
//
// Embedder deliberately does NOT embed ChatModel. A pure embedding-only
// adapter (e.g., a future voyageai adapter) might implement Embedder
// without ChatModel — orthogonality preserves that option.
type Embedder interface {
	Embed(ctx context.Context, texts []string) (vectors []Vector, usage Usage, err error)
	EmbedDimensions() int
}

// StructuredOutputs is the capability for JSON-schema-constrained
// generation (OpenAI response_format, Anthropic tool-as-output trick).
//
// WithSchema is IMMUTABLE — like WithTools — and returns ChatModel
// (NOT StructuredOutputs): re-applying a schema is meaningless, so the
// return type signals that the value is now schema-bound and a second
// WithSchema call is not the intended call shape.
type StructuredOutputs interface {
	ChatModel
	WithSchema(schema []byte) (ChatModel, error)
}
