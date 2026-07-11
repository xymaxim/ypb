package urlutil

import (
	"net/url"
	"testing"
)

func TestExtractParameter(t *testing.T) {
	rawURL := "https://example.com/videoplayback?itag=140&mime=audio%2Fmp4"

	tests := []struct {
		name string
		key  string
		want string
	}{
		{"itag", "itag", "140"},
		{"mime", "mime", "audio/mp4"},
		{"missing", "sq", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractParameter(rawURL, tt.key)
			if got != tt.want {
				t.Fatalf("ExtractParameter(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestBuildSegmentURL(t *testing.T) {
	got, err := BuildSegmentURL(
		"https://example.com/videoplayback?itag=140",
		"123",
	)
	if err != nil {
		t.Fatalf("BuildSegmentURL() error = %v", err)
	}

	want := "https://example.com/videoplayback?itag=140&sq=123"
	if got.String() != want {
		t.Fatalf("BuildSegmentURL() = %q, want %q", got.String(), want)
	}
}

func TestBuildSegmentURLFromParsed(t *testing.T) {
	got, err := url.Parse("https://example.com/videoplayback?itag=140")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	out := BuildSegmentURLFromParsed(got, "123")
	want := "https://example.com/videoplayback?itag=140&sq=123"

	if out.String() != want {
		t.Fatalf("BuildSegmentURLFromParsed() = %q, want %q", out.String(), want)
	}
}
