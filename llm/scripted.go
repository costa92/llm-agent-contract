package llm

import (
	"context"
	"fmt"
	"io"
	"sync"
)

// ScriptedLLM is a deterministic full-capability mock. It implements
// ChatModel + ToolCaller + Embedder + StructuredOutputs and is used
// across the umbrella as the canonical reference: agent unit tests
// (this repo), conformance baseline (sister repos, Phase 1), example
// programs.
//
// Construction is via functional options:
//
//	m := llm.NewScriptedLLM(
//	    llm.WithProvider("scripted"),
//	    llm.WithModel("test-1"),
//	    llm.WithCapabilities(llm.Capabilities{Tools: true, Embeddings: true}),
//	    llm.WithResponses(
//	        llm.TextResponse("hello"),
//	        llm.ToolCallResponse("calc", `{"a":2,"b":3}`),
//	    ),
//	)
//
// Concurrent-safe: the cursor is protected by sync.Mutex.
type ScriptedLLM struct {
	mu       sync.Mutex
	provider string
	model    string
	caps     Capabilities
	cursor   int
	resps    []Response
	embedDim int
	tools    []Tool // bound by WithTools (returns new ScriptedLLM)
}

// Compile-time interface satisfaction. Placed in production code (not
// only in tests) so capability claims are part of the published API
// surface visible via godoc.
var (
	_ ChatModel         = (*ScriptedLLM)(nil)
	_ ToolCaller        = (*ScriptedLLM)(nil)
	_ Embedder          = (*ScriptedLLM)(nil)
	_ StructuredOutputs = (*ScriptedLLM)(nil)
)

// NewScriptedLLM constructs a ScriptedLLM with functional options.
// Default Capabilities are ALL TRUE (full-capability default; for
// capability-degradation testing use ChatOnlyMock instead).
func NewScriptedLLM(opts ...ScriptedOption) *ScriptedLLM {
	s := &ScriptedLLM{
		provider: "scripted",
		model:    "test",
		caps:     Capabilities{Tools: true, Embeddings: true, StructuredOutputs: true, PromptCaching: false},
		embedDim: 4,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Generate returns the next scripted Response or ErrScriptExhausted.
func (s *ScriptedLLM) Generate(_ context.Context, _ Request) (Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cursor >= len(s.resps) {
		s.cursor++
		return Response{}, fmt.Errorf("scripted: %w", ErrScriptExhausted)
	}
	r := s.resps[s.cursor]
	s.cursor++
	return r, nil
}

// Stream synthesises a streaming view of the next scripted Response.
// Emits EventTextDelta (if Text != "") then EventDone with Usage and
// FinishReason populated from the Response.
func (s *ScriptedLLM) Stream(_ context.Context, _ Request) (StreamReader, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cursor >= len(s.resps) {
		s.cursor++
		return nil, fmt.Errorf("scripted: %w", ErrScriptExhausted)
	}
	r := s.resps[s.cursor]
	s.cursor++
	return newScriptedStream(r), nil
}

// Info returns the bound provider/model + Capabilities.
func (s *ScriptedLLM) Info() ProviderInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return ProviderInfo{Provider: s.provider, Model: s.model, Capabilities: s.caps}
}

// WithTools returns a NEW *ScriptedLLM with tools bound (immutable —
// Pattern 2). The receiver is unchanged; safe to call concurrently.
func (s *ScriptedLLM) WithTools(tools []Tool) (ToolCaller, error) {
	s.mu.Lock()
	provider := s.provider
	model := s.model
	caps := s.caps
	cursor := s.cursor
	resps := append([]Response(nil), s.resps...)
	embedDim := s.embedDim
	s.mu.Unlock()
	cp := &ScriptedLLM{
		provider: provider,
		model:    model,
		caps:     caps,
		cursor:   cursor,
		resps:    resps,
		embedDim: embedDim,
		tools:    append([]Tool(nil), tools...),
	}
	return cp, nil
}

// Embed returns deterministic per-text vectors of EmbedDimensions
// length. If WithEmbeds was used to script per-call vectors, those are
// returned in cursor order (independent cursor from Generate).
func (s *ScriptedLLM) Embed(_ context.Context, texts []string) ([]Vector, Usage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Vector, len(texts))
	for i := range texts {
		out[i] = make(Vector, s.embedDim)
		// Deterministic content: fill with float32(i+1)/10 so vectors
		// differ between texts and tests can assert ordering.
		for j := range out[i] {
			out[i][j] = float32(i+1) / 10
		}
	}
	usage := Usage{InputTokens: len(texts), OutputTokens: 0, TotalTokens: len(texts), Source: UsageReported}
	return out, usage, nil
}

// EmbedDimensions returns the bound vector dimension. Defaults to 4
// (small for fast tests) unless overridden via WithEmbedDimensions.
func (s *ScriptedLLM) EmbedDimensions() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.embedDim
}

// WithSchema is honored as a no-op (the mock does not validate JSON
// schemas) but returns a NEW *ScriptedLLM (immutable). Returning
// ChatModel matches the StructuredOutputs interface signature.
func (s *ScriptedLLM) WithSchema(_ []byte) (ChatModel, error) {
	s.mu.Lock()
	provider := s.provider
	model := s.model
	caps := s.caps
	cursor := s.cursor
	resps := append([]Response(nil), s.resps...)
	embedDim := s.embedDim
	tools := append([]Tool(nil), s.tools...)
	s.mu.Unlock()
	cp := &ScriptedLLM{
		provider: provider,
		model:    model,
		caps:     caps,
		cursor:   cursor,
		resps:    resps,
		embedDim: embedDim,
		tools:    tools,
	}
	return cp, nil
}

// ScriptedOption configures a ScriptedLLM at construction time.
type ScriptedOption func(*ScriptedLLM)

// WithProvider sets the Provider field returned by Info().
func WithProvider(p string) ScriptedOption { return func(s *ScriptedLLM) { s.provider = p } }

// WithModel sets the Model field returned by Info().
func WithModel(m string) ScriptedOption { return func(s *ScriptedLLM) { s.model = m } }

// WithCapabilities sets the Capabilities returned by Info().
func WithCapabilities(c Capabilities) ScriptedOption { return func(s *ScriptedLLM) { s.caps = c } }

// WithResponses appends scripted Responses; Generate/Stream consume in order.
func WithResponses(rs ...Response) ScriptedOption {
	return func(s *ScriptedLLM) { s.resps = append(s.resps, rs...) }
}

// WithEmbedDimensions overrides the EmbedDimensions return value.
func WithEmbedDimensions(d int) ScriptedOption {
	return func(s *ScriptedLLM) { s.embedDim = d }
}

// TextResponse is a convenience constructor for plain-text responses
// ending in FinishReasonStop.
func TextResponse(text string) Response {
	return Response{Text: text, FinishReason: FinishReasonStop, Provider: "scripted"}
}

// ToolCallResponse builds a tool-call response (FinishReasonToolCalls)
// for the given tool name and JSON arguments string.
func ToolCallResponse(name, argsJSON string) Response {
	return Response{
		FinishReason: FinishReasonToolCalls,
		Provider:     "scripted",
		ToolCalls:    []ToolCall{{Name: name, Arguments: []byte(argsJSON)}},
	}
}

// scriptedStream is a tiny StreamReader that emits one EventTextDelta
// (if Text != "") followed by EventDone, then io.EOF on subsequent
// Next calls. Close is idempotent.
type scriptedStream struct {
	mu     sync.Mutex
	r      Response
	step   int
	closed bool
}

func newScriptedStream(r Response) StreamReader {
	return &scriptedStream{r: r}
}

func (s *scriptedStream) Next() (StreamEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return StreamEvent{}, io.EOF
	}
	switch s.step {
	case 0:
		s.step++
		if s.r.Text != "" {
			return StreamEvent{Kind: EventTextDelta, Text: s.r.Text}, nil
		}
		// fall through to Done if no text
		fallthrough
	case 1:
		s.step = 2
		usage := s.r.Usage
		return StreamEvent{Kind: EventDone, Usage: &usage, FinishReason: s.r.FinishReason}, nil
	default:
		return StreamEvent{}, io.EOF
	}
}

func (s *scriptedStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}
