package llm

import "context"

// ChatModel is the base contract every provider implements. It is the
// smallest possible interface: Generate (one-shot), Stream (iterator),
// Info (per-(provider × model) identity).
//
// Capabilities beyond plain text generation are expressed as separate
// interfaces (ToolCaller, Embedder, StructuredOutputs); callers detect
// them via type assertion. ProviderInfo.Capabilities is the runtime
// signal for per-(provider × model) variation that type assertion
// cannot see — see doc.go for the canonical negotiation idiom.
//
// All implementations MUST be safe for concurrent use; concurrent
// Generate / Stream calls on the same value are part of the contract.
type ChatModel interface {
	Generate(ctx context.Context, req Request) (Response, error)
	Stream(ctx context.Context, req Request) (StreamReader, error)
	Info() ProviderInfo
}
