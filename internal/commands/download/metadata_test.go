package download

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xymaxim/ypb/internal/actions"
	"github.com/xymaxim/ypb/internal/exec"
)

type recordingRunner struct {
	args []string
}

func (r *recordingRunner) Run(_ context.Context, args ...string) error {
	r.args = append(r.args, args...)
	if err := os.WriteFile(args[len(args)-1], []byte("out"), 0o644); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}
	return nil
}

func (r *recordingRunner) RunWith(
	_ context.Context,
	_ []exec.Option,
	args ...string,
) (*exec.RunResult, error) {
	r.args = append(r.args, args...)
	if err := os.WriteFile(args[len(args)-1], []byte("out"), 0o644); err != nil {
		return nil, fmt.Errorf("writing output: %w", err)
	}
	return &exec.RunResult{Stdout: []byte{}, Stderr: []byte{}}, nil
}

func TestFormatISO8601(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		in       time.Time
		expected string
	}{
		{
			name:     "milliseconds present",
			in:       time.Date(2026, 1, 2, 10, 20, 30, 123000000, time.UTC),
			expected: "2026-01-02T10:20:30.123Z",
		},
		{
			name:     "zero milliseconds",
			in:       time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
			expected: "2026-01-02T10:20:30.000Z",
		},
		{
			name: "non-UTC normalized",
			in: time.Date(
				2026,
				1,
				2,
				12,
				20,
				30,
				123000000,
				time.FixedZone("+02:00", 2*3600),
			),
			expected: "2026-01-02T10:20:30.123Z",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, formatISO8601(tc.in))
		})
	}
}

func TestMetadataTags(t *testing.T) {
	t.Parallel()
	metadata_ctx := &metadataContext{
		LocateOutputContext: actions.LocateOutputContext{
			ID:                  "abcdefgh123",
			Title:               "Test Title",
			StartSequenceNumber: 1,
			EndSequenceNumber:   2,
			ActualStartTime:     time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC),
			ActualEndTime:       time.Date(2026, 1, 2, 10, 5, 1, 0, time.UTC),
			ActualDuration:      5 * time.Minute,
			InputStartTime:      time.Date(2026, 1, 2, 10, 0, 1, 0, time.UTC),
			InputEndTime:        time.Date(2026, 1, 2, 10, 5, 0, 0, time.UTC),
			InputDuration:       5 * time.Minute,
		},
		ChannelTitle: "Test Channel",
	}

	assert.Equal(t, [][2]string{
		{"Title", "Test Title"},
		{"Author", "Test Channel"},
		{"Comment", "https://www.youtube.com/watch?v=abcdefgh123"},
		{"ActualStartTime", "2026-01-02T10:00:00.000Z"},
		{"InputStartTime", "2026-01-02T10:00:01.000Z"},
		{"ActualEndTime", "2026-01-02T10:05:01.000Z"},
		{"InputEndTime", "2026-01-02T10:05:00.000Z"},
		{"StartSegment", "1"},
		{"EndSegment", "2"},
	}, metadataTags(metadata_ctx))
}

func TestEmbedMetadata(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	stem := filepath.Join(tmpDir, "video")
	file := stem + ".mp4"
	require.NoError(t, os.WriteFile(file, []byte("in"), 0o644))

	runner := &recordingRunner{args: make([]string, 0)}
	tags := [][2]string{
		{"Title", "Test Title"},
		{"StartSegment", "1"},
	}

	require.NoError(t, embedMetadata(runner, file, tags))

	require.GreaterOrEqual(t, len(runner.args), 1)
	tmpArg := runner.args[len(runner.args)-1]
	assert.True(t, strings.HasSuffix(tmpArg, ".mp4"))
	assert.NotEqual(t, file, tmpArg)

	assert.Equal(t, []string{
		"-y",
		"-i", file,
		"-map", "0",
		"-c", "copy",
		"-movflags", "use_metadata_tags",
		"-metadata", "Title=Test Title",
		"-metadata", "StartSegment=1",
		tmpArg,
	}, runner.args)

	out, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, "out", string(out))

	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "video.mp4", entries[0].Name())
}

func TestEmbedMetadataNoExtensionSkips(t *testing.T) {
	t.Parallel()

	file := filepath.Join(t.TempDir(), "video")
	require.NoError(t, os.WriteFile(file, []byte("in"), 0o644))

	runner := &recordingRunner{args: make([]string, 0)}
	require.NoError(t, embedMetadata(runner, file, nil))
	assert.Empty(t, runner.args)
}

func TestIsMP4Container(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		path     string
		expected bool
	}{
		{name: "mp4", path: "video.mp4", expected: true},
		{name: "m4a", path: "audio.m4a", expected: true},
		{name: "webm", path: "video.webm", expected: false},
		{name: "no extension", path: "video", expected: false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, isMP4Container(tc.path))
		})
	}
}

func TestEmbedMetadataArgs(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name         string
		file         string
		tmpPath      string
		wantMovflags bool
	}{
		{
			name:         "mp4 container",
			file:         "video.mp4",
			tmpPath:      "video.123456.mp4",
			wantMovflags: true,
		},
		{
			name:         "m4a container",
			file:         "audio.m4a",
			tmpPath:      "audio.123456.m4a",
			wantMovflags: true,
		},
		{
			name:         "mkv container",
			file:         "video.mkv",
			tmpPath:      "video.123456.mkv",
			wantMovflags: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tags := [][2]string{{"Title", "Test Title"}}
			args := embedMetadataArgs(tc.file, tc.tmpPath, tags)

			assert.Equal(t, []string{
				"-y",
				"-i", tc.file,
				"-map",
			}, args[:4])

			assert.Equal(t, tc.tmpPath, args[len(args)-1])

			foundMovflags := false
			for i := range args {
				if args[i] == "-movflags" &&
					i+1 < len(args) &&
					args[i+1] == "use_metadata_tags" {
					foundMovflags = true
					break
				}
			}
			assert.Equal(t, tc.wantMovflags, foundMovflags)
		})
	}
}
