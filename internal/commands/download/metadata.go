package download

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xymaxim/ypb/internal/actions"
	"github.com/xymaxim/ypb/internal/exec"
)

func formatISO8601(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z07:00")
}

type metadataContext struct {
	actions.LocateOutputContext
	ChannelTitle string
}

func metadataTags(ctx *metadataContext) [][2]string {
	return [][2]string{
		{"Title", ctx.Title},
		{"Author", ctx.ChannelTitle},
		{"Comment", "https://www.youtube.com/watch?v=" + ctx.ID},
		{"ActualStartTime", formatISO8601(ctx.ActualStartTime)},
		{"InputStartTime", formatISO8601(ctx.InputStartTime)},
		{"ActualEndTime", formatISO8601(ctx.ActualEndTime)},
		{"InputEndTime", formatISO8601(ctx.InputEndTime)},
		{"StartSegment", strconv.Itoa(ctx.StartSequenceNumber)},
		{"EndSegment", strconv.Itoa(ctx.EndSequenceNumber)},
	}
}

// embedMetadata re-muxes the given output file, writing the provided metadata
// tags with FFmpeg. The file is replaced in place.
func embedMetadata(
	runner exec.Runner,
	file string,
	tags [][2]string,
) error {
	ext := filepath.Ext(file)
	if ext == "" {
		slog.Warn(
			"output file has no extension, skipping metadata embedding",
			"file", file,
		)
		return nil
	}

	stem := strings.TrimSuffix(filepath.Base(file), ext)
	tmp, err := os.CreateTemp(filepath.Dir(file), stem+".*"+ext)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpPath) }()

	args := embedMetadataArgs(file, tmpPath, tags)

	result, err := runner.RunWith(
		context.Background(),
		[]exec.Option{exec.WithQuiet()},
		args...,
	)
	if err != nil {
		if result != nil && len(result.Stderr) > 0 {
			slog.Error("ffmpeg failed", "stderr", string(result.Stderr))
		}
		return fmt.Errorf("running ffmpeg: %w", err)
	}

	if err := os.Rename(tmpPath, file); err != nil {
		return fmt.Errorf("replacing output file: %w", err)
	}

	return nil
}

func embedMetadataArgs(file, tmpPath string, tags [][2]string) []string {
	args := []string{"-y", "-i", file, "-map", "0", "-c", "copy"}

	if isMP4Container(tmpPath) {
		args = append(args, "-movflags", "use_metadata_tags")
	}

	for _, tag := range tags {
		args = append(args, "-metadata", tag[0]+"="+tag[1])
	}

	return append(args, tmpPath)
}

func isMP4Container(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4", ".m4a":
		return true
	default:
		return false
	}
}
