package prompt_test

import (
	"context"
	"testing"

	"github.com/costa92/llm-agent-contract/llm"
	"github.com/costa92/llm-agent-contract/prompt"
)

func TestRequesterLiftsSystem(t *testing.T) {
	tmpl := prompt.MustNew(prompt.Spec{
		System: "You are {persona}.",
		User:   "{q}",
	})
	r, ok := tmpl.(prompt.Requester)
	if !ok {
		t.Fatal("compiled template must satisfy Requester")
	}
	req, err := r.FormatRequest(context.Background(), prompt.Vars{
		"persona": "helpful", "q": "hi",
	}, nil)
	if err != nil {
		t.Fatalf("FormatRequest: %v", err)
	}
	if req.SystemPrompt != "You are helpful." {
		t.Fatalf("system not lifted: %q", req.SystemPrompt)
	}
	for _, m := range req.Messages {
		if m.Role == "system" {
			t.Fatalf("system turn must not remain in Messages: %+v", req.Messages)
		}
	}
	if len(req.Messages) != 1 || req.Messages[0].Content != "hi" {
		t.Fatalf("unexpected messages: %+v", req.Messages)
	}
}

func TestRequesterNoSystem(t *testing.T) {
	tmpl := prompt.MustNew(prompt.Spec{User: "q"})
	r := tmpl.(prompt.Requester)
	req, err := r.FormatRequest(context.Background(), prompt.Vars{}, nil)
	if err != nil {
		t.Fatalf("FormatRequest: %v", err)
	}
	if req.SystemPrompt != "" {
		t.Fatalf("expected empty SystemPrompt, got %q", req.SystemPrompt)
	}
	if len(req.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(req.Messages))
	}
}

func TestRequesterMetadataPassthrough(t *testing.T) {
	tmpl := prompt.MustNew(prompt.Spec{
		User:     "q",
		Metadata: map[string]any{"trace_id": "abc", "source": "rag"},
	})
	r := tmpl.(prompt.Requester)
	req, err := r.FormatRequest(context.Background(), prompt.Vars{}, nil)
	if err != nil {
		t.Fatalf("FormatRequest: %v", err)
	}
	if req.Metadata["trace_id"] != "abc" || req.Metadata["source"] != "rag" {
		t.Fatalf("metadata not passed through: %+v", req.Metadata)
	}
}

func TestRequesterNilMetadata(t *testing.T) {
	tmpl := prompt.MustNew(prompt.Spec{User: "q"})
	r := tmpl.(prompt.Requester)
	req, _ := r.FormatRequest(context.Background(), prompt.Vars{}, nil)
	if req.Metadata != nil {
		t.Fatalf("expected nil Metadata, got %+v", req.Metadata)
	}
}

func TestRequesterMetadataIsolated(t *testing.T) {
	// Mutating the returned Request.Metadata must not corrupt the
	// template's Spec.Metadata for the next caller (defensive copy).
	tmpl := prompt.MustNew(prompt.Spec{
		User:     "q",
		Metadata: map[string]any{"k": "v"},
	})
	r := tmpl.(prompt.Requester)
	req1, _ := r.FormatRequest(context.Background(), prompt.Vars{}, nil)
	req1.Metadata["k"] = "mutated"
	req2, _ := r.FormatRequest(context.Background(), prompt.Vars{}, nil)
	if req2.Metadata["k"] != "v" {
		t.Fatalf("metadata not isolated between calls: %+v", req2.Metadata)
	}
	_ = llm.Request{}
}
