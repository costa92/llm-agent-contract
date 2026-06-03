package agents

import (
	"context"
	"encoding/json"
)

// Tool is a capability unit an Agent may invoke.
//
// Description is shown to the LLM (it decides whether to call); Schema describes
// the parameters as raw JSON Schema (we don't validate it — upstream provider does);
// Execute does the work and returns a string suitable for either prompt-injection
// (ReActAgent's Observation) or aggregation (FunctionCallAgent's answer).
type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

// ExecuteFunc is the signature used when wrapping a plain function as a Tool.
type ExecuteFunc func(ctx context.Context, args json.RawMessage) (string, error)
