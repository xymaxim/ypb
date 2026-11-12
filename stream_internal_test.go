package ypb

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xymaxim/ypb/internal/testutil"
)

func TestServeWrapsHostCheckWithBoundAddress(t *testing.T) {
	t.Parallel()

	s, err := NewStream(
		context.Background(),
		testutil.TestVideoID,
		"127.0.0.1",
		0,
		&StreamConfig{
			Fetcher: &testutil.MockFetcher{VideoID: testutil.TestVideoID},
		},
	)
	if err != nil {
		t.Fatalf("creating stream: %v", err)
	}
	t.Cleanup(s.Stop)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listening: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("closing listener: %v", err)
	}

	if err := s.serve(ln); err == nil {
		t.Fatal("serving on a closed listener: expected an error")
	}

	for _, tt := range []struct {
		name string
		host string
		want int
	}{
		{"rebound Host", "rebind.example:9000", http.StatusForbidden},
		{"bound Host", addr, http.StatusOK},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/info", nil)
			req.Host = tt.host
			w := httptest.NewRecorder()
			s.Server().Handler.ServeHTTP(w, req)

			if w.Code != tt.want {
				t.Fatalf("Host %q: status %d, want %d", tt.host, w.Code, tt.want)
			}
		})
	}
}
