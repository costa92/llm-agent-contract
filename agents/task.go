package agents

import "encoding/json"

// Task pairs a Tool with its args for a single async invocation.
type Task struct {
	Tool Tool
	Args json.RawMessage
}
