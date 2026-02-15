package capture

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"

	"github.com/xymaxim/ypb/internal/actions"
	"github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/commands"
	"github.com/xymaxim/ypb/internal/input"
	"github.com/xymaxim/ypb/internal/playback"
)

type Timelapse struct {
	commands.CommonFlags
	CommonCaptureFlags
	Every    string `help:"Capture frame every duration" placeholder:"DURATION" required:"" short:"e"`
	Stream   string `help:"YouTube video ID"                                    required:""           arg:""`
	Interval string `help:"Time or segment interval"                            required:"" short:"i"`
}

type TimelapseConfig struct {
	StartMoment   input.MomentValue
	EndMoment     input.MomentValue
	CaptureEvery  time.Duration
	OutputFormat  string
	OutputPattern string
}

func (c *Timelapse) Run() error {
	a := app.NewApp()

	// Parse and validate inputs
	config, err := c.parseAndValidateInputs()
	if err != nil {
		return err
	}

	// Collect video information and initialize the app
	if err := commands.CollectVideoInfo(c.Stream, a, c.Port); err != nil {
		return err
	}

	// Locate interval moments
	interval, locateContext, err := c.locateInterval(a.Playback, config)
	if err != nil {
		return err
	}

	// Calculate frame times to capture
	captureTimes := c.calculateCaptureTimes(
		interval.Start.TargetTime,
		interval.End.TargetTime,
		config.CaptureEvery,
	)

	printCapturePlan(captureTimes, config.CaptureEvery)

	// Build output pattern and create output directories
	config.OutputPattern = c.buildOutputPattern(a, captureTimes, config)
	outputDirectory := filepath.Dir(config.OutputPattern)
	if err := os.Mkdir(outputDirectory, os.ModePerm); err != nil {
		return fmt.Errorf("creating output directories: %w", err)
	}

	// Capture all frames
	err = c.captureFrames(a, captureTimes, locateContext, config)
	if err != nil {
		return fmt.Errorf("capturing frames: %w", err)
	}

	return nil
}

func (c *Timelapse) parseAndValidateInputs() (*TimelapseConfig, error) {
	start, end, err := input.ParseInterval(c.Interval)
	if err != nil {
		return nil, fmt.Errorf("parsing input interval: %w", err)
	}

	if err := input.ValidateMoments(start, end); err != nil {
		return nil, fmt.Errorf("bad input interval: %w", err)
	}

	duration, err := input.ParseIntervalPart(c.Every)
	if err != nil {
		return nil, fmt.Errorf("parsing input every duration: %w", err)
	}

	captureEvery, ok := duration.(time.Duration)
	if !ok {
		return nil, errors.New("every duration must be a time.Duration")
	}

	return &TimelapseConfig{
		StartMoment:  start,
		EndMoment:    end,
		CaptureEvery: captureEvery,
		OutputFormat: c.OutputFormat,
	}, nil
}

func (c *Timelapse) locateInterval(
	playback playback.Playbacker,
	config *TimelapseConfig,
) (*playback.RewindInterval, *actions.LocateContext, error) {
	fmt.Print("(<<) Locating start and end moments... ")

	locateContext, err := actions.NewLocateContext(playback, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building locate context: %w", err)
	}

	interval, _, err := actions.LocateInterval(
		playback,
		config.StartMoment,
		config.EndMoment,
		locateContext,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("locating interval: %w", err)
	}

	fmt.Println("done.")

	return interval, locateContext, nil
}

func (c *Timelapse) calculateCaptureTimes(start, end time.Time, every time.Duration) []time.Time {
	var times []time.Time
	for t := start; !t.After(end); t = t.Add(every) {
		times = append(times, t)
	}
	return times
}

func (c *Timelapse) buildOutputPattern(
	a *app.App,
	captureTimes []time.Time,
	config *TimelapseConfig,
) string {
	basename := fmt.Sprintf(
		"%s_%s_%s_e%s",
		commands.AdjustForFilename(a.Playback.Info().Title, 0),
		a.Playback.Info().ID,
		commands.FormatTime(captureTimes[0]),
		commands.FormatDuration(config.CaptureEvery),
	)
	outputDirectory := basename
	outputFilename := fmt.Sprintf("%s_%%04d.%s", basename, config.OutputFormat)

	return filepath.Join(outputDirectory, outputFilename)
}

func (c *Timelapse) captureFrames(
	a *app.App,
	times []time.Time,
	locateContext *actions.LocateContext,
	config *TimelapseConfig,
) error {
	fmt.Printf("(<<) Capturing frames to '%s'...\n", filepath.Dir(config.OutputPattern))

	bar := progressbar.Default(int64(len(times)))

	var skippedCount int
	reference := locateContext.Now
	for frameIndex, t := range times {
		rewindMoment, err := a.Playback.LocateMoment(t, *reference, false)
		if err != nil {
			return fmt.Errorf("locating moment: %w", err)
		}

		if rewindMoment.InGap {
			slog.Warn(
				"frame falls into a gap, skipping",
				"frame",
				frameIndex,
				"time",
				t,
			)
			skippedCount++
			continue
		}

		outputPath := fmt.Sprintf(config.OutputPattern, frameIndex)
		err = actions.CaptureFrame(a.Playback, rewindMoment, outputPath, a.FFmpegRunner)
		if err != nil {
			return fmt.Errorf("capturing frame %d at %s: %w", frameIndex, t, err)
		}

		reference = rewindMoment.Metadata

		bar.Add(1)
	}

	total := len(times)
	fmt.Printf(
		"Success! %d of %d frames captured (%d skipped)\n",
		total-skippedCount,
		total,
		skippedCount,
	)

	return nil
}

func printCapturePlan(times []time.Time, duration time.Duration) {
	total := len(times)

	fmt.Printf(
		"Frames will be captured every %s (%d total):\n",
		commands.FormatDuration(duration),
		total,
	)

	formatTime := func(t time.Time) string {
		return t.Format(time.RFC1123Z)
	}
	if total <= 3 {
		for i := range total {
			fmt.Printf("  Frame %d: %s\n", i+1, formatTime(times[i]))
		}
	} else {
		pad := strings.Repeat(" ", len(strconv.Itoa(total))-1)
		fmt.Printf("  %sFrame 1: %s\n", pad, formatTime(times[0]))
		fmt.Printf("  %sFrame 2: %s\n", pad, formatTime(times[1]))
		fmt.Printf("  %s                       ...\n", pad)
		fmt.Printf("  Frame %d: %s\n", total, formatTime(times[total-1]))
	}
}
