package memory

// This file adds "ChatGPT-style" profile metadata helpers on top of
// MemoryItem.Metadata WITHOUT changing the MemoryItem struct or any
// existing Memory interface method. All state lives inside the
// existing map[string]any under a reserved "_"-prefixed namespace so
// it does not collide with caller-supplied metadata keys.

// Source classifies how a memory entered the system.
type Source string

const (
	// SourceUserSaved marks a memory the user explicitly asked the agent
	// to remember ("Remember that I ..."). Constructors default Pinned=true.
	SourceUserSaved Source = "user_saved"

	// SourceAgentInferred marks a memory the agent decided to record
	// from conversation without an explicit save instruction.
	SourceAgentInferred Source = "agent_inferred"

	// SourceSystem marks a memory injected by the platform (defaults,
	// onboarding bootstrap, etc.).
	SourceSystem Source = "system"

	// SourceUnknown is the zero value (no _source metadata set).
	SourceUnknown Source = ""
)

// Category is the user-facing taxonomy. Mirrors ChatGPT's
// "Manage memories" filters (User / Feedback / Project / Reference).
// Callers can store additional category strings; these constants are
// just the canonical names.
type Category string

const (
	CategoryUser      Category = "user"
	CategoryFeedback  Category = "feedback"
	CategoryProject   Category = "project"
	CategoryReference Category = "reference"
)

// Reserved metadata keys. The leading "_" namespace avoids collision
// with caller-supplied Metadata keys. Kept package-private so callers
// must go through the typed accessors below.
const (
	metaKeyScope    = "_scope"
	metaKeySource   = "_source"
	metaKeyCategory = "_category"
	metaKeyPinned   = "_pinned"
	metaKeyDisabled = "_disabled"
)

// --- Getters ---------------------------------------------------------------
//
// All getters return the zero value when Metadata is nil, the key is
// missing, or the stored value has the wrong type. They never panic.

// GetSource returns the Source recorded on the item, or SourceUnknown
// when absent / wrong type. It accepts BOTH the strongly-typed Source
// form (as written by the legacy llm-agent profile helpers, live in
// memory before serialization) and the plain string form (as setters
// here write, and as any JSON round-trip yields), so it round-trips
// data written by either historical writer.
func GetSource(it MemoryItem) Source {
	if it.Metadata == nil {
		return SourceUnknown
	}
	switch v := it.Metadata[metaKeySource].(type) {
	case Source:
		return v
	case string:
		return Source(v)
	default:
		return SourceUnknown
	}
}

// GetCategory returns the Category recorded on the item, or empty when
// absent / wrong type. Like GetSource it accepts both the typed Category
// and plain string forms for cross-writer round-trip safety.
func GetCategory(it MemoryItem) Category {
	if it.Metadata == nil {
		return ""
	}
	switch v := it.Metadata[metaKeyCategory].(type) {
	case Category:
		return v
	case string:
		return Category(v)
	default:
		return ""
	}
}

// IsPinned reports whether the item is marked pinned. Pinned items are
// excluded from Forget strategies and (when SavedBoost is configured)
// receive a multiplicative score boost during Search.
func IsPinned(it MemoryItem) bool {
	if it.Metadata == nil {
		return false
	}
	raw, ok := it.Metadata[metaKeyPinned]
	if !ok {
		return false
	}
	pinned, ok := raw.(bool)
	return ok && pinned
}

// IsDisabled reports whether the item is marked disabled. Disabled
// items remain in storage (Get / Stats / Forget still see them) but
// are filtered out of Search results.
func IsDisabled(it MemoryItem) bool {
	if it.Metadata == nil {
		return false
	}
	raw, ok := it.Metadata[metaKeyDisabled]
	if !ok {
		return false
	}
	disabled, ok := raw.(bool)
	return ok && disabled
}

// --- Setters ---------------------------------------------------------------
//
// All setters initialize Metadata when nil.

// SetSource writes the Source onto the item's Metadata.
func SetSource(it *MemoryItem, src Source) {
	ensureMetadata(it)
	if src == SourceUnknown {
		delete(it.Metadata, metaKeySource)
		return
	}
	it.Metadata[metaKeySource] = string(src)
}

// SetCategory writes the Category onto the item's Metadata.
func SetCategory(it *MemoryItem, cat Category) {
	ensureMetadata(it)
	if cat == "" {
		delete(it.Metadata, metaKeyCategory)
		return
	}
	it.Metadata[metaKeyCategory] = string(cat)
}

// SetPinned writes the pinned flag onto the item's Metadata.
func SetPinned(it *MemoryItem, pinned bool) {
	ensureMetadata(it)
	if !pinned {
		delete(it.Metadata, metaKeyPinned)
		return
	}
	it.Metadata[metaKeyPinned] = true
}

// SetDisabled writes the disabled flag onto the item's Metadata.
func SetDisabled(it *MemoryItem, disabled bool) {
	ensureMetadata(it)
	if !disabled {
		delete(it.Metadata, metaKeyDisabled)
		return
	}
	it.Metadata[metaKeyDisabled] = true
}

func ensureMetadata(it *MemoryItem) {
	if it.Metadata == nil {
		it.Metadata = make(map[string]any, 4)
	}
}
