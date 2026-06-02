package llm

// These tests pin the Phase 2 contract for AccumulateStream's per-tool-call
// merge: deltas are joined by ToolCallDelta.Index (the stable per-tool-call
// key per the K1 contract), NOT by ID. The previous helper keyed by ID,
// which silently dropped ArgsDelta chunks whose ID field was empty — the
// standard OpenAI/Anthropic/Ollama wire shape where ID is populated only
// on the EventToolCallStart event.

import (
	"io"
	"testing"
)

// sliceStreamReader is a deterministic StreamReader for replaying a fixed
// sequence of StreamEvents. After all events are returned, Next() returns
// io.EOF. Close is idempotent; once closed, Next() returns io.EOF.
type sliceStreamReader struct {
	events []StreamEvent
	idx    int
	closed bool
}

func (r *sliceStreamReader) Next() (StreamEvent, error) {
	if r.closed {
		return StreamEvent{}, io.EOF
	}
	if r.idx >= len(r.events) {
		return StreamEvent{}, io.EOF
	}
	ev := r.events[r.idx]
	r.idx++
	return ev, nil
}

func (r *sliceStreamReader) Close() error {
	r.closed = true
	return nil
}

// ----- 1. OpenAI shape: Start carries ID, subsequent ArgsDelta chunks have empty ID -----
// This is the bug-case the prior ID-keyed merge dropped on the floor.
func TestAccumulateStream_ToolCalls_OpenAIShape(t *testing.T) {
	sr := &sliceStreamReader{events: []StreamEvent{
		{Kind: EventToolCallStart, ToolCall: &ToolCallDelta{Index: 0, ID: "call_1", Name: "add"}},
		{Kind: EventToolCallArgsDelta, ToolCall: &ToolCallDelta{Index: 0, ArgsDelta: `{"a":1`}},
		{Kind: EventToolCallArgsDelta, ToolCall: &ToolCallDelta{Index: 0, ArgsDelta: `,"b":2}`}},
		{Kind: EventToolCallEnd, ToolCall: &ToolCallDelta{Index: 0}},
		{Kind: EventDone, Usage: &Usage{InputTokens: 5, OutputTokens: 7, TotalTokens: 12, Source: UsageReported}, FinishReason: FinishReasonToolCalls},
	}}

	resp, err := AccumulateStream(sr)
	if err != nil {
		t.Fatalf("AccumulateStream: %v", err)
	}
	if got := len(resp.ToolCalls); got != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", got)
	}
	tc := resp.ToolCalls[0]
	if tc.ID != "call_1" {
		t.Errorf("ToolCalls[0].ID = %q, want %q", tc.ID, "call_1")
	}
	if tc.Name != "add" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", tc.Name, "add")
	}
	if string(tc.Arguments) != `{"a":1,"b":2}` {
		t.Errorf("ToolCalls[0].Arguments = %q, want %q", string(tc.Arguments), `{"a":1,"b":2}`)
	}
	if resp.FinishReason != FinishReasonToolCalls {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, FinishReasonToolCalls)
	}
	if resp.Usage.TotalTokens != 12 {
		t.Errorf("Usage.TotalTokens = %d, want 12", resp.Usage.TotalTokens)
	}
}

// ----- 2. Parallel tool calls with distinct IDs, interleaved ArgsDelta chunks with empty ID -----
func TestAccumulateStream_ToolCalls_ParallelDistinctIDs(t *testing.T) {
	sr := &sliceStreamReader{events: []StreamEvent{
		{Kind: EventToolCallStart, ToolCall: &ToolCallDelta{Index: 0, ID: "call_calc", Name: "calc"}},
		{Kind: EventToolCallStart, ToolCall: &ToolCallDelta{Index: 1, ID: "call_search", Name: "search"}},
		{Kind: EventToolCallArgsDelta, ToolCall: &ToolCallDelta{Index: 0, ArgsDelta: `{"x":`}},
		{Kind: EventToolCallArgsDelta, ToolCall: &ToolCallDelta{Index: 1, ArgsDelta: `{"q":"go"`}},
		{Kind: EventToolCallArgsDelta, ToolCall: &ToolCallDelta{Index: 0, ArgsDelta: `42}`}},
		{Kind: EventToolCallArgsDelta, ToolCall: &ToolCallDelta{Index: 1, ArgsDelta: `}`}},
		{Kind: EventToolCallEnd, ToolCall: &ToolCallDelta{Index: 0}},
		{Kind: EventToolCallEnd, ToolCall: &ToolCallDelta{Index: 1}},
		{Kind: EventDone, FinishReason: FinishReasonToolCalls},
	}}

	resp, err := AccumulateStream(sr)
	if err != nil {
		t.Fatalf("AccumulateStream: %v", err)
	}
	if got := len(resp.ToolCalls); got != 2 {
		t.Fatalf("len(ToolCalls) = %d, want 2", got)
	}
	// Order is first-Start observation: Index 0 then Index 1.
	if resp.ToolCalls[0].ID != "call_calc" || resp.ToolCalls[0].Name != "calc" {
		t.Errorf("ToolCalls[0] = {ID:%q, Name:%q}, want {call_calc, calc}", resp.ToolCalls[0].ID, resp.ToolCalls[0].Name)
	}
	if string(resp.ToolCalls[0].Arguments) != `{"x":42}` {
		t.Errorf("ToolCalls[0].Arguments = %q, want %q", string(resp.ToolCalls[0].Arguments), `{"x":42}`)
	}
	if resp.ToolCalls[1].ID != "call_search" || resp.ToolCalls[1].Name != "search" {
		t.Errorf("ToolCalls[1] = {ID:%q, Name:%q}, want {call_search, search}", resp.ToolCalls[1].ID, resp.ToolCalls[1].Name)
	}
	if string(resp.ToolCalls[1].Arguments) != `{"q":"go"}` {
		t.Errorf("ToolCalls[1].Arguments = %q, want %q", string(resp.ToolCalls[1].Arguments), `{"q":"go"}`)
	}
}

// ----- 3. Ollama ID==Name fallback: two parallel calls with the same ID/Name at distinct Indexes -----
// The prior ID-keyed merge would collapse both into one entry; the
// Index-keyed merge preserves them as two ToolCalls.
func TestAccumulateStream_ToolCalls_OllamaIDIsName(t *testing.T) {
	sr := &sliceStreamReader{events: []StreamEvent{
		{Kind: EventToolCallStart, ToolCall: &ToolCallDelta{Index: 0, ID: "lookup", Name: "lookup"}},
		{Kind: EventToolCallArgsDelta, ToolCall: &ToolCallDelta{Index: 0, ArgsDelta: `{"key":"a"}`}},
		{Kind: EventToolCallEnd, ToolCall: &ToolCallDelta{Index: 0}},
		{Kind: EventToolCallStart, ToolCall: &ToolCallDelta{Index: 1, ID: "lookup", Name: "lookup"}},
		{Kind: EventToolCallArgsDelta, ToolCall: &ToolCallDelta{Index: 1, ArgsDelta: `{"key":"b"}`}},
		{Kind: EventToolCallEnd, ToolCall: &ToolCallDelta{Index: 1}},
		{Kind: EventDone, FinishReason: FinishReasonToolCalls},
	}}

	resp, err := AccumulateStream(sr)
	if err != nil {
		t.Fatalf("AccumulateStream: %v", err)
	}
	if got := len(resp.ToolCalls); got != 2 {
		t.Fatalf("len(ToolCalls) = %d, want 2 (Ollama same-Name parallel calls must NOT collapse)", got)
	}
	if string(resp.ToolCalls[0].Arguments) != `{"key":"a"}` {
		t.Errorf("ToolCalls[0].Arguments = %q, want %q", string(resp.ToolCalls[0].Arguments), `{"key":"a"}`)
	}
	if string(resp.ToolCalls[1].Arguments) != `{"key":"b"}` {
		t.Errorf("ToolCalls[1].Arguments = %q, want %q", string(resp.ToolCalls[1].Arguments), `{"key":"b"}`)
	}
}

// ----- 4. Out-of-order Ends: output order is first-Start observation -----
func TestAccumulateStream_ToolCalls_OutOfOrderEnds(t *testing.T) {
	sr := &sliceStreamReader{events: []StreamEvent{
		{Kind: EventToolCallStart, ToolCall: &ToolCallDelta{Index: 0, ID: "first", Name: "f"}},
		{Kind: EventToolCallStart, ToolCall: &ToolCallDelta{Index: 1, ID: "second", Name: "s"}},
		{Kind: EventToolCallArgsDelta, ToolCall: &ToolCallDelta{Index: 1, ArgsDelta: `{"s":1}`}},
		{Kind: EventToolCallArgsDelta, ToolCall: &ToolCallDelta{Index: 0, ArgsDelta: `{"f":1}`}},
		{Kind: EventToolCallEnd, ToolCall: &ToolCallDelta{Index: 1}}, // End for 1 first
		{Kind: EventToolCallEnd, ToolCall: &ToolCallDelta{Index: 0}}, // then End for 0
		{Kind: EventDone, FinishReason: FinishReasonToolCalls},
	}}

	resp, err := AccumulateStream(sr)
	if err != nil {
		t.Fatalf("AccumulateStream: %v", err)
	}
	if got := len(resp.ToolCalls); got != 2 {
		t.Fatalf("len(ToolCalls) = %d, want 2", got)
	}
	if resp.ToolCalls[0].ID != "first" {
		t.Errorf("ToolCalls[0].ID = %q, want %q (first-Start order, not first-End)", resp.ToolCalls[0].ID, "first")
	}
	if resp.ToolCalls[1].ID != "second" {
		t.Errorf("ToolCalls[1].ID = %q, want %q", resp.ToolCalls[1].ID, "second")
	}
}

// ----- 5. No ArgsDelta: tool with zero arguments -----
// Pins current default (nil Arguments for a tool that never emitted any
// ArgsDelta event). Callers reading Arguments must handle nil as "{}".
func TestAccumulateStream_ToolCalls_NoArgsDelta(t *testing.T) {
	sr := &sliceStreamReader{events: []StreamEvent{
		{Kind: EventToolCallStart, ToolCall: &ToolCallDelta{Index: 0, ID: "call_noop", Name: "noop"}},
		{Kind: EventToolCallEnd, ToolCall: &ToolCallDelta{Index: 0}},
		{Kind: EventDone, FinishReason: FinishReasonToolCalls},
	}}

	resp, err := AccumulateStream(sr)
	if err != nil {
		t.Fatalf("AccumulateStream: %v", err)
	}
	if got := len(resp.ToolCalls); got != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", got)
	}
	if resp.ToolCalls[0].Arguments != nil {
		t.Errorf("ToolCalls[0].Arguments = %q, want nil (never received an ArgsDelta event)", string(resp.ToolCalls[0].Arguments))
	}
	if resp.ToolCalls[0].Name != "noop" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", resp.ToolCalls[0].Name, "noop")
	}
}

// ----- 6. Defensive: Start with empty ID, name only -----
// Some adapters might not have an ID at Start time (rare, but covered).
// The accumulator must still produce a ToolCall entry; ID stays empty.
func TestAccumulateStream_ToolCalls_EmptyIDThroughout(t *testing.T) {
	sr := &sliceStreamReader{events: []StreamEvent{
		{Kind: EventToolCallStart, ToolCall: &ToolCallDelta{Index: 0, ID: "", Name: "calc"}},
		{Kind: EventToolCallArgsDelta, ToolCall: &ToolCallDelta{Index: 0, ArgsDelta: `{"x":1}`}},
		{Kind: EventToolCallEnd, ToolCall: &ToolCallDelta{Index: 0}},
		{Kind: EventDone, FinishReason: FinishReasonToolCalls},
	}}

	resp, err := AccumulateStream(sr)
	if err != nil {
		t.Fatalf("AccumulateStream: %v", err)
	}
	if got := len(resp.ToolCalls); got != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1 (empty-ID Start must still create an entry)", got)
	}
	if resp.ToolCalls[0].ID != "" {
		t.Errorf("ToolCalls[0].ID = %q, want empty", resp.ToolCalls[0].ID)
	}
	if resp.ToolCalls[0].Name != "calc" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", resp.ToolCalls[0].Name, "calc")
	}
	if string(resp.ToolCalls[0].Arguments) != `{"x":1}` {
		t.Errorf("ToolCalls[0].Arguments = %q, want %q", string(resp.ToolCalls[0].Arguments), `{"x":1}`)
	}
}

// ----- 7. Text-only stream backward-compat smoke -----
func TestAccumulateStream_Text_Unchanged(t *testing.T) {
	sr := &sliceStreamReader{events: []StreamEvent{
		{Kind: EventTextDelta, Text: "hello "},
		{Kind: EventTextDelta, Text: "world"},
		{Kind: EventTextDelta, Text: "!"},
		{Kind: EventDone, Usage: &Usage{InputTokens: 3, OutputTokens: 4, Source: UsageReported}, FinishReason: FinishReasonStop},
	}}

	resp, err := AccumulateStream(sr)
	if err != nil {
		t.Fatalf("AccumulateStream: %v", err)
	}
	if resp.Text != "hello world!" {
		t.Errorf("Text = %q, want %q", resp.Text, "hello world!")
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("ToolCalls len = %d, want 0", len(resp.ToolCalls))
	}
	if resp.FinishReason != FinishReasonStop {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, FinishReasonStop)
	}
	if resp.Usage.OutputTokens != 4 {
		t.Errorf("Usage.OutputTokens = %d, want 4", resp.Usage.OutputTokens)
	}
}
