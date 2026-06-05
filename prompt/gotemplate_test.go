package prompt_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/costa92/llm-agent-contract/prompt"
)

func TestGoTemplateRange(t *testing.T) {
	tmpl, err := prompt.New(prompt.Spec{
		Engine: prompt.EngineGoTemplate,
		User:   "items:{{range .list}} {{.}}{{end}}",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	msgs, err := tmpl.Format(context.Background(), prompt.Vars{"list": []string{"a", "b", "c"}}, nil)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	got := msgs[len(msgs)-1].Content
	if got != "items: a b c" {
		t.Fatalf("range render: got %q", got)
	}
}

func TestGoTemplateMissingKey(t *testing.T) {
	tmpl := prompt.MustNew(prompt.Spec{
		Engine: prompt.EngineGoTemplate,
		User:   "value is {{.absent}}",
	})
	_, err := tmpl.Format(context.Background(), prompt.Vars{}, nil)
	if err == nil {
		t.Fatal("expected error for missing key")
	}
	if !errors.Is(err, prompt.ErrMissingVar) {
		t.Fatalf("expected ErrMissingVar, got %v", err)
	}
}

func TestGoTemplateMalformedFailsAtNew(t *testing.T) {
	_, err := prompt.New(prompt.Spec{
		Engine: prompt.EngineGoTemplate,
		User:   "broken {{range}}",
	})
	if err == nil {
		t.Fatal("expected compile error at New for malformed template")
	}
	if errors.Is(err, prompt.ErrMissingVar) {
		t.Fatalf("compile error mis-classified: %v", err)
	}
	if !strings.Contains(err.Error(), "compile") {
		t.Fatalf("expected compile context in error, got %v", err)
	}
}
