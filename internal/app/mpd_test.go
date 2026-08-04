package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseLatencyParam(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		query    string
		expected time.Duration
		wantErr  bool
	}{
		{name: "absent", query: "", expected: 0},
		{name: "empty", query: "latency=", expected: 0},
		{name: "zero", query: "latency=0", expected: 0},
		{name: "integer seconds", query: "latency=10", expected: 10 * time.Second},
		{
			name:     "fractional seconds",
			query:    "latency=10.5",
			expected: 10500 * time.Millisecond,
		},
		{name: "negative", query: "latency=-1", wantErr: true},
		{name: "unparseable", query: "latency=abc", wantErr: true},
		{name: "short", query: "l=10", expected: 10 * time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/mpd/now?"+tc.query, nil)
			got, err := parseLatencyParam(req)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestMPDHandlerNowWithLatency(t *testing.T) {
	t.Parallel()

	// Playback is nil; validation must fail before it is used.
	h := &MPDHandler{}

	mux := http.NewServeMux()
	mux.HandleFunc(MPDPath, WithError(h.ServeHTTP))

	req := httptest.NewRequest(http.MethodGet, "/mpd/now?latency=30", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "cannot locate 'now' with latency")
}

func TestMPDHandlerNowMinusDurationWithLatency(t *testing.T) {
	t.Parallel()

	// Playback is nil; validation must fail before it is used.
	h := &MPDHandler{}

	mux := http.NewServeMux()
	mux.HandleFunc(MPDPath, WithError(h.ServeHTTP))

	req := httptest.NewRequest(http.MethodGet, "/mpd/now-10s?latency=30", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "within the latency window, not yet ingested")
}

func TestMPDHandlerIntervalWithinLatencyWindow(t *testing.T) {
	t.Parallel()

	// Playback is nil; validation must fail before it is used.
	h := &MPDHandler{}

	mux := http.NewServeMux()
	mux.HandleFunc(MPDPath, WithError(h.ServeHTTP))

	withinWindow := time.Now().Add(-10 * time.Second).UTC().Format(time.RFC3339)

	testCases := []struct {
		name string
		path string
	}{
		{
			name: "absolute end within the latency window",
			path: "/mpd/now-1h--" + withinWindow + "?latency=30",
		},
		{
			name: "absolute start within the latency window",
			path: "/mpd/" + withinWindow + "--0?latency=30",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
			require.Contains(
				t,
				w.Body.String(),
				"within the latency window, not yet ingested",
			)
		})
	}
}
