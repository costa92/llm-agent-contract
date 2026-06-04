package llm

import "context"

// ImageGenerator is the capability for text-to-image generation. Like
// Embedder, it deliberately does NOT embed ChatModel: a provider's image
// endpoint is orthogonal to chat. Providers without an image endpoint
// (anthropic, deepseek, ollama) do NOT implement this interface; callers
// detect support via type assertion AND consult Capabilities.ImageGeneration
// on the bound ProviderInfo.
//
// A bound model that implements the Go interface but cannot generate images
// returns ErrCapabilityNotSupported:
//
//	return ImageResponse{}, fmt.Errorf("openai: image generation: %w", ErrCapabilityNotSupported)
type ImageGenerator interface {
	GenerateImage(ctx context.Context, req ImageRequest) (ImageResponse, error)
}

// ImageRequest describes a text-to-image call. Providers ignore knobs they
// do not support (documented per provider). Only Prompt is required.
type ImageRequest struct {
	Prompt  string         `json:"prompt"`            // required
	N       int            `json:"n,omitempty"`       // number of images; 0 => provider default (1)
	Size    string         `json:"size,omitempty"`    // e.g. "1024x1024"; "" => provider default
	Quality string         `json:"quality,omitempty"` // e.g. "standard"/"hd"/"high"; "" => provider default
	Format  string         `json:"format,omitempty"`  // output encoding "png"/"jpeg"/"webp"; "" => provider default
	Extra   map[string]any `json:"extra,omitempty"`   // provider-specific knobs, forwarded verbatim
}

// ImageResponse is the result, in request order.
type ImageResponse struct {
	Images   []GeneratedImage `json:"images"`
	Provider string           `json:"provider"`
	Model    string           `json:"model,omitempty"`
	Usage    Usage            `json:"usage"` // best-effort; zero when the provider does not report it
}

// GeneratedImage is one produced image. Exactly one of Bytes or URL is
// populated, chosen by the provider's most direct delivery path (the caller
// does NOT request URL-vs-bytes): OpenAI b64 => Bytes; Volcengine/Minimax
// url => URL; Google always => Bytes.
type GeneratedImage struct {
	Bytes         []byte `json:"bytes,omitempty"`          // inline bytes (base64-decoded) when returned inline
	URL           string `json:"url,omitempty"`            // hosted link when the provider returns a URL
	MimeType      string `json:"mime_type,omitempty"`      // e.g. "image/png"; "" if unknown
	RevisedPrompt string `json:"revised_prompt,omitempty"` // provider's rewritten prompt, if any (dall-e-3)
}
