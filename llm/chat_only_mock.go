package llm

import "context"

// ChatOnlyMock implements ONLY ChatModel — no ToolCaller, no Embedder,
// no StructuredOutputs. Used in agent tests (Phase 3) to verify
// graceful capability degradation: ReAct falls back to scratchpad
// templating when model.(ToolCaller) fails. Phase 0 lands the type so
// downstream tests have a canonical capability-degraded mock.
type ChatOnlyMock struct {
	Provider string
	Model    string
	Resp     Response
}

// Compile-time: ChatModel ONLY — explicitly NOT ToolCaller / Embedder /
// StructuredOutputs (negative assertions in llm/llm_test.go).
var _ ChatModel = (*ChatOnlyMock)(nil)

func (m *ChatOnlyMock) Generate(_ context.Context, _ Request) (Response, error) {
	return m.Resp, nil
}

func (m *ChatOnlyMock) Stream(_ context.Context, _ Request) (StreamReader, error) {
	return newScriptedStream(m.Resp), nil
}

func (m *ChatOnlyMock) Info() ProviderInfo {
	return ProviderInfo{
		Provider:     m.Provider,
		Model:        m.Model,
		Capabilities: Capabilities{}, // ALL false — that's the point
	}
}
