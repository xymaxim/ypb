package app_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/xymaxim/ypb/internal/app"
)

const (
	testAllowed = "http://localhost:9000"
	testEvil    = "http://evil.example"
)

func TestServerAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		host string
		want string
	}{
		{"all interfaces", "", ":9000"},
		{"ipv4 loopback", "127.0.0.1", "127.0.0.1:9000"},
		{"ipv6 loopback", "::1", "[::1]:9000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := app.ServerAddr(tt.host, 9000)
			if got != tt.want {
				t.Fatalf(
					"ServerAddr(%q, 9000) = %q, want %q",
					tt.host,
					got,
					tt.want,
				)
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
		{"allowed", testAllowed, testAllowed},
		{"disallowed", testEvil, ""},
		{"no origin", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nextCalled := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				nextCalled = true
			})
			handler := app.WithCORS(map[string]bool{testAllowed: true}, next)

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
	handler := app.WithCORS(map[string]bool{testAllowed: true}, next)

	req := httptest.NewRequest(http.MethodOptions, "/info", nil)
	req.Header.Set("Origin", testAllowed)
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if nextCalled {
		t.Error("next handler was called for a preflight request")
	}
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != testAllowed {
		t.Errorf("Allow-Origin = %q, want %q", got, testAllowed)
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
	req.Header.Set("Origin", testAllowed)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !nextCalled {
		t.Fatal("next handler was not called")
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q, want none", got)
	}
}
