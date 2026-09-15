package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/shlex"

	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/exec"
)

type CutOptions struct {
	StartSeconds    float64
	EndSeconds      float64
	ExtraFFmpegArgs string
}

// cut writes inputPath, trimmed to opts' time range, to outputPath.
// Output is atomically renamed into place, so inputPath may equal outputPath.
func cut(ctx context.Context, inputPath, outputPath string, opts CutOptions) error {
	slog.Info("cutting file",
		"start", opts.StartSeconds,
		"end", opts.EndSeconds,
	)

	args, err := cutArgs(inputPath, opts)
	if err != nil {
		return err
	}

	slog.Debug("running ffmpeg cut", "args", args)

	ext := filepath.Ext(outputPath)
	stem := strings.TrimSuffix(filepath.Base(outputPath), ext)
	tmp, err := os.CreateTemp(filepath.Dir(outputPath), stem+".*"+ext)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpPath) }()

	args = append(args, tmpPath)

	runner := exec.NewCommandRunner(apppkg.FFmpegBinaryPath)
	if err := runner.Run(ctx, args...); err != nil {
		return fmt.Errorf("ffmpeg: cutting input file: %w", err)
	}

	if err := os.Rename(tmpPath, outputPath); err != nil {
		return fmt.Errorf("replacing output file: %w", err)
	}

	return nil
}

// cutArgs builds ffmpeg arguments for cutting inputPath per opts.
func cutArgs(inputPath string, opts CutOptions) ([]string, error) {
	if opts.StartSeconds < 0 {
		return nil, fmt.Errorf(
			"invalid cut range: start (%.3f) must be non-negative",
			opts.StartSeconds,
		)
	}

	duration := opts.EndSeconds - opts.StartSeconds
	if duration <= 0 {
		return nil, fmt.Errorf(
			"invalid cut range: end (%.3f) must be after start (%.3f)",
			opts.EndSeconds,
			opts.StartSeconds,
		)
	}

	args := []string{
		"-hide_banner",
		"-y",
		"-ss", strconv.FormatFloat(opts.StartSeconds, 'f', 3, 64),
		"-i", inputPath,
		"-t", strconv.FormatFloat(duration, 'f', 3, 64),
		"-c:a", "copy",
		"-avoid_negative_ts", "make_zero",
		"-shortest",
		"-fflags", "+genpts",
	}

	if opts.ExtraFFmpegArgs != "" {
		extra, err := shlex.Split(opts.ExtraFFmpegArgs)
		if err != nil {
			return nil, fmt.Errorf("parsing extra ffmpeg options: %w", err)
		}
		args = append(args, extra...)
	}

	return args, nil
}
