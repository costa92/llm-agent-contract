package llm

// P-1 (flow v2 prerequisite): AccumulateStream must detect a *wrapped* io.EOF,
// not just the bare sentinel. A StreamReader that wraps EOF for context
// (fmt.Errorf("...: %w", io.EOF)) is a legal terminator per the Next contract;
// the old err.Error()=="EOF" string compare silently failed it, turning a
// clean end into a propagated error. The flow v2 concatenate adapter drains
// provider streams through AccumulateStream, so this must be robust.

import (
	"fmt"
	"io"
	"testing"
)

// wrappedEOFReader returns its events, then terminates with a WRAPPED io.EOF
// instead of the bare sentinel.
type wrappedEOFReader struct {
	events []StreamEvent
	idx    int
}

func (r *wrappedEOFReader) Next() (StreamEvent, error) {
	if r.idx >= len(r.events) {
		return StreamEvent{}, fmt.Errorf("provider closed: %w", io.EOF)
	}
	ev := r.events[r.idx]
	r.idx++
	return ev, nil
}

func (r *wrappedEOFReader) Close() error { return nil }

func TestAccumulateStream_WrappedEOF(t *testing.T) {
	sr := &wrappedEOFReader{events: []StreamEvent{
		{Kind: EventTextDelta, Text: "hello "},
		{Kind: EventTextDelta, Text: "world"},
		{Kind: EventDone, FinishReason: FinishReasonStop},
	}}

	resp, err := AccumulateStream(sr)
	if err != nil {
		t.Fatalf("wrapped io.EOF must be treated as a clean end, got error: %v", err)
	}
	if resp.Text != "hello world" {
		t.Fatalf("Text = %q, want %q", resp.Text, "hello world")
	}
}
