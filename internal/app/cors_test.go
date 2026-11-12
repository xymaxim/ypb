package app_test

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/xymaxim/ypb/internal/app"
)

const (
	testAllowedOrigin    = "http://localhost:9000"
	testDisallowedOrigin = "http://evil.example"
	testInvalidOrigin    = "localhost:9000"
)

func TestNewOriginSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		origins []string
		want    map[string]bool
	}{
		{
			name: "empty",
			want: map[string]bool{},
		},
		{
			name:    "normalized",
			origins: []string{" HTTP://LOCALHOST:9000/ ", ""},
			want:    map[string]bool{testAllowedOrigin: true},
		},
		{
			name:    "ipv6",
			origins: []string{"http://[::1]:9000"},
			want:    map[string]bool{"http://[::1]:9000": true},
		},
		{
			name:    "default port left as-is",
			origins: []string{"https://example.com:443"},
			want:    map[string]bool{"https://example.com:443": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := app.NewOriginSet(tt.origins)
			if err != nil {
				t.Fatalf("NewOriginSet(%q) returned error: %v", tt.origins, err)
			}
			if !maps.Equal(got, tt.want) {
				t.Fatalf("NewOriginSet(%q) = %v, want %v", tt.origins, got, tt.want)
			}
		})
	}
}

func TestNewOriginSetRejectsInvalid(t *testing.T) {
	t.Parallel()

	tests := []string{
		"*",
		"null",
		testInvalidOrigin,
		"http://",
		"http://example.com/path",
		"http://example.com?x",
		"http://example.com#x",
		"http://user:pass@example.com",
		"http://example.com//",
		"ftp://example.com",
	}

	for _, origin := range tests {
		t.Run(origin, func(t *testing.T) {
			t.Parallel()

			if _, err := app.NewOriginSet([]string{origin}); err == nil {
				t.Fatalf("NewOriginSet(%q) returned no error", origin)
			}
		})
	}
}

func TestWithCORS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		origin     string
		wantOrigin string
	}{
		{"allowed", testAllowedOrigin, testAllowedOrigin},
		{"disallowed", testDisallowedOrigin, ""},
		{"no origin", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nextCalled := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				nextCalled = true
			})
			handler := app.WithCORS(map[string]bool{testAllowedOrigin: true}, next)

			req := httptest.NewRequest(http.MethodGet, "/info", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if !nextCalled {
				t.Error("next handler was not called")
			}
			if got := w.Header().
				Get("Access-Control-Allow-Origin"); got != tt.wantOrigin {
				t.Errorf("Allow-Origin = %q, want %q", got, tt.wantOrigin)
			}
			if !slices.Contains(w.Header().Values("Vary"), "Origin") {
				t.Error("missing Vary: Origin")
			}
		})
	}
}

func TestWithCORSPreflight(t *testing.T) {
	t.Parallel()

	nextCalled := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	})
	handler := app.WithCORS(map[string]bool{testAllowedOrigin: true}, next)

	req := httptest.NewRequest(http.MethodOptions, "/info", nil)
	req.Header.Set("Origin", testAllowedOrigin)
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if nextCalled {
		t.Error("next handler was called for a preflight request")
	}
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != testAllowedOrigin {
		t.Errorf("Allow-Origin = %q, want %q", got, testAllowedOrigin)
	}
	if !strings.Contains(w.Header().Get("Access-Control-Allow-Methods"), "GET") {
		t.Errorf(
			"Allow-Methods = %q, want it to include GET",
			w.Header().Get("Access-Control-Allow-Methods"),
		)
	}
}

func TestWithCORSDisabled(t *testing.T) {
	t.Parallel()

	nextCalled := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	})
	handler := app.WithCORS(nil, next)

	req := httptest.NewRequest(http.MethodGet, "/info", nil)
	req.Header.Set("Origin", testAllowedOrigin)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !nextCalled {
		t.Fatal("next handler was not called")
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q, want none", got)
	}
}
