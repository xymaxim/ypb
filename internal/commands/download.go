package commands

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/xymaxim/ypb/internal/actions"
	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/input"
	"github.com/xymaxim/ypb/internal/urlutil"
	"github.com/xymaxim/ypb/playback"
)

type Download struct {
	CommonFlags
	Stream           string `arg:"" help:"YouTube video ID"                                             required:""`
	Interval         string `       help:"Time or segment interval"                                     required:"" short:"i"` //nolint:lll
	NoMetadata       bool   `       help:"Skip embedding metadata tags"`
	Cut              bool   `       help:"Cut downloaded file to input times (experimental)"                        short:"c"` //nolint:lll
	CutFFmpegOptions string `       help:"Extra FFmpeg options for cut (e.g. \"-c:v libx264 -crf 20\")"`
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

	fmt.Println(formatActualLine("start", interval.Start, c.Cut))
	fmt.Println(" ", formatActualLine("end", interval.End, c.Cut))
	if c.Cut {
		fmt.Println("--cut enabled, output will be trimmed to these times")
	}

	mux := http.NewServeMux()
	apppkg.RegisterSegmentRoute(mux, app)
	mux.HandleFunc(apppkg.MPDPath, apppkg.WithError(
		func(w http.ResponseWriter, r *http.Request) error {
			return serveMPD(w, app, interval)
		},
	),
	)

	app.Server.Handler = mux

	go func() {
		slog.Debug("starting server", "addr", app.Server.Addr)
		err = app.Server.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
	}()

	mpdURL := urlutil.FormatServerAddress(app.Server.Addr) +
		"/mpd/" +
		url.PathEscape(c.Interval)

	ytdlpPathFile, err := os.CreateTemp("", "ypb-path-*")
	if err != nil {
		return fmt.Errorf("creating output path file: %w", err)
	}
	ytdlpPath := ytdlpPathFile.Name()
	if err := ytdlpPathFile.Close(); err != nil {
		return fmt.Errorf("closing output path file: %w", err)
	}
	defer func() { _ = os.Remove(ytdlpPath) }()

	args := append(
		[]string{
			mpdURL,
			"--force-generic-extractor",
			"--output", buildOutputName(outputContext),
		},
		ytdlpOptions...,
	)
	args = append(args, "--print-to-file", "after_move:filepath", ytdlpPath)

	fmt.Println("(<<) Downloading and merging media...")
	if err := app.YtdlpRunner.Run(context.Background(), args...); err != nil {
		return fmt.Errorf("downloading failed: %w", err)
	}

	pathBytes, err := os.ReadFile(ytdlpPath)
	if err != nil {
		return fmt.Errorf("reading output file path: %w", err)
	}
	outputPath := strings.TrimSpace(string(pathBytes))
	if outputPath == "" {
		return errors.New("output file path is empty")
	}

	if c.Cut {
		fmt.Println("(<<) Cutting downloaded file...")

		cutStart := outputContext.InputStartTime.
			Sub(outputContext.ActualStartTime).Seconds()
		if cutStart < 0 {
			slog.Warn("input start time was in a gap, clamped to start")
			cutStart = 0
		}

		actualDuration := outputContext.ActualDuration.Seconds()
		cutEnd := outputContext.InputEndTime.
			Sub(outputContext.ActualStartTime).Seconds()
		if cutEnd > actualDuration {
			slog.Warn("input end time was in a gap, clamped to end")
			cutEnd = actualDuration
		}

		opts := CutOptions{
			StartSeconds:    cutStart,
			EndSeconds:      cutEnd,
			ExtraFFmpegArgs: c.CutFFmpegOptions,
		}
		if err := cut(context.Background(), outputPath, outputPath, opts); err != nil {
			return fmt.Errorf("cutting downloaded file: %w", err)
		}
	}

	if c.NoMetadata {
		slog.Info("skip metadata tag embedding")
		return nil
	}

	slog.Info("embedding metadata tags")
	metadata_ctx := &metadataContext{
		LocateOutputContext: outputContext,
		ChannelTitle:        app.Playback.Info().ChannelTitle,
	}
	if err := embedMetadata(
		app.FFmpegRunner,
		outputPath,
		metadataTags(metadata_ctx),
	); err != nil {
		return fmt.Errorf("embedding metadata: %w", err)
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

func formatActualLine(side string, moment *playback.RewindMoment, cutEnabled bool) string {
	if cutEnabled {
		return fmt.Sprintf(
			"Actual %s: %s, sq=%d",
			side,
			moment.TargetTime.Format(time.RFC1123Z),
			moment.Metadata.SequenceNumber,
		)
	}

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
