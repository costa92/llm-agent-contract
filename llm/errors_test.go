package llm

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestTypedErrors_ErrorsAs(t *testing.T) {
	inner := errors.New("inner SDK error")

	t.Run("AuthError", func(t *testing.T) {
		wrapped := fmt.Errorf("outer: %w", &AuthError{Provider: "openai", Wrapped: inner})
		var target *AuthError
		if !errors.As(wrapped, &target) {
			t.Fatal("errors.As(AuthError) = false, want true")
		}
		if target.Provider != "openai" {
			t.Fatalf("Provider = %q, want openai", target.Provider)
		}
	})

	t.Run("RateLimitError", func(t *testing.T) {
		wrapped := fmt.Errorf("outer: %w", &RateLimitError{Provider: "openai", Wrapped: inner})
		var target *RateLimitError
		if !errors.As(wrapped, &target) {
			t.Fatal("errors.As(RateLimitError) = false, want true")
		}
		if target.Provider != "openai" {
			t.Fatalf("Provider = %q, want openai", target.Provider)
		}
	})

	t.Run("InvalidRequestError", func(t *testing.T) {
		wrapped := fmt.Errorf("outer: %w", &InvalidRequestError{Provider: "openai", Wrapped: inner})
		var target *InvalidRequestError
		if !errors.As(wrapped, &target) {
			t.Fatal("errors.As(InvalidRequestError) = false, want true")
		}
		if target.Provider != "openai" {
			t.Fatalf("Provider = %q, want openai", target.Provider)
		}
	})

	t.Run("TransientError", func(t *testing.T) {
		wrapped := fmt.Errorf("outer: %w", &TransientError{Provider: "openai", Wrapped: inner})
		var target *TransientError
		if !errors.As(wrapped, &target) {
			t.Fatal("errors.As(TransientError) = false, want true")
		}
		if target.Provider != "openai" {
			t.Fatalf("Provider = %q, want openai", target.Provider)
		}
	})
}

func TestTypedErrors_UnwrapChain(t *testing.T) {
	inner := errors.New("inner SDK error")
	cases := []struct {
		name string
		err  interface {
			error
			Unwrap() error
		}
	}{
		{"AuthError", &AuthError{Provider: "openai", Wrapped: inner}},
		{"RateLimitError", &RateLimitError{Provider: "openai", Wrapped: inner}},
		{"InvalidRequestError", &InvalidRequestError{Provider: "openai", Wrapped: inner}},
		{"TransientError", &TransientError{Provider: "openai", Wrapped: inner}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !errors.Is(tc.err.Unwrap(), inner) {
				t.Fatalf("Unwrap() = %v, want %v", tc.err.Unwrap(), inner)
			}
		})
	}
}

func TestRateLimitError_ErrorString(t *testing.T) {
	err := &RateLimitError{
		Provider:   "openai",
		RetryAfter: "30",
		Reason:     "quota_exhausted",
		Wrapped:    errors.New("too many requests"),
	}

	got := err.Error()
	for _, want := range []string{"openai", "rate limited", "quota_exhausted", "30"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Error() = %q, missing %q", got, want)
		}
	}
}

func TestTypedErrors_NilWrapped(t *testing.T) {
	cases := []struct {
		name string
		err  interface {
			error
			Unwrap() error
		}
	}{
		{"AuthError", &AuthError{Provider: "openai"}},
		{"RateLimitError", &RateLimitError{Provider: "openai"}},
		{"InvalidRequestError", &InvalidRequestError{Provider: "openai"}},
		{"TransientError", &TransientError{Provider: "openai"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_ = tc.err.Error()
			if tc.err.Unwrap() != nil {
				t.Fatalf("Unwrap() = %v, want nil", tc.err.Unwrap())
			}
		})
	}
}
