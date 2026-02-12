package capture

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/xymaxim/ypb/internal/actions"
	"github.com/xymaxim/ypb/internal/app"

	// "github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/internal/commands"
	"github.com/xymaxim/ypb/internal/input"
)

type Frame struct {
	commands.CommonFlags
	Moment       string `       help:"Moment to capture"   required:"" short:"m"`
	OutputFormat string `       help:"Output image format" required:""           name:"of" default:"png"`
	Stream       string `arg:"" help:"YouTube video ID"    required:""`
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

	var buf bytes.Buffer
	err = a.Playback.StreamSegment(
		a.Playback.Info().BestVideo().Itag,
		rewindMoment.Metadata.SequenceNumber,
		&buf,
	)
	if err != nil {
		return fmt.Errorf("downloading segment: %w", err)
	}

	outputPath := fmt.Sprintf(
		"%s_%s_%s.%s",
		commands.AdjustForFilename(a.Playback.Info().Title, 0),
		a.Playback.Info().ID,
		commands.FormatTime(rewindMoment.TargetTime),
		c.OutputFormat,
	)
	at := rewindMoment.TargetTime.Sub(rewindMoment.Metadata.Time()).Seconds()

	if err := ExtractFrame(&buf, at, outputPath); err != nil {
		return fmt.Errorf("extracting frame at %.3fs: %w", at, err)
	}

	fmt.Printf("Success! Saved to '%s'\n", outputPath)

	return nil
}

func ExtractFrame(r io.Reader, at float64, outputPath string) error {
	cmd := exec.Command("ffmpeg",
		"-ss", fmt.Sprintf("%.3f", at),
		"-i", "pipe:0",
		"-vframes", "1",
		"-y",
		outputPath,
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("getting stdin pipe: %w", err)
	}
	defer stdin.Close()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting ffmpeg: %w", err)
	}

	if _, err := io.Copy(stdin, r); err != nil {
		stdin.Close()
		return fmt.Errorf("streaming video to ffmpeg: %w", err)
	}

	if err := stdin.Close(); err != nil {
		return fmt.Errorf("failed to close stdin pipe: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg: %w", err)
	}

	return nil
}
