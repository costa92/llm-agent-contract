package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

// imageGenMock is a minimal ImageGenerator used only to prove the
// interface is satisfiable and to exercise ErrCapabilityNotSupported
// on a model that does not support image generation.
type imageGenMock struct {
	resp      ImageResponse
	supported bool
}

// Compile-time: imageGenMock implements ImageGenerator.
var _ ImageGenerator = (*imageGenMock)(nil)

func (m *imageGenMock) GenerateImage(_ context.Context, _ ImageRequest) (ImageResponse, error) {
	if !m.supported {
		return ImageResponse{}, fmt.Errorf("mock: image generation: %w", ErrCapabilityNotSupported)
	}
	return m.resp, nil
}

// ----- ImageGenerator interface is satisfiable + honours the sentinel -----
func TestImageGenerator_Interface(t *testing.T) {
	ctx := context.Background()

	// Capability-gated model returns the canonical sentinel.
	off := &imageGenMock{supported: false}
	if _, err := off.GenerateImage(ctx, ImageRequest{Prompt: "a cat"}); !errors.Is(err, ErrCapabilityNotSupported) {
		t.Fatalf("GenerateImage = %v, want ErrCapabilityNotSupported", err)
	}

	// Supported model returns its response unchanged.
	want := ImageResponse{
		Images:   []GeneratedImage{{Bytes: []byte{0x1, 0x2, 0x3}, MimeType: "image/png"}},
		Provider: "openai",
		Model:    "gpt-image-1",
		Usage:    Usage{InputTokens: 7, TotalTokens: 7, Source: UsageReported},
	}
	on := &imageGenMock{supported: true, resp: want}
	got, err := on.GenerateImage(ctx, ImageRequest{Prompt: "a cat", N: 1, Size: "1024x1024"})
	if err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}
	if got.Provider != "openai" || got.Model != "gpt-image-1" {
		t.Errorf("response identity = %s/%s, want openai/gpt-image-1", got.Provider, got.Model)
	}
	if len(got.Images) != 1 || string(got.Images[0].Bytes) != "\x01\x02\x03" {
		t.Errorf("Images = %+v, want one image with bytes 0x010203", got.Images)
	}
}

// ----- ImageRequest JSON round-trip -----
func TestImageRequest_JSONRoundTrip(t *testing.T) {
	in := ImageRequest{
		Prompt:  "a watercolour fox",
		N:       2,
		Size:    "1024x1024",
		Quality: "hd",
		Format:  "png",
		Extra:   map[string]any{"style": "vivid"},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"prompt":"a watercolour fox","n":2,"size":"1024x1024","quality":"hd","format":"png","extra":{"style":"vivid"}}`
	if string(b) != want {
		t.Errorf("Marshal:\n got  %s\n want %s", b, want)
	}

	var out ImageRequest
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out.Prompt != in.Prompt || out.N != in.N || out.Size != in.Size ||
		out.Quality != in.Quality || out.Format != in.Format {
		t.Errorf("round-trip scalar mismatch:\n got  %+v\n want %+v", out, in)
	}
	if out.Extra["style"] != "vivid" {
		t.Errorf("round-trip Extra = %+v, want style=vivid", out.Extra)
	}
}

// ----- ImageRequest omitempty: zero-value request stays minimal -----
func TestImageRequest_JSONOmitEmpty(t *testing.T) {
	b, err := json.Marshal(ImageRequest{Prompt: "x"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"prompt":"x"}`
	if string(b) != want {
		t.Errorf("Marshal zero-value:\n got  %s\n want %s", b, want)
	}
}

// ----- ImageResponse JSON round-trip (nested GeneratedImage + Usage) -----
func TestImageResponse_JSONRoundTrip(t *testing.T) {
	in := ImageResponse{
		Images: []GeneratedImage{
			{Bytes: []byte("PNG"), MimeType: "image/png", RevisedPrompt: "a fox, watercolour"},
			{URL: "https://example.com/img.png", MimeType: "image/png"},
		},
		Provider: "openai",
		Model:    "gpt-image-1",
		Usage:    Usage{InputTokens: 5, OutputTokens: 0, TotalTokens: 5, Source: UsageReported},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var out ImageResponse
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out.Provider != in.Provider || out.Model != in.Model {
		t.Errorf("identity mismatch: got %s/%s", out.Provider, out.Model)
	}
	if out.Usage != in.Usage {
		t.Errorf("Usage round-trip:\n got  %+v\n want %+v", out.Usage, in.Usage)
	}
	if len(out.Images) != 2 {
		t.Fatalf("Images len = %d, want 2", len(out.Images))
	}
	// Bytes survive base64 round-trip (encoding/json encodes []byte as base64).
	if string(out.Images[0].Bytes) != "PNG" {
		t.Errorf("Images[0].Bytes = %q, want PNG", out.Images[0].Bytes)
	}
	if out.Images[0].RevisedPrompt != "a fox, watercolour" {
		t.Errorf("Images[0].RevisedPrompt = %q", out.Images[0].RevisedPrompt)
	}
	if out.Images[1].URL != "https://example.com/img.png" {
		t.Errorf("Images[1].URL = %q", out.Images[1].URL)
	}
	if out.Images[1].Bytes != nil {
		t.Errorf("Images[1].Bytes = %v, want nil (URL-delivered image)", out.Images[1].Bytes)
	}
}
