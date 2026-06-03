package memory

import (
	"time"
)

// Kind identifies which of the three Memory types an item belongs to.
type Kind string

const (
	KindWorking  Kind = "working"
	KindEpisodic Kind = "episodic"
	KindSemantic Kind = "semantic"
)

// MemoryItem is the unit of storage. Importance ∈ [0,1] drives forget
// + consolidate. Tags are arbitrary labels (used by Semantic for
// overlap scoring; cosmetic in Working/Episodic).
type MemoryItem struct {
	ID         string
	Content    string
	Tags       []string
	Importance float64
	CreatedAt  time.Time
	AccessedAt time.Time
	Metadata   map[string]any
}

// SearchResult pairs a MemoryItem with its query-relevance Score.
// Score is a domain-specific composite (per Memory type) in [0, +∞).
type SearchResult struct {
	Item  MemoryItem
	Score float64
}

// Stats summarizes a Memory's contents — useful for debugging + the
// MemoryTool stats action.
type Stats struct {
	Count         int
	Capacity      int           // 0 = unlimited
	OldestAge     time.Duration // duration since the oldest item's CreatedAt
	AvgImportance float64
}

// ListFilter narrows what List returns. Zero-value = no filter (return
// all non-disabled items, regardless of scope). All non-empty fields
// constrain the result set conjunctively (AND across fields). Within
// Tags the match is any-of (OR within the slice).
type ListFilter struct {
	// Scope matches items via Scope.Matches semantics (empty axis on the
	// filter scope is a wildcard for that axis). Zero-value matches any
	// scope including legacy unscoped items.
	Scope Scope

	// Source / Category are exact matches; empty string = any.
	Source   Source
	Category Category

	// Tags is an any-of filter (case-insensitive). Empty slice = any.
	Tags []string

	// PinnedOnly restricts results to items where IsPinned(item) == true.
	PinnedOnly bool

	// IncludeDisabled controls whether items where IsDisabled(item) ==
	// true appear in the page. Defaults to false (disabled items hidden).
	IncludeDisabled bool

	// MinImportance is an inclusive lower bound on Importance. A value
	// <= 0 means no minimum.
	MinImportance float64
}

// ListPage is one page of items, deterministically ordered by
// (CreatedAt DESC, ID ASC). NextCursor is the empty string when the
// caller has reached the end of the filtered result set; otherwise
// pass it back as the cursor argument to fetch the next page.
type ListPage struct {
	Items      []MemoryItem
	NextCursor string
}

// ConsolidateOptions tunes Consolidate. Threshold is the minimum
// importance to promote (default 0.7). MinAge optionally requires
// items to have been around at least this long before promotion
// (default 0 = any age qualifies).
type ConsolidateOptions struct {
	Threshold float64
	MinAge    time.Duration
}

// ForgetStrategy picks the rule used by Forget.
type ForgetStrategy string

const (
	ForgetByImportance ForgetStrategy = "importance"
	ForgetByAge        ForgetStrategy = "age"
	ForgetByCapacity   ForgetStrategy = "capacity"
)

// ForgetOptions tunes Forget. Threshold is the cutoff for the
// "importance" strategy; MaxAge for "age"; Keep is the cap retained
// after "capacity" eviction.
type ForgetOptions struct {
	Strategy  ForgetStrategy
	Threshold float64       // for importance
	MaxAge    time.Duration // for age
	Keep      int           // for capacity (number of items to KEEP)
}
