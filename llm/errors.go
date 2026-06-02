package llm

import (
	"errors"
	"fmt"
)

// Sentinel errors for the llm package. Callers detect via errors.Is.
// Both sentinels MUST survive `fmt.Errorf("...: %w", sentinel)` wrapping.
var (
	// ErrCapabilityNotSupported is returned by methods on capability
	// interfaces when the bound model does not actually support the
	// capability — even though the Go type implements the interface.
	//
	// Canonical wrap pattern:
	//   return nil, fmt.Errorf("anthropic: embeddings: %w", llm.ErrCapabilityNotSupported)
	//
	// Callers detect with errors.Is(err, llm.ErrCapabilityNotSupported).
	ErrCapabilityNotSupported = errors.New("llm: capability not supported by bound model")

	// ErrScriptExhausted is returned by ScriptedLLM when the script runs
	// out of pre-recorded responses. Test code matches with errors.Is.
	ErrScriptExhausted = errors.New("llm: scripted llm: script exhausted")
)

// AuthError is returned by adapters when the provider rejects credentials.
//
// The original SDK error is preserved in Wrapped - call errors.Unwrap to
// retrieve it, or errors.As(err, &openaiErr) for provider-specific detail.
//
// Canonical detection:
//
//	var e *llm.AuthError
//	if errors.As(err, &e) { /* handle 401/403 */ }
type AuthError struct {
	Provider string
	Wrapped  error
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("%s: authentication failed: %v", e.Provider, e.Wrapped)
}

func (e *AuthError) Unwrap() error { return e.Wrapped }

// RateLimitError indicates the provider is rate-limiting the caller.
//
// RetryAfter is the raw value of any Retry-After header (provider-specific
// format - RFC 7231 HTTP-date or a number of seconds). Consumers parse.
//
// Reason is an optional discriminator for provider-specific quota states
// (e.g., "quota_exhausted" for OpenAI insufficient_quota; "" by default).
type RateLimitError struct {
	Provider   string
	RetryAfter string
	Reason     string
	Wrapped    error
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("%s: rate limited (reason=%q, retry_after=%q): %v",
		e.Provider, e.Reason, e.RetryAfter, e.Wrapped)
}

func (e *RateLimitError) Unwrap() error { return e.Wrapped }

// InvalidRequestError indicates the request was malformed, the model name
// was wrong, the model wasn't pulled (Ollama), or any other 4xx-other
// condition that operator action - not a retry - must resolve.
type InvalidRequestError struct {
	Provider string
	Wrapped  error
}

func (e *InvalidRequestError) Error() string {
	return fmt.Sprintf("%s: invalid request: %v", e.Provider, e.Wrapped)
}

func (e *InvalidRequestError) Unwrap() error { return e.Wrapped }

// TransientError indicates a 5xx, network failure, or
// context.DeadlineExceeded - the caller MAY retry per the K4 retry
// state machine (Phase 2).
//
// Phase 1 adapters return this; the retry loop itself lands in Phase 2.
type TransientError struct {
	Provider string
	Wrapped  error
}

func (e *TransientError) Error() string {
	return fmt.Sprintf("%s: transient error: %v", e.Provider, e.Wrapped)
}

func (e *TransientError) Unwrap() error { return e.Wrapped }
