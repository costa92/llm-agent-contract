package prompt

import (
	"fmt"
	"strings"
)

// braceTemplate is the compiled form of an EngineBrace source string. It
// is a flat list of segments: literal text or a {name} placeholder. A
// literal "{{" in the source escapes to a single "{"; "}}" escapes to a
// single "}". A single "{" must open a placeholder closed by "}", else
// compileBrace returns a parse error (surfaced from New, not Format).
type braceTemplate struct {
	name     string
	segments []braceSegment
}

type braceSegment struct {
	literal string // set when isVar is false
	varName string // set when isVar is true
	isVar   bool
}

func compileBrace(name, src string) (*braceTemplate, error) {
	bt := &braceTemplate{name: name}
	var lit strings.Builder

	flushLit := func() {
		if lit.Len() > 0 {
			bt.segments = append(bt.segments, braceSegment{literal: lit.String()})
			lit.Reset()
		}
	}

	i := 0
	for i < len(src) {
		ch := src[i]
		switch ch {
		case '{':
			// "{{" -> literal "{"
			if i+1 < len(src) && src[i+1] == '{' {
				lit.WriteByte('{')
				i += 2
				continue
			}
			// placeholder: scan to the closing "}"
			end := strings.IndexByte(src[i+1:], '}')
			if end < 0 {
				return nil, fmt.Errorf("prompt: compile %s: unmatched '{' at offset %d", name, i)
			}
			varName := src[i+1 : i+1+end]
			if strings.ContainsAny(varName, "{}") || strings.TrimSpace(varName) == "" {
				return nil, fmt.Errorf("prompt: compile %s: malformed placeholder %q", name, "{"+varName+"}")
			}
			flushLit()
			bt.segments = append(bt.segments, braceSegment{varName: varName, isVar: true})
			i = i + 1 + end + 1
		case '}':
			// "}}" -> literal "}"
			if i+1 < len(src) && src[i+1] == '}' {
				lit.WriteByte('}')
				i += 2
				continue
			}
			return nil, fmt.Errorf("prompt: compile %s: unmatched '}' at offset %d", name, i)
		default:
			lit.WriteByte(ch)
			i++
		}
	}
	flushLit()

	return bt, nil
}

func (bt *braceTemplate) render(vars Vars) (string, error) {
	var sb strings.Builder
	for _, seg := range bt.segments {
		if !seg.isVar {
			sb.WriteString(seg.literal)
			continue
		}
		v, ok := vars[seg.varName]
		if !ok {
			return "", fmt.Errorf("prompt: %s: %w %q", bt.name, ErrMissingVar, seg.varName)
		}
		sb.WriteString(fmt.Sprintf("%v", v))
	}
	return sb.String(), nil
}
