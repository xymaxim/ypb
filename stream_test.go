package ypb_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xymaxim/ypb"
	"github.com/xymaxim/ypb/internal/testutil"
)

func TestCORS(t *testing.T) {
	fetcher := &testutil.MockFetcher{VideoID: testutil.TestVideoID}

	s, err := ypb.NewStream(
		context.Background(),
		testutil.TestVideoID,
		0,
		&ypb.StreamConfig{
			Fetcher: fetcher,
		},
	)
	if err != nil {
		t.Fatalf("creating stream: %v", err)
	}

	handler := s.Server().Handler

	req := httptest.NewRequest(http.MethodGet, "/info", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("missing Access-Control-Allow-Origin header")
	}
}

func TestHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		host string
		want string
	}{
		{"default: all interfaces", "", ":0"},
		{"this computer only", "127.0.0.1", "127.0.0.1:0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, err := ypb.NewStream(
				context.Background(),
				testutil.TestVideoID,
				0,
				&ypb.StreamConfig{
					Fetcher: &testutil.MockFetcher{
						VideoID: testutil.TestVideoID,
					},
					Host: tt.host,
				},
			)
			if err != nil {
				t.Fatalf("creating stream: %v", err)
			}
			defer s.Stop()

			if got := s.Server().Addr; got != tt.want {
				t.Fatalf("server address = %q, want %q", got, tt.want)
			}
		})
	}
}
