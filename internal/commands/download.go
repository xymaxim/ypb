package commands

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/xymaxim/ypb/internal/actions"
	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/input"
	"github.com/xymaxim/ypb/internal/urlutil"
	"github.com/xymaxim/ypb/playback"
)

type Download struct {
	CommonFlags
	Stream   string `arg:"" help:"YouTube video ID"         required:""`
	Interval string `       help:"Time or segment interval" required:"" short:"i"`
	LatencyFlag
	YtdlpOptionsFlag
}

func (c *Download) Validate() error {
	return ValidateLatency(c.Latency)
}

func (c *Download) Run() error {
	startupTime := time.Now().UTC()
	pinnedTime, err := ResolvePinnedTime(c.Now, startupTime)
	if err != nil {
		return err
	}

	slog.Info(
		"reference times",
		slog.Time("startup", startupTime),
		slog.Time("pinned", pinnedTime),
	)

	if err := checkYtdlp(); err != nil {
		return err
	}

	start, end, err := input.ParseInterval(c.Interval, &pinnedTime)
	if err != nil {
		return fmt.Errorf("parsing input interval: %w", err)
	}
	if err := input.ValidateMoments(start, end, startupTime); err != nil {
		return fmt.Errorf("bad input interval: %w", err)
	}
	for _, mv := range []input.MomentValue{start, end} {
		if err := input.ValidateMoment(
			mv,
			ToLatencyDuration(c.Latency),
			startupTime,
		); err != nil {
			return fmt.Errorf("bad input interval: %w", err)
		}
	}

	ytdlpOptions := NormalizeYtdlpOptions(c.YtdlpOptions)
	app, err := apppkg.InitApp(c.Stream, c.Port, ytdlpOptions)
	if err != nil {
		return err
	}

	fmt.Printf("(<<) Stream '%s' is alive!\n", app.Playback.Info().Title)

	fmt.Println("(<<) Locating start and end moments...")
	locateContext, err := actions.NewLocateContext(app.Playback, nil, &pinnedTime)
	if err != nil {
		return fmt.Errorf("building locate context: %w", err)
	}
	locateContext.Latency = ToLatencyDuration(c.Latency)

	interval, outputContext, err := actions.LocateInterval(
		app.Playback,
		start,
		end,
		locateContext,
	)
	if err != nil {
		return fmt.Errorf("locating interval: %w", err)
	}

	fmt.Println(formatActualLine("start", interval.Start))
	fmt.Println(" ", formatActualLine("end", interval.End))

	mux := http.NewServeMux()
	apppkg.RegisterSegmentRoute(mux, app)
	mux.HandleFunc(apppkg.MPDPath, apppkg.WithError(
		func(w http.ResponseWriter, r *http.Request) error {
			return serveMPD(w, app, interval)
		}),
	)

	app.Server.Handler = mux

	go func() {
		slog.Debug("starting server", "addr", app.Server.Addr)
		err = app.Server.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
	}()

	mpdURL, err := url.JoinPath(urlutil.FormatServerAddress(app.Server.Addr), apppkg.MPDPath)
	if err != nil {
		return fmt.Errorf("building URL: %w", err)
	}

	args := append(
		[]string{
			mpdURL,
			"--force-generic-extractor",
			"--output", buildOutputName(outputContext),
		},
		ytdlpOptions...,
	)

	fmt.Println("(<<) Downloading and merging media...")
	if err := app.YtdlpRunner.Run(context.Background(), args...); err != nil {
		return fmt.Errorf("downloading failed: %w", err)
	}

	return nil
}

func serveMPD(w http.ResponseWriter, app *apppkg.App, interval *playback.RewindInterval) error {
	out, err := actions.ComposeStatic(
		app.Playback,
		interval,
		urlutil.FormatServerAddress(app.Server.Addr),
		app.FFprobeRunner,
	)
	if err != nil {
		return fmt.Errorf("composing manifest: %w", err)
	}

	w.Header().Set("Content-Type", "application/dash+xml")
	if _, err := w.Write(out); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	return nil
}

func formatActualLine(side string, moment *playback.RewindMoment) string {
	diffPart := ""

	diff := moment.TimeDifference()
	if diff.Abs() >= time.Second {
		diffPart = fmt.Sprintf(" (%s)", FormatDifference(diff, true))
	}

	return fmt.Sprintf(
		"Actual %s: %s%s, sq=%d",
		side,
		moment.ActualTime.Format(time.RFC1123Z),
		diffPart,
		moment.Metadata.SequenceNumber,
	)
}

func buildOutputName(ctx *actions.LocateOutputContext) string {
	return actions.BuildOutputStem(ctx) + ".%(ext)s"
}
