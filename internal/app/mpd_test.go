package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xymaxim/ypb/internal/actions"
	"github.com/xymaxim/ypb/internal/input"
	"github.com/xymaxim/ypb/playback"
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

func TestMPDHandlerNowPlus(t *testing.T) {
	t.Parallel()

	// Playback is nil; validation must fail before it is used.
	h := &MPDHandler{}

	mux := http.NewServeMux()
	mux.HandleFunc(MPDPath, WithError(h.ServeHTTP))

	req := httptest.NewRequest(http.MethodGet, "/mpd/now+1m", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "cannot locate 'now + 1m0s': in the future")
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

func TestDynamicCacheKeyEquivalence(t *testing.T) {
	t.Parallel()

	// Same instant spelled with different zones must produce the same key.
	target := time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC)
	offsetSpelling := time.Date(
		2026, 1, 2, 12, 20, 30, 0,
		time.FixedZone("UTC+2", 2*3600),
	)

	key1, err := dynamicCacheKey(target, 0)
	require.NoError(t, err)
	key2, err := dynamicCacheKey(offsetSpelling, 0)
	require.NoError(t, err)
	require.Equal(t, key1, key2)

	// Latency shifts the located segment, so it must be part of the key.
	keyLatency, err := dynamicCacheKey(target, 30*time.Second)
	require.NoError(t, err)
	require.NotEqual(t, key1, keyLatency)
}

func TestDynamicCacheKeyVariants(t *testing.T) {
	t.Parallel()

	target := time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC)
	targetKey := fmt.Sprintf("t:%d|0", target.UTC().Truncate(time.Second).Unix())

	testCases := []struct {
		name    string
		value   input.MomentValue
		latency time.Duration
		want    string
	}{
		{
			name:    "absolute time",
			value:   target,
			latency: 0,
			want:    targetKey,
		},
		{
			name:    "sequence number",
			value:   playback.SequenceNumber(123),
			latency: 0,
			want:    "sq:123|0",
		},
		{
			name:    "now keyword",
			value:   input.NowKeyword,
			latency: 0,
			want:    "k:now|0",
		},
		{
			name: "expression with now left",
			value: input.MomentExpression{
				Operator: input.OpMinus,
				Left:     input.NowKeyword,
				Right:    30 * time.Second,
			},
			latency: 0,
			want:    "e:-30000000000|0",
		},
		{
			name: "expression with time left",
			value: input.MomentExpression{
				Operator: input.OpMinus,
				Left:     time.Date(2026, 1, 2, 10, 20, 45, 0, time.UTC),
				Right:    15 * time.Second,
			},
			latency: 0,
			want:    targetKey,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := dynamicCacheKey(tc.value, tc.latency)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestDynamicCacheGetSet(t *testing.T) {
	h := &MPDHandler{}

	_, ok := h.dynamicCache.get("k")
	require.False(t, ok)

	entry := dynamicCacheEntry{
		AvailabilityStartTime: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
		StartActualTime:       time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
		StartTargetTime:       time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
		probe: actions.DynamicMomentProbe{
			StartNumber: 5,
			AnchorTime:  time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
			AudioPTS:    1000,
			VideoPTS:    2000,
		},
	}
	h.dynamicCache.set("k", entry)

	got, ok := h.dynamicCache.get("k")
	require.True(t, ok)
	require.Equal(t, entry.AvailabilityStartTime, got.AvailabilityStartTime)
	require.Equal(t, entry.StartActualTime, got.StartActualTime)
	require.Equal(t, entry.StartTargetTime, got.StartTargetTime)
	require.Equal(t, entry.probe, got.probe)
}

func TestDynamicCacheIdleEviction(t *testing.T) {
	h := &MPDHandler{}

	h.dynamicCache.set("k", dynamicCacheEntry{})

	// Age the entry beyond the idle timeout.
	h.dynamicCache.mu.Lock()
	h.dynamicCache.items["k"] = dynamicCacheEntry{
		LastAccess: time.Now().Add(-(dynamicCacheIdleTimeout + time.Hour)),
	}
	h.dynamicCache.mu.Unlock()

	_, ok := h.dynamicCache.get("k")
	require.False(t, ok)
	require.Empty(t, h.dynamicCache.items)
}

func TestDynamicCacheOverflow(t *testing.T) {
	h := &MPDHandler{}

	// Exceed the max entry count; the cache resets, keeping only the newest entry.
	for i := 0; i < dynamicCacheMaxEntries+1; i++ {
		h.dynamicCache.set(
			fmt.Sprintf("k%d", i),
			dynamicCacheEntry{},
		)
	}

	require.Len(t, h.dynamicCache.items, 1)
	require.Contains(t, h.dynamicCache.items, fmt.Sprintf("k%d", dynamicCacheMaxEntries))

	_, ok := h.dynamicCache.get("k0")
	require.False(t, ok)
}
