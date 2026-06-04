package llm

import "encoding/json"

// Request is the chat-layer request type.
//
// It is messages-first, with SystemPrompt lifted out so providers that
// support a top-level system field can map cleanly without inventing a
// fake message turn.
type Request struct {
	Messages        []Message      `json:"messages"`                    // multi-turn dialog (preferred over Prompt)
	SystemPrompt    string         `json:"system_prompt,omitempty"`     // lifted out of Messages for Anthropic top-level system
	MaxOutputTokens int            `json:"max_output_tokens,omitempty"` // 0 = use provider default
	Temperature     *float32       `json:"temperature,omitempty"`       // pointer: nil = use provider default
	Metadata        map[string]any `json:"metadata,omitempty"`          // provider-specific extras (rare; prefer typed)
}

// Response is the chat-layer response type.
type Response struct {
	Text         string       `json:"text"`
	FinishReason FinishReason `json:"finish_reason,omitempty"`
	Provider     string       `json:"provider"`
	Model        string       `json:"model,omitempty"`
	Usage        Usage        `json:"usage"`
	ToolCalls    []ToolCall   `json:"tool_calls,omitempty"`
}

// Message is a single turn in a conversation.
type Message struct {
	Role    string         `json:"role"`             // "user", "assistant", "tool", "system"
	Content string         `json:"content"`          // text content of the turn
	Images  []MessageImage `json:"images,omitempty"` // images attached to a user turn for vision models; ignored by text-only models (Capabilities.Vision == false)
}

// MessageImage is one image attached to a user Message for vision
// (image-understanding) models. Exactly one of URL or Bytes is set; the
// provider chooses delivery (data URI vs hosted link). Providers whose
// bound model lacks vision (Capabilities.Vision == false) ignore these,
// so attaching images to a text-only model is a silent no-op, not an error.
type MessageImage struct {
	URL      string `json:"url,omitempty"`       // http(s) or data: URI; some providers (e.g. Moonshot/Kimi) accept only data: base64
	Bytes    []byte `json:"bytes,omitempty"`     // raw image bytes; the provider encodes to a data: URI
	MimeType string `json:"mime_type,omitempty"` // e.g. "image/png"; required alongside Bytes
	Detail   string `json:"detail,omitempty"`    // vision detail hint "low"/"high"/"auto"; ignored if the provider does not support it
}

// Tool declares a function the model may call. Parameters is a raw
// JSON Schema document — this package doesn't validate it (the
// upstream provider does) so callers can use whatever schema dialect
// their provider expects.
//
// Shape is intentionally simple and provider-agnostic.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ToolCall is what the model returns when it decides to invoke a Tool.
// The ID field is provider-assigned and is used by the agent dedupe
// layer as one half of the (message_id, tool_use_id) key.
type ToolCall struct {
	ID        string          `json:"id,omitempty"` // provider-assigned
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// Vector is one embedding. Length matches Embedder.EmbedDimensions().
type Vector []float32

// Usage carries token accounting for one request. Source distinguishes
// reported (provider returned actual counts), estimated (computed from
// tokenizer), and unknown (mid-stream abort, no usage available).
//
// Source != "" is an invariant after Phase 2 lands (K4); for Phase 0
// the Source field exists but defaults to UsageUnknown when the zero
// value is used.
type Usage struct {
	InputTokens  int         `json:"input_tokens"`
	OutputTokens int         `json:"output_tokens"`
	TotalTokens  int         `json:"total_tokens,omitempty"`
	Source       UsageSource `json:"source,omitempty"`
}

// UsageSource enumerates the provenance of token counts in a Usage.
// Reported = provider returned actual counts; Estimated = computed
// from a tokenizer; Unknown = mid-stream abort, no usage available
// (NOT zero-tokens — Pitfall 5).
type UsageSource string

const (
	UsageReported  UsageSource = "reported"
	UsageEstimated UsageSource = "estimated"
	UsageUnknown   UsageSource = "unknown"
)

// FinishReason mirrors common provider stop reasons.
type FinishReason string

// FinishReason constants mirror the OpenAI /v1/chat/completions
// stop_reason field so providers can pass them through unchanged.
const (
	FinishReasonStop          FinishReason = "stop"
	FinishReasonLength        FinishReason = "length"
	FinishReasonContentFilter FinishReason = "content_filter"
	FinishReasonToolCalls     FinishReason = "tool_calls"
	FinishReasonFunctionCall  FinishReason = "function_call"
	FinishReasonUnknown       FinishReason = "unknown"
)
