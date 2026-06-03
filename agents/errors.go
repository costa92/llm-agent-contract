package agents

import "errors"

// ErrToolNotFound is returned when a tool lookup fails — e.g. an Agent emits an
// Action naming a tool absent from the Registry. Consumers may assert it with
// errors.Is across the module boundary.
var ErrToolNotFound = errors.New("agents: tool not found")

// ErrEmptyInput is returned when an Agent is run with empty input.
var ErrEmptyInput = errors.New("agents: empty input")
