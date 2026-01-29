package playback_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/xymaxim/ypb/internal/playback"
)

type fakePlayback struct {
	*playback.Playback
	baseURLs map[string]string
}

func newFakePlayback() *fakePlayback {
	return &fakePlayback{
		baseURLs: map[string]string{"0": "initial"},
	}
}

func (pb *fakePlayback) BaseURLs() map[string]string {
	return pb.baseURLs
}

func (pb *fakePlayback) RefreshBaseURLs() error {
	pb.baseURLs = map[string]string{"0": "refreshed"}
	return nil
}

func TestClient_CheckRetry_StatusForbidden(t *testing.T) {
	pb := newFakePlayback()
	client := playback.NewClient(pb)
	client.RetryWaitMax = time.Millisecond

	var requestCount int
	attemptsBeforeOK := client.RetryMax - 1

	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			if requestCount <= attemptsBeforeOK {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer ts.Close()

	_, err := client.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	wantBaseURLs := map[string]string{"0": "refreshed"}
	if diff := cmp.Diff(pb.BaseURLs(), wantBaseURLs); diff != "" {
		t.Fatalf("base URLs mismatched after retries (- got, + want):\n%s", diff)
	}
}

func TestClient_CheckRetry_StatusServiceUnavailable(t *testing.T) {
	pb := newFakePlayback()
	client := playback.NewClient(pb)
	client.RetryWaitMax = time.Millisecond

	var requestCount int
	attemptsBeforeOK := client.RetryMax - 1

	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			if requestCount <= attemptsBeforeOK {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer ts.Close()

	_, err := client.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	wantBaseURLs := map[string]string{"0": "initial"}
	if diff := cmp.Diff(pb.BaseURLs(), wantBaseURLs); diff != "" {
		t.Fatalf("base URLs mismatch (- got, + want):\n%s", diff)
	}
}
