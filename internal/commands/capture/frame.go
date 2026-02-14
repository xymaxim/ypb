package capture

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/xymaxim/ypb/internal/actions"
	"github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/commands"
	"github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/internal/input"
	"github.com/xymaxim/ypb/internal/playback"
)

type Frame struct {
	commands.CommonFlags
	Moment       string `help:"Moment to capture"   required:"" short:"m"`
	OutputFormat string `help:"Output image format" required:""           name:"of" default:"png"`
	Stream       string `help:"YouTube video ID"    required:""                                   arg:""`
}

func (c *Frame) Run() error {
	a := app.NewApp()

	if err := commands.CollectVideoInfo(c.Stream, a, c.Port); err != nil {
		return err
	}

	fmt.Println("(<<) Locating and capturing the moment...")
	locateContext, err := actions.NewLocateContext(a.Playback, nil)
	if err != nil {
		return fmt.Errorf("building locate context: %w", err)
	}

	momentValue, err := input.ParseIntervalPart(c.Moment)
	if err != nil {
		return fmt.Errorf("parsing input moment: %w", err)
	}

	rewindMoment, err := actions.LocateMoment(a.Playback, momentValue, locateContext)
	if err != nil {
		return fmt.Errorf("locating moment: %w", err)
	}

	if rewindMoment.InGap {
		fmt.Printf("Moment falls into a stream gap, exit")
		return nil
	}

	fmt.Printf(
		"Frame time: %s, sq=%d\n",
		rewindMoment.TargetTime.Format(time.RFC1123Z),
		rewindMoment.Metadata.SequenceNumber,
	)

	outputPath := fmt.Sprintf(
		"%s_%s_%s.%s",
		commands.AdjustForFilename(a.Playback.Info().Title, 0),
		a.Playback.Info().ID,
		commands.FormatTime(rewindMoment.TargetTime),
		c.OutputFormat,
	)

	err = DownloadAndExtractFrame(a, rewindMoment, outputPath)
	if err != nil {
		return fmt.Errorf("capturing frame: %w", err)
	}

	fmt.Printf("Success! Saved to '%s'\n", outputPath)

	return nil
}

func DownloadAndExtractFrame(a *app.App, moment *playback.RewindMoment, outputPath string) error {
	var buf bytes.Buffer

	err := a.Playback.StreamSegment(
		a.Playback.Info().BestVideo().Itag,
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

	err = ExtractFrame(a.FFmpegRunner, &buf, moment, outputPath)
	if err != nil {
		return fmt.Errorf("extracting frame: %w", err)
	}

	return nil
}

func ExtractFrame(
	runner exec.Runner,
	buf *bytes.Buffer,
	moment *playback.RewindMoment,
	outputPath string,
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
		if err := extractLastFrame(runner, buf, outputPath); err != nil {
			return fmt.Errorf("extracting last frame: %w", err)
		}
	}

	return nil
}

func extractLastFrame(runner exec.Runner, buf *bytes.Buffer, outputPath string) error {
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
