package prompt_test

import (
	"context"
	"sync"
	"testing"

	"github.com/costa92/llm-agent-contract/llm"
	"github.com/costa92/llm-agent-contract/prompt"
)

func roles(msgs []llm.Message) []string {
	r := make([]string, len(msgs))
	for i, m := range msgs {
		r[i] = m.Role
	}
	return r
}

func eqStr(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestFormatOrdering(t *testing.T) {
	tmpl := prompt.MustNew(prompt.Spec{
		System: "You are a {persona} assistant.",
		FewShot: []prompt.Turn{
			{Role: "user", Content: "ping"},
			{Role: "assistant", Content: "pong"},
		},
		User:        "{question}",
		HistorySlot: prompt.BeforeUser,
	})
	history := []llm.Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello"},
	}
	msgs, err := tmpl.Format(context.Background(), prompt.Vars{
		"persona":  "concise",
		"question": "2+2?",
	}, history)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	want := []string{"system", "user", "assistant", "user", "assistant", "user"}
	if !eqStr(roles(msgs), want) {
		t.Fatalf("order mismatch: got %v want %v", roles(msgs), want)
	}
	if msgs[0].Content != "You are a concise assistant." {
		t.Fatalf("system not interpolated: %q", msgs[0].Content)
	}
	if msgs[len(msgs)-1].Content != "2+2?" {
		t.Fatalf("user not interpolated: %q", msgs[len(msgs)-1].Content)
	}
	// history must be spliced verbatim before user, after few-shot
	if msgs[3].Content != "hi" || msgs[4].Content != "hello" {
		t.Fatalf("history not spliced correctly: %+v", msgs[3:5])
	}
}

func TestFormatNoHistory(t *testing.T) {
	tmpl := prompt.MustNew(prompt.Spec{
		User:        "q",
		HistorySlot: prompt.NoHistory,
	})
	history := []llm.Message{{Role: "user", Content: "ignored"}}
	msgs, err := tmpl.Format(context.Background(), prompt.Vars{}, history)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	want := []string{"user"}
	if !eqStr(roles(msgs), want) {
		t.Fatalf("NoHistory should drop history: got %v", roles(msgs))
	}
}

func TestFormatNilHistory(t *testing.T) {
	tmpl := prompt.MustNew(prompt.Spec{System: "sys", User: "q"})
	msgs, err := tmpl.Format(context.Background(), prompt.Vars{}, nil)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if !eqStr(roles(msgs), []string{"system", "user"}) {
		t.Fatalf("nil history: got %v", roles(msgs))
	}
}

func TestFormatNoSystem(t *testing.T) {
	tmpl := prompt.MustNew(prompt.Spec{User: "q"})
	msgs, err := tmpl.Format(context.Background(), prompt.Vars{}, nil)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if !eqStr(roles(msgs), []string{"user"}) {
		t.Fatalf("empty system should produce no system turn: got %v", roles(msgs))
	}
}

func TestFormatConcurrent(t *testing.T) {
	tmpl := prompt.MustNew(prompt.Spec{
		System: "be {persona}",
		FewShot: []prompt.Turn{
			{Role: "user", Content: "{shot}"},
		},
		User: "{q}",
	})
	history := []llm.Message{{Role: "user", Content: "h"}}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				msgs, err := tmpl.Format(context.Background(), prompt.Vars{
					"persona": "x", "shot": "s", "q": "question",
				}, history)
				if err != nil {
					t.Errorf("Format: %v", err)
					return
				}
				if len(msgs) != 4 {
					t.Errorf("expected 4 msgs, got %d", len(msgs))
					return
				}
			}
		}()
	}
	wg.Wait()
}
