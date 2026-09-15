package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		"-shortest",
		"-fflags", "+genpts",
	}
	return append(args, extra...)
}

func TestCutArgs(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		opts     CutOptions
		expected []string
	}{
		{
			name: "plain range",
			opts: CutOptions{
				StartSeconds: 1.5,
				EndSeconds:   30.0,
			},
			expected: expectedCutArgs("1.500", "28.500"),
		},
		{
			name: "extra args appended",
			opts: CutOptions{
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
			opts: CutOptions{
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
	_, err := cutArgs("in.mp4", CutOptions{
		StartSeconds: 10,
		EndSeconds:   5,
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid cut range")
}

func TestCutArgsZeroDuration(t *testing.T) {
	t.Parallel()
	_, err := cutArgs("in.mp4", CutOptions{
		StartSeconds: 5,
		EndSeconds:   5,
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid cut range")
}

func TestCutArgsNegativeStart(t *testing.T) {
	t.Parallel()
	_, err := cutArgs("in.mp4", CutOptions{
		StartSeconds: -1,
		EndSeconds:   10,
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "must be non-negative")
}

func TestCutArgsInvalidExtraArgs(t *testing.T) {
	t.Parallel()
	_, err := cutArgs("in.mp4", CutOptions{
		StartSeconds:    0,
		EndSeconds:      5,
		ExtraFFmpegArgs: `-vf "unterminated`,
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "parsing extra ffmpeg options")
}
