// Package prompt owns the contract's reusable prompt-template seam.
//
// A Template turns named variables into an ordered []llm.Message: an
// optional System turn, optional few-shot example turns, an injected
// history slice, and the current user turn. It is the prompt-assembly
// analogue of llm.ChatModel — variable interpolation plus message
// layout — and its output type is contract's own llm.Message, so
// agents, RAG, and a future flow-node layer can share one primitive.
//
// The engine is stdlib-only: a minimal {var} interpolator (the default)
// or Go text/template. There are no third-party dependencies; the only
// import beyond the standard library is the sibling llm package.
//
// # Exported surface
//
//   - Template          base interface: Format(ctx, Vars, history) -> []llm.Message
//   - Requester         OPTIONAL capability: FormatRequest -> llm.Request
//     (lifts a leading system turn into Request.SystemPrompt)
//   - Vars              map[string]any interpolation input
//   - Spec              the builder: System / FewShot / User / HistorySlot /
//     Engine / Metadata, compiled once by New
//   - Turn              one literal template turn (role + content)
//   - Engine            EngineBrace (default) | EngineGoTemplate
//   - HistoryPlacement  BeforeUser (default) | NoHistory
//   - ErrMissingVar     sentinel wrapped on a missing variable
//   - New / MustNew      compile a Spec into an immutable Template
//
// # Engines
//
// EngineBrace (default) is a minimal {var} interpolator. Values are
// inserted verbatim and never re-parsed, so it is injection-safe even
// when a variable value contains template-looking text. Rules:
//
//   - {name} is replaced by the Vars["name"] value, formatted with %v.
//   - A referenced {name} with no entry in Vars is a STRICT error that
//     wraps ErrMissingVar (the name is included) — typos surface in
//     tests rather than shipping a silently-broken prompt.
//   - A literal brace is escaped by doubling: "{{" renders "{" and "}}"
//     renders "}".
//   - An unmatched "{" / "}" or an empty "{}" placeholder is a compile
//     error returned by New (never deferred to Format).
//
// EngineGoTemplate opts into stdlib text/template ({{.var}} plus
// pipelines, range, if). It is compiled with Option("missingkey=error"),
// so a missing key also wraps ErrMissingVar at render time; a malformed
// template fails at New, not Format. text/template does NOT auto-escape,
// which is correct for prompts (raw text is wanted) — but template
// STRINGS should be developer-authored, not user-supplied, to avoid
// action injection. EngineBrace has no such surface.
//
// # Message assembly
//
// Format emits, in order: the System turn (if Spec.System is non-empty),
// the FewShot turns, the runtime history (spliced per Spec.HistorySlot),
// then the User turn. With the default HistorySlot (BeforeUser) the order
// is: system, few-shot, history, user. NoHistory ignores the history
// argument entirely. A nil history is a no-op. Spec.User is required;
// New rejects an empty user turn.
//
// A compiled Template is immutable and safe for concurrent Format calls.
//
// # Requester / FormatRequest
//
// Requester is the established optional-capability idiom (compare
// llm.ToolCaller, memory.Lister): callers type-assert for it.
// FormatRequest runs Format, lifts a leading system message into
// llm.Request.SystemPrompt (per the llm.Request convention), and passes
// Spec.Metadata through (defensively copied) onto llm.Request.Metadata so
// a downstream bridge — e.g. RAG — does not silently drop request extras.
//
// # v1 scope (explicit non-goals)
//
//   - Text only. Spec, Turn and Vars are all strings, so v1 produces
//     PURELY TEXTUAL turns. Although llm.Message carries Images for
//     vision models, this package has NO mechanism to attach images in
//     v1; an image escape-hatch is deferred to a later version.
//   - Few-shot is STRUCTURAL (the Spec.FewShot []Turn slice), not a
//     dynamic in-string loop. Generating a variable NUMBER of few-shot
//     turns from a slice is not on the v1 structured path; an
//     EngineGoTemplate {{range}} can build such content inside a single
//     turn's text, but that is content generation, not structured turns.
//   - No streaming variant (templates are pure CPU string assembly), no
//     named-partial registry, and no single-text Render helper in v1.
//     The output is always []llm.Message (or llm.Request via Requester).
//
// NEW in v0.5.
package prompt
