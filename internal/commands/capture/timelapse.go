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

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/xymaxim/ypb/internal/actions"
	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/commands"
	"github.com/xymaxim/ypb/internal/input"
	"github.com/xymaxim/ypb/playback"
)

type Timelapse struct {
	commands.CommonFlags
	CommonCaptureFlags
	Every    string `help:"Capture frame every duration" placeholder:"DURATION" required:"" short:"e"`
	Stream   string `help:"YouTube video ID"                                    required:""           arg:""`
	Interval string `help:"Time or segment interval"                            required:"" short:"i"`
	commands.LatencyFlag
	commands.YtdlpOptionsFlag
}

type TimelapseConfig struct {
	StartMoment   input.MomentValue
	EndMoment     input.MomentValue
	CaptureEvery  time.Duration
	OutputFormat  string
	OutputPattern string
}

func (c *Timelapse) Validate() error {
	if err := commands.ValidateLatency(c.Latency); err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}

func (c *Timelapse) Run() error {
	startupTime := time.Now().UTC()
	pinnedTime, err := commands.ResolvePinnedTime(c.Now, startupTime)
	if err != nil {
		return err
	}

	slog.Info(
		"reference times",
		slog.Time("startup", startupTime),
		slog.Time("pinned", pinnedTime),
	)

	config, err := c.parseAndValidateInputs(&pinnedTime, startupTime)
	if err != nil {
		return err
	}

	ytdlpOptions := commands.NormalizeYtdlpOptions(c.YtdlpOptions)
	app, err := apppkg.InitApp(c.Stream, c.Port, ytdlpOptions)
	if err != nil {
		return err
	}

	fmt.Printf("(<<) Stream '%s' is alive!\n", app.Playback.Info().Title)

	interval, locateContext, err := c.locateInterval(app.Playback, pinnedTime, config)
	if err != nil {
		return err
	}

	captureTimes := c.calculateCaptureTimes(
		interval.Start.TargetTime,
		interval.End.TargetTime,
		config.CaptureEvery,
	)

	printCapturePlan(captureTimes, config.CaptureEvery)

	config.OutputPattern = c.buildOutputPattern(app, captureTimes, config)
	outputDirectory := filepath.Dir(config.OutputPattern)
	if err := os.Mkdir(outputDirectory, os.ModePerm); err != nil {
		return fmt.Errorf("creating output directories: %w", err)
	}

	err = c.captureFrames(app, captureTimes, locateContext, config)
	if err != nil {
		return fmt.Errorf("capturing frames: %w", err)
	}

	return nil
}

func (c *Timelapse) parseAndValidateInputs(
	refTime *time.Time,
	now time.Time,
) (*TimelapseConfig, error) {
	start, end, err := input.ParseInterval(c.Interval, refTime)
	if err != nil {
		return nil, fmt.Errorf("parsing input interval: %w", err)
	}

	if err := input.ValidateMoments(start, end, now); err != nil {
		return nil, fmt.Errorf("bad input interval: %w", err)
	}
	for _, mv := range []input.MomentValue{start, end} {
		if err := input.ValidateMoment(
			mv,
			commands.ToLatencyDuration(c.Latency),
			now,
		); err != nil {
			return nil, fmt.Errorf("bad input interval: %w", err)
		}
	}

	duration, err := input.ParseIntervalPart(c.Every, nil)
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
	pb playback.Playbacker,
	pinnedTime time.Time,
	config *TimelapseConfig,
) (*playback.RewindInterval, *actions.LocateContext, error) {
	fmt.Print("(<<) Locating start and end moments... ")

	locateContext, err := actions.NewLocateContext(pb, nil, &pinnedTime)
	if err != nil {
		return nil, nil, fmt.Errorf("building locate context: %w", err)
	}
	locateContext.Latency = commands.ToLatencyDuration(c.Latency)

	interval, _, err := actions.LocateInterval(
		pb,
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
	app *apppkg.App,
	captureTimes []time.Time,
	config *TimelapseConfig,
) string {
	basename := fmt.Sprintf(
		"%s_%s_%s_e%s",
		actions.AdjustForFilename(app.Playback.Info().Title, 0),
		app.Playback.Info().ID,
		actions.FormatTime(captureTimes[0]),
		actions.FormatDuration(config.CaptureEvery),
	)
	outputDirectory := basename
	outputFilename := fmt.Sprintf("%s_%%04d.%s", basename, config.OutputFormat)

	return filepath.Join(outputDirectory, outputFilename)
}

func (c *Timelapse) captureFrames(
	app *apppkg.App,
	times []time.Time,
	locateContext *actions.LocateContext,
	config *TimelapseConfig,
) error {
	fmt.Printf("(<<) Capturing frames to '%s'...\n", filepath.Dir(config.OutputPattern))

	start := time.Now()
	totalFrames := len(times)

	p := message.NewPrinter(language.English)
	onFrame := func(index int, _ bool) {
		done := index + 1
		elapsed := time.Since(start)
		framesPerMin := float64(done) / elapsed.Minutes()
		eta := time.Duration(
			float64(totalFrames-done) / framesPerMin * float64(time.Minute),
		)
		fmt.Printf("\r%3.0f%% at %4.0f fr/min ETA %-10s (frame %s/%s)",
			float64(done)/float64(totalFrames)*100,
			framesPerMin,
			eta.Round(time.Second),
			p.Sprintf("%d", done),
			p.Sprintf("%d", totalFrames),
		)
		if done == totalFrames {
			fmt.Println()
		}
	}

	captured, skipped, err := actions.CaptureFrames(
		app.Playback,
		times,
		locateContext,
		config.OutputPattern,
		app.FFmpegRunner,
		onFrame,
	)
	if err != nil {
		return fmt.Errorf("capturing frames: %w", err)
	}

	fmt.Printf(
		"Success! %d of %d frames captured (%d skipped)\n",
		captured, len(times), skipped,
	)

	return nil
}

func printCapturePlan(times []time.Time, duration time.Duration) {
	total := len(times)

	frameWord := "frames"
	if total == 1 {
		frameWord = "frame"
	}
	fmt.Printf(
		"Will capture %d %s at %s intervals:\n",
		total,
		frameWord,
		actions.FormatDuration(duration),
	)

	formatTime := func(t time.Time) string {
		return t.Format(time.RFC1123Z)
	}
	if total <= 3 {
		for i := range total {
			fmt.Printf("  Frame %d: %s\n", i, formatTime(times[i]))
		}
	} else {
		pad := strings.Repeat(" ", len(strconv.Itoa(total-1))-1)
		fmt.Printf("  %sFrame 0: %s\n", pad, formatTime(times[0]))
		fmt.Printf("  %sFrame 1: %s\n", pad, formatTime(times[1]))
		fmt.Printf("  %s                       ...\n", pad)
		fmt.Printf("  Frame %d: %s\n", total-1, formatTime(times[total-1]))
	}
}
