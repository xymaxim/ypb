package app_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
		testAddr = "127.0.0.1:9000"
		ipv6Addr = "[::1]:9000"
		rebound  = "rebind.example:9000" // a name a DNS rebinding page would use
	)

	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name        string
		listenAddr  string
		requestHost string
		want        int
	}{
		{"loopback, by IP", testAddr, "127.0.0.1:9000", http.StatusOK},
		{"loopback, by name", testAddr, "localhost:9000", http.StatusOK},
		{"loopback, by name in capitals", testAddr, "LOCALHOST:9000", http.StatusOK},
		{"loopback, by IPv6", ipv6Addr, ipv6Addr, http.StatusOK},
		{"loopback, no port", testAddr, "127.0.0.1", http.StatusOK},
		{"loopback, another name", testAddr, rebound, http.StatusForbidden},
		{"loopback, LAN address", testAddr, "192.168.0.30:9000", http.StatusForbidden},
		{"localhost, another name", "localhost:9000", rebound, http.StatusForbidden},
		{"all interfaces, IPv4", "0.0.0.0:9000", rebound, http.StatusOK},
		{"all interfaces, IPv6", "[::]:9000", rebound, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/info", nil)
			req.Host = tt.requestHost
			w := httptest.NewRecorder()
			app.WithHostCheck(tt.listenAddr, ok).ServeHTTP(w, req)

			if w.Code != tt.want {
				t.Fatalf(
					"listening on %q, Host %q: status %d, want %d",
					tt.listenAddr,
					tt.requestHost,
					w.Code,
					tt.want,
				)
			}
		})
	}
}

func TestWithHostCheckOnBoundAddress(t *testing.T) {
	t.Parallel()

	ln := listen(t, 0)
	addr := ln.Addr().String()
	server := &http.Server{
		Handler: app.WithHostCheck(
			addr,
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = server.Serve(ln) }()
	t.Cleanup(func() { _ = server.Close() })

	for _, tt := range []struct {
		host string
		want int
	}{
		{"rebind.example", http.StatusForbidden},
		{addr, http.StatusOK},
	} {
		req, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			"http://"+addr+"/info",
			nil,
		)
		if err != nil {
			t.Fatalf("creating request: %v", err)
		}
		req.Host = tt.host

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET %s with Host %q: %v", addr, tt.host, err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != tt.want {
			t.Fatalf("Host %q: status %d, want %d", tt.host, resp.StatusCode, tt.want)
		}
	}
}
