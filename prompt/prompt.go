package prompt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/costa92/llm-agent-contract/llm"
)

// Vars is the interpolation input. Values are formatted with the
// engine's default rules: EngineBrace renders each value with fmt's %v;
// EngineGoTemplate uses text/template's own value formatting.
type Vars map[string]any

// Template renders Vars into an ordered message list. Implementations
// MUST be safe for concurrent use (a compiled template is immutable);
// Format is pure given (vars, history) and performs no I/O.
type Template interface {
	// Format interpolates vars and assembles the messages. history is
	// spliced in at the template's history slot (see Spec.HistorySlot);
	// pass nil when there is no prior conversation.
	Format(ctx context.Context, vars Vars, history []llm.Message) ([]llm.Message, error)
}

// Requester is an OPTIONAL capability: a Template that can emit a full
// llm.Request (lifting a leading System turn into Request.SystemPrompt
// per the llm contract). Callers test for it; the base interface stays
// minimal.
//
//	if r, ok := tmpl.(prompt.Requester); ok {
//	    req, err := r.FormatRequest(ctx, vars, history)
//	}
type Requester interface {
	FormatRequest(ctx context.Context, vars Vars, history []llm.Message) (llm.Request, error)
}

// Turn is one literal template turn (role + content template).
type Turn struct {
	Role    string // "system" | "user" | "assistant" (llm role strings)
	Content string // interpolated with the same Vars
}

// HistoryPlacement controls where the runtime []llm.Message history is
// spliced into the assembled message list.
type HistoryPlacement int

const (
	// BeforeUser splices history after few-shot and before the user
	// turn: system, few-shot, HISTORY, user. This is the default.
	BeforeUser HistoryPlacement = iota
	// NoHistory ignores the history argument to Format entirely.
	NoHistory
)

// Engine selects the interpolation syntax.
type Engine int

const (
	// EngineBrace is the default: minimal {var} interpolation. Values
	// are inserted verbatim (never re-parsed), so it is injection-safe.
	// A literal brace is escaped {{ -> {. A referenced {name} with no
	// value in Vars is a strict error (wraps ErrMissingVar).
	EngineBrace Engine = iota
	// EngineGoTemplate opts into stdlib text/template ({{.var}} plus
	// pipelines, range, if). Compiled with Option("missingkey=error"),
	// so a missing key wraps ErrMissingVar at render time.
	EngineGoTemplate
)

// Spec declares the message layout a Template renders. It is the
// builder input; New(spec) compiles it once into an immutable Template.
type Spec struct {
	// System is the system instruction template (engine-interpolated).
	// Empty means no system turn.
	System string

	// FewShot are fixed example turns, interpolated with the SAME vars
	// as the rest (usually constant, but vars are allowed). They are
	// emitted after System and before History.
	FewShot []Turn

	// User is the current user-turn template. Required (non-empty).
	User string

	// HistorySlot controls where the runtime []llm.Message history is
	// spliced. Default (BeforeUser): system, few-shot, history, user.
	HistorySlot HistoryPlacement

	// Engine selects the interpolation syntax. Zero value = EngineBrace.
	Engine Engine

	// Metadata is passed through verbatim onto llm.Request.Metadata by
	// FormatRequest (it does not affect Format's []llm.Message output).
	// It lets a Template carry provider-specific request extras (e.g. a
	// RAG bridge preserving its own metadata) without losing them when
	// the prompt is turned into a request. Keys/values are copied as-is;
	// nil yields a nil Request.Metadata.
	Metadata map[string]any
}

// ErrMissingVar is returned (wrapped) by Format when a referenced
// placeholder has no value in Vars. For EngineBrace the missing key
// name is included; for EngineGoTemplate the underlying text/template
// "map has no entry" error is wrapped. Callers test with errors.Is.
var ErrMissingVar = errors.New("prompt: missing template variable")

// compiled is the immutable Template produced by New. It pre-compiles
// every template fragment so Format is pure CPU string assembly.
type compiled struct {
	system      *fragment
	fewShot     []compiledTurn
	user        *fragment
	historySlot HistoryPlacement
	metadata    map[string]any
}

type compiledTurn struct {
	role    string
	content *fragment
}

// New compiles spec into an immutable Template. It validates the engine
// syntax up front (returns an error on a malformed template) so
// render-time failures are limited to missing vars / type mismatches.
func New(spec Spec) (Template, error) {
	if strings.TrimSpace(spec.User) == "" {
		return nil, errors.New("prompt: Spec.User is required (empty user turn)")
	}

	c := &compiled{
		historySlot: spec.HistorySlot,
		metadata:    spec.Metadata,
	}

	var err error
	if spec.System != "" {
		if c.system, err = compileFragment(spec.Engine, "system", spec.System); err != nil {
			return nil, err
		}
	}
	for i, t := range spec.FewShot {
		f, ferr := compileFragment(spec.Engine, fmt.Sprintf("fewshot[%d]", i), t.Content)
		if ferr != nil {
			return nil, ferr
		}
		c.fewShot = append(c.fewShot, compiledTurn{role: t.Role, content: f})
	}
	if c.user, err = compileFragment(spec.Engine, "user", spec.User); err != nil {
		return nil, err
	}

	return c, nil
}

// MustNew is the test/package-var convenience: panics on a compile error.
func MustNew(spec Spec) Template {
	t, err := New(spec)
	if err != nil {
		panic(err)
	}
	return t
}

// Format implements Template.
func (c *compiled) Format(ctx context.Context, vars Vars, history []llm.Message) ([]llm.Message, error) {
	msgs := make([]llm.Message, 0, len(c.fewShot)+len(history)+2)

	if c.system != nil {
		content, err := c.system.render(vars)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, llm.Message{Role: "system", Content: content})
	}

	for _, t := range c.fewShot {
		content, err := t.content.render(vars)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, llm.Message{Role: t.role, Content: content})
	}

	if c.historySlot != NoHistory {
		msgs = append(msgs, history...)
	}

	userContent, err := c.user.render(vars)
	if err != nil {
		return nil, err
	}
	msgs = append(msgs, llm.Message{Role: "user", Content: userContent})

	return msgs, nil
}

// FormatRequest implements Requester. It runs Format, lifts a leading
// system message into Request.SystemPrompt, and passes Spec.Metadata
// through onto Request.Metadata.
func (c *compiled) FormatRequest(ctx context.Context, vars Vars, history []llm.Message) (llm.Request, error) {
	msgs, err := c.Format(ctx, vars, history)
	if err != nil {
		return llm.Request{}, err
	}

	var req llm.Request
	if len(msgs) > 0 && msgs[0].Role == "system" {
		req.SystemPrompt = msgs[0].Content
		msgs = msgs[1:]
	}
	req.Messages = msgs

	if c.metadata != nil {
		md := make(map[string]any, len(c.metadata))
		for k, v := range c.metadata {
			md[k] = v
		}
		req.Metadata = md
	}

	return req, nil
}

// fragment is a single compiled, renderable template fragment. Exactly
// one of brace / goTmpl is set, per the Spec.Engine that compiled it.
type fragment struct {
	name   string
	brace  *braceTemplate
	goTmpl *template.Template
}

func compileFragment(engine Engine, name, src string) (*fragment, error) {
	switch engine {
	case EngineGoTemplate:
		t, err := template.New(name).Option("missingkey=error").Parse(src)
		if err != nil {
			return nil, fmt.Errorf("prompt: compile %s: %w", name, err)
		}
		return &fragment{name: name, goTmpl: t}, nil
	case EngineBrace:
		bt, err := compileBrace(name, src)
		if err != nil {
			return nil, err
		}
		return &fragment{name: name, brace: bt}, nil
	default:
		return nil, fmt.Errorf("prompt: unknown engine %d", engine)
	}
}

func (f *fragment) render(vars Vars) (string, error) {
	if f.goTmpl != nil {
		var sb strings.Builder
		if err := f.goTmpl.Execute(&sb, map[string]any(vars)); err != nil {
			// text/template wraps the missingkey=error failure in its
			// own ExecError; normalize it to ErrMissingVar so callers
			// can errors.Is against one sentinel across both engines.
			if strings.Contains(err.Error(), "map has no entry for key") {
				return "", fmt.Errorf("prompt: %s: %w: %v", f.name, ErrMissingVar, err)
			}
			return "", fmt.Errorf("prompt: %s: %w", f.name, err)
		}
		return sb.String(), nil
	}
	return f.brace.render(vars)
}
