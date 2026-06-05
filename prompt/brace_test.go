package prompt_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/costa92/llm-agent-contract/llm"
	"github.com/costa92/llm-agent-contract/prompt"
)

func TestBraceMissingVar(t *testing.T) {
	tmpl, err := prompt.New(prompt.Spec{User: "hello {missing}"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = tmpl.Format(context.Background(), prompt.Vars{}, nil)
	if err == nil {
		t.Fatal("expected error for missing var, got nil")
	}
	if !errors.Is(err, prompt.ErrMissingVar) {
		t.Fatalf("expected ErrMissingVar, got %v", err)
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected missing var name in error, got %v", err)
	}
}

func TestBraceRender(t *testing.T) {
	tests := []struct {
		name string
		src  string
		vars prompt.Vars
		want string
	}{
		{"single", "hi {name}", prompt.Vars{"name": "bob"}, "hi bob"},
		{"repeated", "{x}-{x}", prompt.Vars{"x": "a"}, "a-a"},
		{"adjacent", "{a}{b}", prompt.Vars{"a": "1", "b": "2"}, "12"},
		{"escape-open", "use {{var}} syntax", prompt.Vars{}, "use {var} syntax"},
		{"escape-both", "{{name}}={name}", prompt.Vars{"name": "v"}, "{name}=v"},
		{"empty-value-allowed", "[{x}]", prompt.Vars{"x": ""}, "[]"},
		{"non-string-value", "n={n}", prompt.Vars{"n": 42}, "n=42"},
		{"no-vars", "plain text", prompt.Vars{}, "plain text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := prompt.New(prompt.Spec{User: tt.src})
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			msgs, err := tmpl.Format(context.Background(), tt.vars, nil)
			if err != nil {
				t.Fatalf("Format: %v", err)
			}
			got := msgs[len(msgs)-1].Content
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBraceParseErrorAtNew(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"unmatched-open", "hello {name"},
		{"unmatched-close", "hello name}"},
		{"empty-placeholder", "x {} y"},
		{"blank-placeholder", "x {  } y"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := prompt.New(prompt.Spec{User: tt.src})
			if err == nil {
				t.Fatalf("expected New to fail for %q", tt.src)
			}
			// Parse failures must surface at New, never deferred to Format.
			if errors.Is(err, prompt.ErrMissingVar) {
				t.Fatalf("parse error mis-classified as missing-var: %v", err)
			}
		})
	}
}

func TestEmptyUserRejected(t *testing.T) {
	for _, src := range []string{"", "   ", "\t\n"} {
		if _, err := prompt.New(prompt.Spec{User: src}); err == nil {
			t.Fatalf("expected New to reject empty User %q", src)
		}
	}
}

func TestSystemAndFewShotMissingVar(t *testing.T) {
	// Missing var in System should also surface (wrapped) at Format.
	tmpl := prompt.MustNew(prompt.Spec{System: "be {persona}", User: "hi"})
	_, err := tmpl.Format(context.Background(), prompt.Vars{}, nil)
	if !errors.Is(err, prompt.ErrMissingVar) {
		t.Fatalf("expected ErrMissingVar from system, got %v", err)
	}

	// Few-shot var also interpolated with the same Vars.
	tmpl2 := prompt.MustNew(prompt.Spec{
		FewShot: []prompt.Turn{{Role: "user", Content: "{example}"}},
		User:    "q",
	})
	msgs, err := tmpl2.Format(context.Background(), prompt.Vars{"example": "ex"}, nil)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if msgs[0].Role != "user" || msgs[0].Content != "ex" {
		t.Fatalf("fewshot not interpolated: %+v", msgs[0])
	}
	_ = llm.Message{}
}
