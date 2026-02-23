package capture

import (
	"fmt"
	"time"

	"github.com/xymaxim/ypb/internal/actions"
	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/commands"
	"github.com/xymaxim/ypb/internal/input"
	"github.com/xymaxim/ypb/internal/playback"
)

type Frame struct {
	commands.CommonFlags
	CommonCaptureFlags
	Moment string `help:"Moment to capture" required:"" short:"m"`
	Stream string `help:"YouTube video ID"  required:""           arg:""`
}

type FrameConfig struct {
	MomentValue  any
	OutputFormat string
	OutputPath   string
}

func (c *Frame) Run() error {
	app := apppkg.NewApp()

	// Parse and validate inputs
	config, err := c.parseAndValidateInputs()
	if err != nil {
		return err
	}

	// Collect video information and initialize the app
	if err := commands.CollectVideoInfo(c.Stream, app, c.Port); err != nil {
		return err
	}

	// Locate the moment
	rewindMoment, _, err := c.locateMoment(app.Playback, config)
	if err != nil {
		return err
	}

	if rewindMoment.InGap {
		fmt.Println("Moment falls into a stream gap, exit")
		return nil
	}

	fmt.Printf(
		"Frame to be captured: %s, sq=%d\n",
		rewindMoment.TargetTime.Format(time.RFC1123Z),
		rewindMoment.Metadata.SequenceNumber,
	)

	// Capture the frame
	config.OutputPath = fmt.Sprintf(
		"%s_%s_%s.%s",
		commands.AdjustForFilename(app.Playback.Info().Title, 0),
		app.Playback.Info().ID,
		commands.FormatTime(rewindMoment.TargetTime),
		c.OutputFormat,
	)
	err = actions.CaptureFrame(app.Playback, rewindMoment, config.OutputPath, app.FFmpegRunner)
	if err != nil {
		return fmt.Errorf("capturing frame: %w", err)
	}

	fmt.Printf("Success! Saved to '%s'\n", config.OutputPath)

	return nil
}

func (c *Frame) parseAndValidateInputs() (*FrameConfig, error) {
	momentValue, err := input.ParseIntervalPart(c.Moment)
	if err != nil {
		return nil, fmt.Errorf("parsing input moment: %w", err)
	}

	return &FrameConfig{
		MomentValue:  momentValue,
		OutputFormat: c.OutputFormat,
	}, nil
}

func (c *Frame) locateMoment(
	pb playback.Playbacker,
	config *FrameConfig,
) (*playback.RewindMoment, *actions.LocateContext, error) {
	fmt.Println("(<<) Locating and capturing the moment...")

	locateContext, err := actions.NewLocateContext(pb, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building locate context: %w", err)
	}

	moment, err := actions.LocateMoment(pb, config.MomentValue, locateContext)
	if err != nil {
		return nil, nil, fmt.Errorf("locating moment: %w", err)
	}

	return moment, locateContext, nil
}
