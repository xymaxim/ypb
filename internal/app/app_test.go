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

	const (
		testAddr        = "127.0.0.1:9000"
		testAllowedHost = "ypb.local"

		// reboundHost is the Host header an attacker page would send after
		// rebinding its domain to the local server.
		reboundHost = "rebind.example:9000"
	)

	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name        string
		listenAddr  string
		allowedHost string
		requestHost string
		want        int
	}{
		{"loopback, by IP", testAddr, "", "127.0.0.1:9000", http.StatusOK},
		{"loopback, by name", testAddr, "", "localhost:9000", http.StatusOK},
		{"loopback, by name in capitals", testAddr, "", "LOCALHOST:9000", http.StatusOK},
		{"loopback, by IPv6", "[::1]:9000", "", "[::1]:9000", http.StatusOK},
		{"loopback, no port", testAddr, "", "127.0.0.1", http.StatusOK},
		{"loopback, another name", testAddr, "", reboundHost, http.StatusForbidden},
		{"loopback, LAN address", testAddr, "", "192.168.0.30:9000", http.StatusForbidden},
		{
			"localhost, another name",
			"localhost:9000",
			"",
			reboundHost,
			http.StatusForbidden,
		},
		{"allowed host", testAddr, testAllowedHost, "ypb.local:9000", http.StatusOK},
		{
			"allowed host in capitals",
			testAddr,
			testAllowedHost,
			"YPB.LOCAL:9000",
			http.StatusOK,
		},
		{"allowed host, no port", testAddr, testAllowedHost, "ypb.local", http.StatusOK},
		{
			"allowed host is exact",
			testAddr,
			testAllowedHost,
			"ypb.local.evil:9000",
			http.StatusForbidden,
		},
		{
			"other name with allowed host",
			testAddr,
			testAllowedHost,
			reboundHost,
			http.StatusForbidden,
		},
		{"all interfaces", ":9000", testAllowedHost, reboundHost, http.StatusOK},
		{
			"all IPv4 interfaces",
			"0.0.0.0:9000",
			testAllowedHost,
			reboundHost,
			http.StatusOK,
		},
		{"all IPv6 interfaces", "[::]:9000", testAllowedHost, reboundHost, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/info", nil)
			req.Host = tt.requestHost
			w := httptest.NewRecorder()
			app.WithHostCheck(tt.listenAddr, tt.allowedHost, ok).ServeHTTP(w, req)

			if w.Code != tt.want {
				t.Fatalf(
					"listening on %q, allowed Host %q, Host %q: status %d, want %d",
					tt.listenAddr,
					tt.allowedHost,
					tt.requestHost,
					w.Code,
					tt.want,
				)
			}
		})
	}
}
