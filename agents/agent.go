package agents

import "context"

// Agent is the minimal contract every Agent implementation satisfies.
type Agent interface {
	Name() string
	Run(ctx context.Context, input string) (Result, error)
	// RunStream emits trace step events through a channel. The channel is closed
	// when the Agent finishes; the final event always has Done=true with either
	// Final or Err set. Phase 8 SSE handlers are the natural consumer; service
	// layers don't need to write Step→event conversion themselves.
	RunStream(ctx context.Context, input string) (<-chan StepEvent, error)
}

// StepEvent is the transport unit emitted by RunStream.
//
//   - Done = false: Step is an intermediate event, Final/Err are nil.
//   - Done = true: terminal event, exactly one of Final or Err is non-nil.
//   - Channel close after the terminal event signals no more events.
//
// When ctx is canceled mid-run, the terminal event has Err set to ctx.Err()
// (typically context.Canceled or context.DeadlineExceeded). The channel is
// never closed silently — `for ev := range ch` consumers can rely on seeing
// a Done event before close, even on cancel.
type StepEvent struct {
	Step  Step
	Done  bool
	Final *Result
	Err   error
}

// Result carries the final answer plus full trace and accumulated usage.
//
// Trace memory contract (eng review 2026-04-27): Result.Trace is a debug
// snapshot for synchronous Run() callers and has no size limit. Streaming
// consumers (RunStream / SSE / gRPC stream) should consume StepEvents only
// and ignore Result.Trace at the end — they're the same information twice.
// Phase 8 SSE handlers should discard res.Trace once the channel closes
// (events already flushed to client). High-concurrency services that ignore
// this rule end up holding 50–100 Steps (~4KB each) per in-flight handler
// — 100 concurrent handlers ≈ 40MB wasted.
type Result struct {
	Answer string
	Trace  []Step
	Usage  Usage
}

// Usage tracks LLM cost across a single Run.
type Usage struct {
	LLMCalls int
	Tokens   int
}

// Step is one entry in the trace. Kind decides which fields are meaningful.
type Step struct {
	Kind    StepKind
	Content string // Thought / Reflection / Plan body
	Tool    string // Action only
	Args    string // Action only — raw JSON string
	Result  string // Observation only
}

// StepKind enumerates trace step types.
type StepKind string

const (
	StepThought     StepKind = "thought"
	StepAction      StepKind = "action"
	StepObservation StepKind = "observation"
	StepReflection  StepKind = "reflection"
	StepPlan        StepKind = "plan"
	StepFinal       StepKind = "final"
)
