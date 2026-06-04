package llm

import (
	"encoding/json"
	"testing"
)

// ----- Message.Images is additive: a text-only message stays unchanged -----
func TestMessage_JSONOmitsImagesWhenEmpty(t *testing.T) {
	b, err := json.Marshal(Message{Role: "user", Content: "hello"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"role":"user","content":"hello"}`
	if string(b) != want {
		t.Errorf("Marshal text-only message:\n got  %s\n want %s", b, want)
	}
}

// ----- Message with attached images round-trips (URL + Bytes variants) -----
func TestMessage_ImagesJSONRoundTrip(t *testing.T) {
	in := Message{
		Role:    "user",
		Content: "what is in these images?",
		Images: []MessageImage{
			{URL: "data:image/png;base64,iVBORw0KGgo=", MimeType: "image/png", Detail: "high"},
			{Bytes: []byte("JPG"), MimeType: "image/jpeg"},
		},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var out Message
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out.Role != in.Role || out.Content != in.Content {
		t.Errorf("scalar mismatch: got %s/%q", out.Role, out.Content)
	}
	if len(out.Images) != 2 {
		t.Fatalf("Images len = %d, want 2", len(out.Images))
	}
	if out.Images[0].URL != in.Images[0].URL || out.Images[0].Detail != "high" {
		t.Errorf("Images[0] = %+v", out.Images[0])
	}
	// Bytes survive the encoding/json base64 round-trip.
	if string(out.Images[1].Bytes) != "JPG" || out.Images[1].MimeType != "image/jpeg" {
		t.Errorf("Images[1] = %+v, want bytes=JPG mime=image/jpeg", out.Images[1])
	}
	// URL-delivered image has no inline bytes.
	if out.Images[0].Bytes != nil {
		t.Errorf("Images[0].Bytes = %v, want nil", out.Images[0].Bytes)
	}
}

// ----- Capabilities.Vision is a distinct, JSON-serialised flag -----
func TestCapabilities_VisionFlag(t *testing.T) {
	b, err := json.Marshal(Capabilities{Vision: true})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"tools":false,"embeddings":false,"structured_outputs":false,"prompt_caching":false,"image_generation":false,"vision":true}`
	if string(b) != want {
		t.Errorf("Marshal:\n got  %s\n want %s", b, want)
	}
}
