package memory

import (
	"context"
)

// SnapshotVersion is the current Snapshot schema version. Older versions on
// import may need migration; Importers reject unknown versions.
const SnapshotVersion = 1

// Snapshot is a portable dump of one Memory's contents. JSON-serializable
// (encoding/json). Vectors are inlined; on Restore they are reused as-is so
// the receiving Memory does not need to re-embed existing content.
type Snapshot struct {
	Version int            `json:"version"`
	Kind    Kind           `json:"kind"`
	Items   []SnapshotItem `json:"items"`
}

// SnapshotItem pairs a MemoryItem with its cached embedding vector.
type SnapshotItem struct {
	Item   MemoryItem `json:"item"`
	Vector []float32  `json:"vector"`
}

// ImportMode controls how Importer merges incoming items with existing ones.
type ImportMode string

const (
	// ImportReplace wipes the target memory then loads the snapshot.
	ImportReplace ImportMode = "replace"
	// ImportMerge adds unseen items; skips items whose ID already exists.
	ImportMerge ImportMode = "merge"
	// ImportUpsert adds unseen items; overwrites items whose ID already exists.
	ImportUpsert ImportMode = "upsert"
)

// ImportReport summarizes the outcome of an Import call.
type ImportReport struct {
	Loaded   int     `json:"loaded"`
	Skipped  int     `json:"skipped"`
	Replaced int     `json:"replaced"`
	Errors   []error `json:"-"` // not serialized; surfaces via Error()
}

// SnapshotStore is the pluggable persistence backend. Implementations are
// keyed (the key identifies a logical snapshot, often a session/user ID).
// Stdlib-only impl in core is FilesystemStore; downstream repos can inject
// SQLite/Postgres/S3/Redis stores without core taking a dep.
type SnapshotStore interface {
	Save(ctx context.Context, key string, snap Snapshot) error
	Load(ctx context.Context, key string) (Snapshot, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context) ([]string, error)
}
