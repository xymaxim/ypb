package commands

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gosimple/slug"

	"github.com/xymaxim/ypb/internal/actions"
	"github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/input"
	"github.com/xymaxim/ypb/internal/playback"
	"github.com/xymaxim/ypb/internal/urlutil"
)

type Download struct {
	Stream       string   `arg:"" help:"YouTube video ID"                         required:""`
	Interval     string   `       help:"Time or segment interval"                 required:"" short:"i"`
	Port         int      `       help:"Port to start playback on"                            short:"p" default:"8080"`
	YtdlpPath    string   `       help:"Path to yt-dlp binary"                                          default:"yt-dlp" type:"path"`
	YtdlpOptions []string `arg:"" help:"Options to pass to yt-dlp (use after --)"                                                    optional:"" passthrough:""` //nolint:lll
}

func (c *Download) Run() error {
	if err := checkYtdlp(c.YtdlpPath); err != nil {
		return err
	}

	a := app.NewApp(c.YtdlpPath)

	start, end, err := input.ParseInterval(c.Interval)
	if err != nil {
		return fmt.Errorf("parsing input interval: %w", err)
	}
	if err := input.ValidateMoments(start, end); err != nil {
		return fmt.Errorf("bad input interval: %w", err)
	}

	if err := collectVideoInfo(c.Stream, a, c.Port); err != nil {
		return err
	}

	fmt.Println("(<<) Locating start and end moments...")
	locateContext, err := actions.NewLocateContext(a.Playback, nil)
	if err != nil {
		return fmt.Errorf("building locate context: %w", err)
	}

	interval, outputContext, err := actions.LocateInterval(
		a.Playback,
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
	mux.HandleFunc("/mpd", app.WithError(
		func(w http.ResponseWriter, r *http.Request) error {
			out, err := actions.ComposeStatic(
				a.Playback,
				interval,
				urlutil.FormatServerAddress(a.Server.Addr),
			)
			if err != nil {
				return fmt.Errorf("composing  manifest: %w", err)
			}
			w.Header().Set("Content-Type", "application/dash+xml")
			_, err = w.Write(out)
			if err != nil {
				return fmt.Errorf("writing manifest: %w", err)
			}
			return nil
		},
	))
	mux.HandleFunc(app.SegmentPath, app.WithError(a.SegmentHandler))

	a.Server.Handler = mux

	go func() {
		slog.Debug("starting server", "addr", a.Server.Addr)
		err = a.Server.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
	}()

	mpdURL, err := url.JoinPath(urlutil.FormatServerAddress(a.Server.Addr), "mpd")
	if err != nil {
		return fmt.Errorf("building URL: %w", err)
	}

	ytdlpOptions := c.YtdlpOptions
	if len(ytdlpOptions) > 0 && ytdlpOptions[0] == "--" {
		ytdlpOptions = ytdlpOptions[1:]
	}
	args := append(
		[]string{
			mpdURL,
			"--newline",
			"--output", buildOutputName(outputContext),
		},
		ytdlpOptions...,
	)

	if err := a.YtdlpRunner.Run(args...); err != nil {
		return fmt.Errorf("downloading failed: %w", err)
	}

	return nil
}

func formatActualLine(side string, moment *playback.RewindMoment) string {
	diffPart := ""

	diff := moment.TimeDifference()
	if diff.Abs() >= time.Second {
		diffPart = fmt.Sprintf(" (%s)", formatDifference(diff, true))
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
	return fmt.Sprintf(
		"%s_%s_%s_%s.%%(ext)s",
		adjustForFilename(ctx.Title, 0),
		ctx.ID,
		formatTime(ctx.InputStartTime),
		formatDuration(ctx.InputDuration),
	)
}

func adjustForFilename(s string, length int) string {
	const maxAdjustedLength = 30

	if length == 0 {
		length = maxAdjustedLength
	}

	slug.MaxLength = length
	slug.Lowercase = false

	return slug.Make(s)
}

func formatTime(t time.Time) string {
	return t.Format("20060102T030405-07")
}

func formatDuration(d time.Duration) string {
	s := d.Truncate(time.Second).String()
	s = strings.ReplaceAll(s, "m0s", "m")
	s = strings.ReplaceAll(s, "h0m", "h")
	return s
}

func formatDifference(diff time.Duration, showPlus bool) string {
	sign := ""
	if diff > 0 && showPlus {
		sign = "+"
	}
	return sign + formatDuration(diff)
}
