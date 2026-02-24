package actions

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"time"

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

	err = extractFrame(moment, outputPath, buf.Bytes(), runner)
	if err != nil {
		return fmt.Errorf("extracting frame: %w", err)
	}

	return nil
}

func CaptureFrames(
	pb playback.Playbacker,
	times []time.Time,
	locateContext *LocateContext,
	outputPattern string,
	runner exec.Runner,
	onFrame func(index int, skipped bool),
) (captured, skipped int, err error) {
	var previousSq playback.SequenceNumber
	var previousSegment []byte

	reference := locateContext.Head

	for frameIndex, t := range times {
		rewindMoment, err := pb.LocateMoment(t, reference, false)
		if err != nil {
			return captured, skipped, fmt.Errorf(
				"frame %d at %s: locating moment: %w",
				frameIndex,
				t,
				err,
			)
		}

		if rewindMoment.InGap {
			skipped++
			if onFrame != nil {
				onFrame(frameIndex, true)
			}
			continue
		}

		sq := rewindMoment.Metadata.SequenceNumber

		// Download the segment only if it differs from the previous one
		if previousSegment == nil || previousSq != sq {
			var buf bytes.Buffer
			if err := pb.StreamSegment(
				pb.Info().BestVideo().Itag,
				sq,
				&buf,
			); err != nil {
				return captured, skipped, fmt.Errorf(
					"frame %d at %s: downloading segment, sq=%d: %w",
					frameIndex,
					t,
					sq,
					err,
				)
			}
			previousSq = sq
			previousSegment = buf.Bytes()
		}

		outputPath := fmt.Sprintf(outputPattern, frameIndex)
		if err := extractFrame(
			rewindMoment,
			outputPath,
			previousSegment,
			runner,
		); err != nil {
			return captured, skipped, fmt.Errorf(
				"frame %d at %s: extracting frame: %w",
				frameIndex,
				t,
				err,
			)
		}

		captured++
		reference = rewindMoment.Metadata

		if onFrame != nil {
			onFrame(frameIndex, false)
		}
	}

	return captured, skipped, nil
}

func extractFrame(
	moment *playback.RewindMoment,
	outputPath string,
	segment []byte,
	runner exec.Runner,
) error {
	at := moment.TargetTime.Sub(moment.Metadata.Time()).Seconds()
	slog.Debug("extracting frame", "sq", moment.Metadata.SequenceNumber, "t", at)

	result, err := runner.RunWith(
		[]exec.Option{
			exec.WithQuiet(),
			exec.WithStdin(bytes.NewReader(segment)),
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
		if err := extractLastFrame(outputPath, segment, runner); err != nil {
			return fmt.Errorf("extracting last frame: %w", err)
		}
	}

	return nil
}

func extractLastFrame(outputPath string, segment []byte, runner exec.Runner) error {
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
			exec.WithStdin(bytes.NewReader(segment)),
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
