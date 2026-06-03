// Package agents is the leaf contract for the llm-agent framework's agent layer:
// the Agent and Tool interfaces plus the pure trace data types (Result, Step,
// StepEvent, Usage, Task) and the StepKind enum. It is stdlib-only — no go.sum,
// no internal/* imports, no project-specific packages — so any consumer can
// depend on the agent contract without pulling in the concrete framework.
//
// Concrete Agent paradigms (Simple/ReAct/Reflection/PlanAndSolve/FunctionCall),
// constructors (NewFuncTool/NewSimpleAgent/…), the Registry, and the
// agents.Tool → llm.Tool bridge (AsLLMTool) live in github.com/costa92/llm-agent,
// which re-exports these types via aliases for backward compatibility.
package agents
