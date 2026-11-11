package app_test

import (
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
