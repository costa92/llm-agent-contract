package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
)

// ----- ChatOnlyMock negative capability assertions -----
func TestChatOnlyMockExcludesCapabilities(t *testing.T) {
	var m ChatModel = &ChatOnlyMock{Provider: "test", Model: "m"}
	if _, ok := m.(ToolCaller); ok {
		t.Fatal("ChatOnlyMock must not implement ToolCaller")
	}
	if _, ok := m.(Embedder); ok {
		t.Fatal("ChatOnlyMock must not implement Embedder")
	}
	if _, ok := m.(StructuredOutputs); ok {
		t.Fatal("ChatOnlyMock must not implement StructuredOutputs")
	}
	info := m.Info()
	if info.Capabilities.Tools || info.Capabilities.Embeddings ||
		info.Capabilities.StructuredOutputs || info.Capabilities.PromptCaching {
		t.Errorf("ChatOnlyMock.Info().Capabilities = %+v, want all-false", info.Capabilities)
	}
}

// ----- ScriptedLLM happy paths -----
func TestScriptedLLM_Capabilities(t *testing.T) {
	ctx := context.Background()
	m := NewScriptedLLM(
		WithProvider("scripted"),
		WithModel("test-1"),
		WithResponses(TextResponse("hello"), TextResponse("world")),
	)

	// Generate path
	r, err := m.Generate(ctx, Request{})
	if err != nil {
		t.Fatalf("Generate#1: %v", err)
	}
	if r.Text != "hello" {
		t.Errorf("Generate#1 Text = %q, want %q", r.Text, "hello")
	}

	// Stream path
	sr, err := m.Stream(ctx, Request{})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	t.Cleanup(func() { _ = sr.Close() })
	resp, err := AccumulateStream(sr)
	if err != nil {
		t.Fatalf("AccumulateStream: %v", err)
	}
	if resp.Text != "world" {
		t.Errorf("AccumulateStream Text = %q, want %q", resp.Text, "world")
	}

	// Exhaustion
	_, err = m.Generate(ctx, Request{})
	if !errors.Is(err, ErrScriptExhausted) {
		t.Errorf("expected ErrScriptExhausted, got %v", err)
	}

	// Embed
	em := NewScriptedLLM(WithEmbedDimensions(8))
	vecs, usage, err := em.Embed(ctx, []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vecs) != 3 {
		t.Errorf("Embed len = %d, want 3", len(vecs))
	}
	if em.EmbedDimensions() != 8 {
		t.Errorf("EmbedDimensions = %d, want 8", em.EmbedDimensions())
	}
	if len(vecs[0]) != 8 {
		t.Errorf("Embed[0] dim = %d, want 8", len(vecs[0]))
	}
	if usage.Source != UsageReported {
		t.Errorf("Embed Usage.Source = %q, want %q", usage.Source, UsageReported)
	}

	// WithSchema (returns ChatModel — schema-bound)
	sm := NewScriptedLLM(WithResponses(TextResponse("schema")))
	bound, err := sm.WithSchema([]byte(`{"type":"object"}`))
	if err != nil {
		t.Fatalf("WithSchema: %v", err)
	}
	if _, ok := bound.(ChatModel); !ok {
		t.Fatal("WithSchema must return a ChatModel")
	}
}

// ----- ToolCaller immutability + concurrency -----
func TestToolCallerImmutable(t *testing.T) {
	ctx := context.Background()
	base := NewScriptedLLM(WithResponses(TextResponse("x"), TextResponse("y")))

	toolsA := []Tool{{Name: "a", Description: "A", Parameters: json.RawMessage(`{}`)}}
	toolsB := []Tool{{Name: "b", Description: "B", Parameters: json.RawMessage(`{}`)}}

	a, err := base.WithTools(toolsA)
	if err != nil {
		t.Fatalf("WithTools(A): %v", err)
	}
	b, err := base.WithTools(toolsB)
	if err != nil {
		t.Fatalf("WithTools(B): %v", err)
	}
	if a == b {
		t.Fatal("WithTools must return distinct values")
	}

	// Concurrent Generate calls must not race (-race flag asserts this).
	var wg sync.WaitGroup
	wg.Add(2)
	errCh := make(chan error, 2)
	go func() {
		defer wg.Done()
		_, err := a.Generate(ctx, Request{})
		errCh <- err
	}()
	go func() {
		defer wg.Done()
		_, err := b.Generate(ctx, Request{})
		errCh <- err
	}()
	wg.Wait()
	close(errCh)
	for e := range errCh {
		if e != nil {
			t.Errorf("concurrent Generate err: %v", e)
		}
	}
}

// ----- StreamReader idempotent close -----
func TestStreamReaderClosesIdempotent(t *testing.T) {
	ctx := context.Background()
	m := NewScriptedLLM(WithResponses(TextResponse("hi")))
	sr, err := m.Stream(ctx, Request{})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if err := sr.Close(); err != nil {
		t.Errorf("Close#1: %v", err)
	}
	// MUST NOT panic on second Close
	if err := sr.Close(); err != nil {
		t.Errorf("Close#2: %v", err)
	}
	// After close, Next returns io.EOF
	_, err = sr.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("Next after Close = %v, want io.EOF", err)
	}
}

// ----- Sentinel errors.Is round-trip -----
func TestSentinelErrors_ErrorsIs(t *testing.T) {
	cases := []struct {
		name string
		s    error
	}{
		{"ErrCapabilityNotSupported", ErrCapabilityNotSupported},
		{"ErrScriptExhausted", ErrScriptExhausted},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wrapped := fmt.Errorf("wrap: %w", c.s)
			if !errors.Is(wrapped, c.s) {
				t.Errorf("errors.Is(wrapped, %s) = false, want true", c.name)
			}
		})
	}
}

// ----- StreamEventKind variant ordering -----
func TestStreamEventKind_Variants(t *testing.T) {
	cases := []struct {
		k    StreamEventKind
		want uint8
	}{
		{EventTextDelta, 0},
		{EventToolCallStart, 1},
		{EventToolCallArgsDelta, 2},
		{EventToolCallEnd, 3},
		{EventThinkingDelta, 4},
		{EventDone, 5},
	}
	for _, c := range cases {
		if uint8(c.k) != c.want {
			t.Errorf("kind = %d, want %d", c.k, c.want)
		}
	}
}

// ----- ProviderInfo JSON round-trip (Capabilities serialisation) -----
func TestProviderInfo_JSONRoundTrip(t *testing.T) {
	in := ProviderInfo{
		Provider: "openai",
		Model:    "gpt-4o-mini",
		Capabilities: Capabilities{
			Tools: true, Embeddings: true, StructuredOutputs: false, PromptCaching: false,
		},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"provider":"openai","model":"gpt-4o-mini","capabilities":{"tools":true,"embeddings":true,"structured_outputs":false,"prompt_caching":false}}`
	if string(b) != want {
		t.Errorf("Marshal:\n got  %s\n want %s", b, want)
	}

	var out ProviderInfo
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out != in {
		t.Errorf("round-trip:\n got  %+v\n want %+v", out, in)
	}
}
