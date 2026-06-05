package prompt_test

import (
	"context"
	"fmt"

	"github.com/costa92/llm-agent-contract/llm"
	"github.com/costa92/llm-agent-contract/prompt"
)

func Example() {
	tmpl := prompt.MustNew(prompt.Spec{
		System: "You are a {persona} assistant. Answer in {lang}.",
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
		"lang":     "English",
		"question": "what is 2+2?",
	}, history)
	if err != nil {
		panic(err)
	}

	for _, m := range msgs {
		fmt.Printf("%s: %s\n", m.Role, m.Content)
	}

	// Output:
	// system: You are a concise assistant. Answer in English.
	// user: ping
	// assistant: pong
	// user: hi
	// assistant: hello
	// user: what is 2+2?
}

func ExampleRequester() {
	tmpl := prompt.MustNew(prompt.Spec{
		System: "You are a {persona} assistant.",
		User:   "{question}",
	})

	// Requester is an optional capability: type-assert for it.
	r, ok := tmpl.(prompt.Requester)
	if !ok {
		panic("template does not implement Requester")
	}
	req, err := r.FormatRequest(context.Background(), prompt.Vars{
		"persona":  "helpful",
		"question": "what is 2+2?",
	}, nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("system_prompt: %s\n", req.SystemPrompt)
	fmt.Printf("messages: %d\n", len(req.Messages))
	fmt.Printf("user: %s\n", req.Messages[0].Content)

	// Output:
	// system_prompt: You are a helpful assistant.
	// messages: 1
	// user: what is 2+2?
}
