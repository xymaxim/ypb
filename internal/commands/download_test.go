package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

//nolint:paralleltest
func TestAdjustForFilename(t *testing.T) {
	testCases := []struct {
		name     string
		s        string
		length   int
		expected string
	}{
		{
			name:     "french title with default length",
			s:        "En direct : Titre de la   vidéo — 24h/7 | Panorama, 360 / ? ",
			length:   0,
			expected: "En-direct-Titre-de-la-video",
		},
		{
			name:     "french title with full length",
			s:        "En direct : Titre de la   vidéo — 24h/7 | Panorama, 360 / ? ",
			length:   1000,
			expected: "En-direct-Titre-de-la-video-24h-7-Panorama-360",
		},
		{
			name:     "french title without trimmed word",
			s:        "En direct : Titre de la   vidéo — 24h/7 | Panorama, 360 / ? ",
			length:   5,
			expected: "En",
		},
		{
			name: "japanese title",
			//nolint:gosmopolitan
			s:        "【LIVE】新宿駅前の様子 Shinjuku, Tokyo JAPAN【ライブカメラ】 | TBS NEWS DIG",
			length:   50,
			expected: "LIVE-Xin-Su-Yi-Qian-noYang-Zi-Shinjuku-Tokyo-JAPAN",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, adjustForFilename(tc.s, tc.length))
		})
	}
}

func TestFormatTime(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		ts       string
		expected string
	}{
		{
			name:     "time in UTC",
			ts:       "2026-01-02T10:20:30+00:00",
			expected: "20260102T102030+00",
		},
		{
			name:     "time in custom time zone",
			ts:       "2026-01-02T10:20:30-02:00",
			expected: "20260102T102030-02",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tt, err := time.Parse(time.RFC3339, tc.ts)
			if err != nil {
				t.Fatalf("parsing input time string: %v", err)
			}
			assert.Equal(t, tc.expected, formatTime(tt))
		})
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		dur      time.Duration
		expected string
	}{
		{
			name:     "3s",
			dur:      3 * time.Second,
			expected: "3s",
		},
		{
			name:     "2m",
			dur:      2 * time.Minute,
			expected: "2m",
		},
		{
			name:     "1h",
			dur:      time.Hour,
			expected: "1h",
		},
		{
			name:     "100h",
			dur:      100 * time.Hour,
			expected: "100h",
		},
		{
			name:     "1h3s",
			dur:      time.Hour + 3*time.Second,
			expected: "1h3s",
		},
		{
			name:     "2m3s",
			dur:      2*time.Minute + 3*time.Second,
			expected: "2m3s",
		},
		{
			name:     "1h2m3s",
			dur:      time.Hour + 2*time.Minute + 3*time.Second,
			expected: "1h2m3s",
		},
		{
			name:     "3.6s",
			dur:      3*time.Second + 600*time.Millisecond,
			expected: "3s",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, formatDuration(tc.dur))
		})
	}
}

func TestFormatDifference(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		diff     time.Duration
		showPlus bool
		expected string
	}{
		{
			name:     "+1s",
			diff:     time.Second,
			showPlus: true,
			expected: "+1s",
		},
		{
			name:     "1s",
			diff:     time.Second,
			showPlus: false,
			expected: "1s",
		},
		{
			name:     "-1s",
			diff:     -time.Second,
			showPlus: true,
			expected: "-1s",
		},
		{
			name:     "0s",
			diff:     time.Duration(0),
			showPlus: true,
			expected: "0s",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, formatDifference(tc.diff, tc.showPlus))
		})
	}
}
