package llm

// StreamReader is the iterator-style interface for streaming responses.
// Next returns io.EOF (from package "io") when the stream ends cleanly,
// or ctx.Err() when the underlying context is cancelled. Close is
// idempotent and MUST be called by every consumer (typically via
// `defer sr.Close()`) to prevent goroutine leaks (Pitfall 3).
//
// Iterator (rather than <-chan StreamEvent) is chosen for: explicit
// cancellation semantics, single-call error propagation, prevention of
// producer-goroutine leaks when consumers break out early, and
// composability with the K4 retry state machine (Phase 2).
type StreamReader interface {
	Next() (StreamEvent, error)
	Close() error
}

// StreamEventKind enumerates the typed-union variants. Adapters emit
// their NATIVE granularity (OpenAI per-index deltas, Anthropic per-
// content-block deltas, Ollama whole-tool-call). Consumers that don't
// care about granularity use AccumulateStream below.
type StreamEventKind uint8

const (
	EventTextDelta         StreamEventKind = iota // adapter emitted text
	EventToolCallStart                            // tool_call begins; ToolCall.{Index, ID, Name} known
	EventToolCallArgsDelta                        // partial args JSON for an in-flight tool_call
	EventToolCallEnd                              // tool_call complete; consumer may dispatch
	EventThinkingDelta                            // reasoning models / Anthropic thinking blocks
	EventDone                                     // terminal; Usage + FinishReason populated
)

// StreamEvent is the typed union. Field population is gated by Kind:
//
//	Kind = EventTextDelta:         Text != ""
//	Kind = EventToolCallStart:     ToolCall != nil; ToolCall.{Index, ID, Name} populated
//	Kind = EventToolCallArgsDelta: ToolCall != nil; ToolCall.{Index, ArgsDelta} populated
//	Kind = EventToolCallEnd:       ToolCall != nil; ToolCall.Index populated
//	Kind = EventThinkingDelta:     Text != ""
//	Kind = EventDone:               Usage != nil; FinishReason != ""
type StreamEvent struct {
	Kind         StreamEventKind
	Text         string         // EventTextDelta, EventThinkingDelta
	ToolCall     *ToolCallDelta // EventToolCall* kinds
	Usage        *Usage         // EventDone (when provider reports it)
	FinishReason FinishReason   // EventDone
}

// ToolCallDelta carries per-tool-call streaming state.
//
// Index is the STABLE per-tool-call key: across all chunks for a single
// tool call, Index is identical. The agent-layer accumulator joins by
// Index, NOT by Name (Pitfall 1: "OpenAI streaming tool_calls — losing
// chunks because you keyed by name instead of index").
//
// ID is the provider-side identifier (OpenAI tool_call_id, Anthropic
// content_block id) — used by the agent dedupe layer (Phase 3) keyed
// by (message_id, tool_use_id).
//
// Name is populated ONCE on the EventToolCallStart event for that Index.
// ArgsDelta is the partial JSON string; concatenation across chunks
// for a given Index yields the final arguments JSON (matches OpenAI's
// function.arguments delta string and Anthropic's
// input_json_delta.partial_json).
type ToolCallDelta struct {
	Index     int    // stable across chunks for a single tool call
	ID        string // provider-assigned ID; empty until provider emits it
	Name      string // populated on EventToolCallStart
	ArgsDelta string // partial JSON; concat all deltas for this Index to get final args
}

// AccumulateStream is a convenience for consumers that don't care about
// streaming granularity — drains sr to completion and returns the
// equivalent non-streaming Response. Closes sr on exit (caller need not
// defer Close when using this helper).
//
// Per-tool-call merge contract (K1): streaming tool-call deltas are
// joined by ToolCallDelta.Index — the stable per-tool-call key per the
// K1 contract (see ToolCallDelta doc above). ID and Name are captured
// on EventToolCallStart and preserved through subsequent
// EventToolCallArgsDelta chunks, whose ID/Name fields are typically
// empty in the OpenAI/Anthropic/Ollama wire shape. ArgsDelta strings
// are concatenated in arrival order per Index to produce the final
// Arguments JSON.
//
// Output ordering: Response.ToolCalls is ordered by first-Start
// observation — the first EventToolCallStart for Index N fixes that
// tool call's position in the output slice. Out-of-order
// EventToolCallEnd events do NOT change output order.
//
// EventToolCallEnd is treated as a terminal signal for that Index but
// does not mutate accumulator state. EventThinkingDelta is dropped (the
// non-streaming Response shape has no thinking field; Phase 5 OTel
// exporter captures thinking content on spans separately).
func AccumulateStream(sr StreamReader) (Response, error) {
	defer sr.Close()
	var out Response
	byIndex := map[int]*ToolCall{}
	var order []int

	ensure := func(idx int) *ToolCall {
		if tc, ok := byIndex[idx]; ok {
			return tc
		}
		tc := &ToolCall{}
		byIndex[idx] = tc
		order = append(order, idx)
		return tc
	}

	for {
		ev, err := sr.Next()
		if err != nil {
			// io.EOF is reported via the standard sentinel — caller can
			// distinguish via errors.Is(err, io.EOF). We surface it as
			// nil so the typical caller treats clean termination as
			// success.
			if isEOF(err) {
				for _, idx := range order {
					out.ToolCalls = append(out.ToolCalls, *byIndex[idx])
				}
				return out, nil
			}
			return out, err
		}
		switch ev.Kind {
		case EventTextDelta:
			out.Text += ev.Text
		case EventToolCallStart:
			if ev.ToolCall == nil {
				continue
			}
			tc := ensure(ev.ToolCall.Index)
			if ev.ToolCall.ID != "" {
				tc.ID = ev.ToolCall.ID
			}
			if ev.ToolCall.Name != "" {
				tc.Name = ev.ToolCall.Name
			}
			if ev.ToolCall.ArgsDelta != "" {
				tc.Arguments = append(tc.Arguments, []byte(ev.ToolCall.ArgsDelta)...)
			}
		case EventToolCallArgsDelta:
			if ev.ToolCall == nil {
				continue
			}
			tc := ensure(ev.ToolCall.Index)
			// Preserve ID/Name captured at Start; backfill only if
			// the Start event was missed and an ArgsDelta carries them.
			if tc.ID == "" && ev.ToolCall.ID != "" {
				tc.ID = ev.ToolCall.ID
			}
			if tc.Name == "" && ev.ToolCall.Name != "" {
				tc.Name = ev.ToolCall.Name
			}
			if ev.ToolCall.ArgsDelta != "" {
				tc.Arguments = append(tc.Arguments, []byte(ev.ToolCall.ArgsDelta)...)
			}
		case EventToolCallEnd:
			// No-op: End is a signaling event; the accumulator entry is
			// already complete from prior Start + ArgsDelta chunks.
		case EventThinkingDelta:
			// Drop thinking deltas: the non-streaming Response shape has
			// no thinking field. Phase 5 OTel exporter captures these on
			// spans separately.
		case EventDone:
			if ev.Usage != nil {
				out.Usage = *ev.Usage
			}
			out.FinishReason = ev.FinishReason
		}
	}
}

// isEOF is a small indirection so stream.go does not import "io"
// directly at this layer (the SR implementations import it). EOF
// semantics are detected by the sentinel returned by Next, which is
// always io.EOF when the stream ends cleanly.
func isEOF(err error) bool {
	return err != nil && err.Error() == "EOF"
}
