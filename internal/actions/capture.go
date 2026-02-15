package actions

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"

	"github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/internal/playback"
)

// CaptureFrame extracts a frame corresponding to a moment.
func CaptureFrame(
	pb playback.Playbacker,
	moment *playback.RewindMoment,
	outputPath string,
	runner exec.Runner,
) error {
	var buf bytes.Buffer

	err := pb.StreamSegment(
		pb.Info().BestVideo().Itag,
		moment.Metadata.SequenceNumber,
		&buf,
	)
	if err != nil {
		return fmt.Errorf(
			"downloading segment, sq=%d: %w",
			moment.Metadata.SequenceNumber,
			err,
		)
	}

	err = extractFrame(moment, outputPath, &buf, runner)
	if err != nil {
		return fmt.Errorf("extracting frame: %w", err)
	}

	return nil
}

func extractFrame(
	moment *playback.RewindMoment,
	outputPath string,
	buf *bytes.Buffer,
	runner exec.Runner,
) error {
	at := moment.TargetTime.Sub(moment.Metadata.Time()).Seconds()
	slog.Debug("extracting frame", "sq", moment.Metadata.SequenceNumber, "t", at)

	result, err := runner.RunWith(
		[]exec.Option{
			exec.WithQuiet(),
			exec.WithStdin(bytes.NewReader(buf.Bytes())),
		},
		"-hide_banner", "-y",
		"-i", "pipe:0",
		"-ss", fmt.Sprintf("%.3f", at),
		"-vframes", "1",
		outputPath,
	)
	if err != nil {
		return fmt.Errorf(
			"getting frame at %.3f: %w (stderr: %s)",
			at,
			err,
			result.Stderr,
		)
	}

	// Frame not found, extract last frame as fallback
	if _, statErr := os.Stat(outputPath); os.IsNotExist(statErr) {
		slog.Debug("frame not found, extract last frame")
		if err := extractLastFrame(outputPath, buf, runner); err != nil {
			return fmt.Errorf("extracting last frame: %w", err)
		}
	}

	return nil
}

func extractLastFrame(outputPath string, buf *bytes.Buffer, runner exec.Runner) error {
	tempFile, err := os.CreateTemp("", "ypb-*.mp4")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	// Step 1. Remux a segment to a temp file
	remuxResult, remuxErr := runner.RunWith(
		[]exec.Option{
			exec.WithQuiet(),
			exec.WithStdin(bytes.NewReader(buf.Bytes())),
		},
		"-hide_banner", "-y",
		"-i", "pipe:0",
		"-c", "copy",
		"-y",
		tempFile.Name(),
	)
	if remuxErr != nil {
		return fmt.Errorf("remuxing segment: %w (stderr: %s)", remuxErr, remuxResult.Stderr)
	}

	// Step 2. Extract the last frame from a temp file
	extractResult, extractErr := runner.RunWith(
		[]exec.Option{exec.WithQuiet()},
		"-hide_banner", "-y",
		"-sseof", "-1",
		"-i", tempFile.Name(),
		"-update", "true",
		outputPath,
	)
	if extractErr != nil {
		return fmt.Errorf(
			"getting frame: %w (stderr: %s)",
			extractErr,
			extractResult.Stderr,
		)
	}

	return nil
}
