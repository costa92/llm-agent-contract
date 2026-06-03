package memory

import (
	"errors"
)

// ErrNotFound is returned by Get / Update / Remove when id is absent.
var ErrNotFound = errors.New("memory: item not found")

// ErrEmptyQuery is returned by Search when query is whitespace-only.
var ErrEmptyQuery = errors.New("memory: query is required")

// ErrEmbedderRequired is returned by constructors when Embedder is nil.
var ErrEmbedderRequired = errors.New("memory: embedder is required")

// ErrKindDisabled is returned when an operation targets a memory kind
// that wasn't activated on this Manager.
var ErrKindDisabled = errors.New("memory: kind disabled on this manager")

// ErrSnapshotVersionMismatch is returned by Import when snap.Version is unknown.
var ErrSnapshotVersionMismatch = errors.New("memory: snapshot version mismatch")

// ErrSnapshotKindMismatch is returned by Import when snap.Kind != receiving Memory.Type().
var ErrSnapshotKindMismatch = errors.New("memory: snapshot kind mismatch")

// ErrSnapshotStoreNotConfigured is returned by Manager.ExportAll / ImportAll
// when ManagerOptions.SnapshotStore was not set.
var ErrSnapshotStoreNotConfigured = errors.New("memory: SnapshotStore not configured on Manager")
