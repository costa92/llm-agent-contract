package memory

// Scope identifies a memory partition. Empty fields are wildcards in
// lookup; non-empty fields constrain. Zero-value Scope{} = "match every
// item" (i.e. default behavior pre-v0.7, so existing callers see no
// change).
//
// Scope is propagated through context.Context via WithScope / ScopeFrom
// and read by ScopedManager on every operation.
type Scope struct {
	User    string
	Project string
	Session string
}

// IsZero reports whether the scope matches every item (all fields empty).
func (s Scope) IsZero() bool {
	return s.User == "" && s.Project == "" && s.Session == ""
}

// Equal reports literal field equality.
func (s Scope) Equal(other Scope) bool {
	return s.User == other.User && s.Project == other.Project && s.Session == other.Session
}

// Matches reports whether the (filter) scope s matches the (stored)
// concrete scope. Wildcard rule: an empty field on s ⇒ any value on
// concrete is accepted. If concrete has all-empty fields (legacy /
// unscoped data) and s is non-zero, Matches returns false — legacy
// data is invisible to scoped queries. Callers that want "see legacy
// data" should pass a zero-value Scope (which always returns true).
func (s Scope) Matches(concrete Scope) bool {
	if s.User != "" && s.User != concrete.User {
		return false
	}
	if s.Project != "" && s.Project != concrete.Project {
		return false
	}
	if s.Session != "" && s.Session != concrete.Session {
		return false
	}
	return true
}
