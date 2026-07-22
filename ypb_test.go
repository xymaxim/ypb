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
