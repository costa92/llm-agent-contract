package memory

import (
	"context"
)

// Memory is the contract every memory type satisfies. All methods are
// goroutine-safe in the bundled implementations.
type Memory interface {
	Type() Kind
	Add(ctx context.Context, item MemoryItem) (string, error) // returns generated ID
	Search(ctx context.Context, query string, topK int) ([]SearchResult, error)
	Get(ctx context.Context, id string) (MemoryItem, error)
	Update(ctx context.Context, id string, fn func(*MemoryItem)) error
	Remove(ctx context.Context, id string) error
	Stats() Stats
}

// Lister is implemented by Memory types that support enumeration. It
// is an OPTIONAL interface — the core Memory interface does NOT embed
// Lister, preserving the v0.6 additive-only contract. Callers test for
// the capability with a type assertion:
//
//	if l, ok := mem.(memory.Lister); ok { l.List(...) }
type Lister interface {
	List(ctx context.Context, filter ListFilter, pageSize int, cursor string) (ListPage, error)
}

// Exporter dumps a Memory to a Snapshot.
type Exporter interface {
	Export(ctx context.Context) (Snapshot, error)
}

// Importer restores from a Snapshot, replacing or merging existing content per
// mode. The Snapshot's Kind must match the receiving Memory's Type.
type Importer interface {
	Import(ctx context.Context, snap Snapshot, mode ImportMode) (ImportReport, error)
}

// Manager coordinates the 3 Memory types under one façade. It is the
// behavioral contract satisfied by the concrete *Manager in the memory
// engine repos. Lookup exposes the single-kind Memory for callers (such
// as MemoryTool) that need per-kind capability assertions (Lister, etc.).
type Manager interface {
	HasKind(kind Kind) bool
	Add(ctx context.Context, kind Kind, item MemoryItem) (string, error)
	Get(ctx context.Context, kind Kind, id string) (MemoryItem, error)
	Update(ctx context.Context, kind Kind, id string, fn func(*MemoryItem)) error
	Remove(ctx context.Context, kind Kind, id string) error
	Search(ctx context.Context, kind Kind, query string, topK int) ([]SearchResult, error)
	StatsAll() map[Kind]Stats
	SearchAll(ctx context.Context, query string, topK int) (map[Kind][]SearchResult, error)
	ListAll(ctx context.Context, filter ListFilter, pageSize int, cursors map[Kind]string) (map[Kind]ListPage, error)
	Consolidate(ctx context.Context, opts ConsolidateOptions) (int, error)
	Forget(ctx context.Context, kind Kind, opts ForgetOptions) (int, error)
	ExportAll(ctx context.Context, persistKey string) (map[Kind]Snapshot, error)
	ImportAll(ctx context.Context, snaps map[Kind]Snapshot, persistKey string, mode ImportMode) (map[Kind]ImportReport, error)
	Lookup(kind Kind) (Memory, error)
}
