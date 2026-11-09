package app_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xymaxim/ypb/internal/app"
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

func TestWithHostCheck(t *testing.T) {
	t.Parallel()

	const rebound = "rebind.example:9000" // a name a DNS rebinding page would use

	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name        string
		listenHost  string
		requestHost string
		want        int
	}{
		{"loopback, by IP", testHost, "127.0.0.1:9000", http.StatusOK},
		{"loopback, by name", testHost, "localhost:9000", http.StatusOK},
		{"loopback, by name in capitals", testHost, "LOCALHOST:9000", http.StatusOK},
		{"loopback, by IPv6", "::1", "[::1]:9000", http.StatusOK},
		{"loopback, no port", testHost, "127.0.0.1", http.StatusOK},
		{"loopback, another name", testHost, rebound, http.StatusForbidden},
		{"loopback, LAN address", testHost, "192.168.0.30:9000", http.StatusForbidden},
		{"localhost, another name", "localhost", rebound, http.StatusForbidden},
		{"all interfaces", "", rebound, http.StatusOK},
		{"all IPv4 interfaces", "0.0.0.0", rebound, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/info", nil)
			req.Host = tt.requestHost
			w := httptest.NewRecorder()
			app.WithHostCheck(tt.listenHost, ok).ServeHTTP(w, req)

			if w.Code != tt.want {
				t.Fatalf(
					"listening on %q, Host %q: status %d, want %d",
					tt.listenHost,
					tt.requestHost,
					w.Code,
					tt.want,
				)
			}
		})
	}
}
