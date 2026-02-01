package playback_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/xymaxim/ypb/internal/playback"
)

type fakePlayback struct {
	*playback.Playback
	addr     string
	baseURLs map[string]string
}

func newFakePlayback(addr string) *fakePlayback {
	return &fakePlayback{
		addr: addr,
		baseURLs: map[string]string{
			"0": strings.TrimRight(addr, "/") + "/initial/itag/0",
		},
	}
}

func (pb *fakePlayback) BaseURLs() map[string]string {
	return pb.baseURLs
}

func (pb *fakePlayback) RefreshBaseURLs() error {
	pb.baseURLs = map[string]string{
		"0": strings.TrimRight(pb.addr, "/") + "/refreshed/itag/0",
	}
	return nil
}

func TestClient_CheckRetry(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name            string
		statusCode      int
		path            string
		wantPath        string
		wantBaseURLPath string
	}{
		{
			name:            "service unavailable - head sequence number",
			statusCode:      http.StatusServiceUnavailable,
			path:            "/initial/itag/0",
			wantPath:        "/initial/itag/0",
			wantBaseURLPath: "/initial/itag/0",
		},
		{
			name:            "service unavailable - segment",
			statusCode:      http.StatusServiceUnavailable,
			path:            "/initial/itag/0/sq/0",
			wantPath:        "/initial/itag/0/sq/0",
			wantBaseURLPath: "/initial/itag/0",
		},
		{
			name:            "forbidden - head sequence number",
			statusCode:      http.StatusForbidden,
			path:            "/initial/itag/0",
			wantPath:        "/refreshed/itag/0",
			wantBaseURLPath: "/refreshed/itag/0",
		},
		{
			name:            "forbidden - segment",
			statusCode:      http.StatusForbidden,
			path:            "/initial/itag/0/sq/0",
			wantPath:        "/refreshed/itag/0/sq/0",
			wantBaseURLPath: "/refreshed/itag/0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var requestCount int
			attemptsBeforeOK := 2
			ts := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requestCount++
					if requestCount <= attemptsBeforeOK {
						w.WriteHeader(tc.statusCode)
						return
					}
					w.WriteHeader(http.StatusOK)
				}),
			)
			defer ts.Close()

			pb := newFakePlayback(ts.URL)
			client := playback.NewClient(pb)
			client.RetryWaitMax = time.Millisecond
			client.RetryMax = attemptsBeforeOK + 1

			u, err := url.JoinPath(ts.URL, tc.path)
			if err != nil {
				t.Fatal(err)
			}

			resp, err := client.Get(u)
			if err != nil {
				t.Fatal(err)
			}

			wantBaseURLs := map[string]string{
				"0": strings.TrimRight(ts.URL, "/") + tc.wantBaseURLPath,
			}
			if diff := cmp.Diff(pb.BaseURLs(), wantBaseURLs); diff != "" {
				t.Errorf("base URLs mismatch (- have, + want):\n%s", diff)
			}

			haveURL := resp.Request.Header.Get("X-Request-Url")
			wantURL := strings.TrimRight(ts.URL, "/") + tc.wantPath
			if haveURL != wantURL {
				t.Errorf(
					"request URL mismatch:\n have: %s\n want: %s",
					haveURL,
					wantURL,
				)
			}
		})
	}
}
