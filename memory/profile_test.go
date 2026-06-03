package memory

import (
	"encoding/json"
	"testing"
)

// TestGetSourceAcceptsBothForms locks the cross-writer round-trip property:
// GetSource/GetCategory must read both the strongly-typed form (written by the
// legacy llm-agent profile helpers, live in memory pre-serialization) and the
// plain string form (written by the setters here, and produced by any JSON
// round-trip).
func TestGetSourceAcceptsBothForms(t *testing.T) {
	typed := MemoryItem{Metadata: map[string]any{
		metaKeySource:   SourceUserSaved, // typed Source
		metaKeyCategory: CategoryProject, // typed Category
	}}
	if got := GetSource(typed); got != SourceUserSaved {
		t.Fatalf("GetSource(typed) = %q, want %q", got, SourceUserSaved)
	}
	if got := GetCategory(typed); got != CategoryProject {
		t.Fatalf("GetCategory(typed) = %q, want %q", got, CategoryProject)
	}

	str := MemoryItem{Metadata: map[string]any{
		metaKeySource:   string(SourceAgentInferred), // plain string
		metaKeyCategory: string(CategoryUser),
	}}
	if got := GetSource(str); got != SourceAgentInferred {
		t.Fatalf("GetSource(string) = %q, want %q", got, SourceAgentInferred)
	}
	if got := GetCategory(str); got != CategoryUser {
		t.Fatalf("GetCategory(string) = %q, want %q", got, CategoryUser)
	}
}

// TestSettersWriteJSONStableForm verifies setters store the plain string form
// so a value reads back identically before and after a JSON round-trip, and
// that the zero value deletes the key.
func TestSettersWriteJSONStableForm(t *testing.T) {
	it := MemoryItem{}
	SetSource(&it, SourceSystem)
	SetCategory(&it, CategoryFeedback)
	SetPinned(&it, true)
	SetDisabled(&it, true)

	// String storage means JSON round-trip is a no-op for the getters.
	raw, err := json.Marshal(it)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back MemoryItem
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if GetSource(back) != SourceSystem || GetCategory(back) != CategoryFeedback {
		t.Fatalf("round-trip lost source/category: %+v", back.Metadata)
	}
	if !IsPinned(back) || !IsDisabled(back) {
		t.Fatalf("round-trip lost pinned/disabled: %+v", back.Metadata)
	}

	// Zero value deletes the key.
	SetSource(&it, SourceUnknown)
	SetCategory(&it, "")
	SetPinned(&it, false)
	SetDisabled(&it, false)
	for _, k := range []string{metaKeySource, metaKeyCategory, metaKeyPinned, metaKeyDisabled} {
		if _, ok := it.Metadata[k]; ok {
			t.Fatalf("expected key %q deleted on zero value", k)
		}
	}
}
