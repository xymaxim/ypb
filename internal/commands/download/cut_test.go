package download

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xymaxim/ypb/internal/actions"
)

func expectedCutArgs(start, duration string, extra ...string) []string {
	args := []string{
		"-hide_banner",
		"-y",
		"-ss", start,
		"-i", "in.mp4",
		"-t", duration,
		"-c:a", "copy",
		"-avoid_negative_ts", "make_zero",
	}
	return append(args, extra...)
}

func TestCutArgs(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		opts     cutOptions
		expected []string
	}{
		{
			name: "plain range",
			opts: cutOptions{
				StartSeconds: 1.5,
				EndSeconds:   30.0,
			},
			expected: expectedCutArgs("1.500", "28.500"),
		},
		{
			name: "extra args appended",
			opts: cutOptions{
				StartSeconds:    0,
				EndSeconds:      10,
				ExtraFFmpegArgs: "-c:v libx264 -crf 20",
			},
			expected: expectedCutArgs(
				"0.000",
				"10.000",
				"-c:v",
				"libx264",
				"-crf",
				"20",
			),
		},
		{
			name: "quoted extra args",
			opts: cutOptions{
				StartSeconds:    2,
				EndSeconds:      5,
				ExtraFFmpegArgs: `-vf "scale=1280:-1"`,
			},
			expected: expectedCutArgs("2.000", "3.000", "-vf", "scale=1280:-1"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			args, err := cutArgs("in.mp4", tc.opts)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, args)
		})
	}
}

func TestCutArgsInvalidRange(t *testing.T) {
	t.Parallel()
	_, err := cutArgs("in.mp4", cutOptions{
		StartSeconds: 10,
		EndSeconds:   5,
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid cut range")
}

func TestCutArgsZeroDuration(t *testing.T) {
	t.Parallel()
	_, err := cutArgs("in.mp4", cutOptions{
		StartSeconds: 5,
		EndSeconds:   5,
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid cut range")
}

func TestCutArgsNegativeStart(t *testing.T) {
	t.Parallel()
	_, err := cutArgs("in.mp4", cutOptions{
		StartSeconds: -1,
		EndSeconds:   10,
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "must be non-negative")
}

func TestCutArgsInvalidExtraArgs(t *testing.T) {
	t.Parallel()
	_, err := cutArgs("in.mp4", cutOptions{
		StartSeconds:    0,
		EndSeconds:      5,
		ExtraFFmpegArgs: `-vf "unterminated`,
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "parsing extra ffmpeg options")
}

func TestCutOffsets(t *testing.T) {
	t.Parallel()
	actualStart := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	actualEnd := time.Date(2026, 1, 2, 10, 5, 0, 0, time.UTC)

	testCases := []struct {
		name          string
		inputStart    time.Time
		inputEnd      time.Time
		expectedStart float64
		expectedEnd   float64
	}{
		{
			name:          "inputs inside actual bounds",
			inputStart:    actualStart.Add(10 * time.Second),
			inputEnd:      actualEnd.Add(-10 * time.Second),
			expectedStart: 10,
			expectedEnd:   290,
		},
		{
			name:          "input start in a gap clamped to start",
			inputStart:    actualStart.Add(-10 * time.Second),
			inputEnd:      actualEnd.Add(-10 * time.Second),
			expectedStart: 0,
			expectedEnd:   290,
		},
		{
			name:          "input end in a gap clamped to end",
			inputStart:    actualStart.Add(10 * time.Second),
			inputEnd:      actualEnd.Add(30 * time.Second),
			expectedStart: 10,
			expectedEnd:   300,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := &actions.LocateOutputContext{
				ID:                  "test",
				Title:               "Test Title",
				StartSequenceNumber: 1,
				EndSequenceNumber:   2,
				ActualStartTime:     actualStart,
				ActualEndTime:       actualEnd,
				ActualDuration:      actualEnd.Sub(actualStart),
				InputStartTime:      tc.inputStart,
				InputEndTime:        tc.inputEnd,
				InputDuration:       tc.inputEnd.Sub(tc.inputStart),
			}

			start, end := cutOffsets(ctx)
			assert.InDelta(t, tc.expectedStart, start, 1e-9)
			assert.InDelta(t, tc.expectedEnd, end, 1e-9)
		})
	}
}
