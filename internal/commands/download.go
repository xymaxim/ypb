package commands

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/xymaxim/ypb/internal/actions"
	"github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/input"
	"github.com/xymaxim/ypb/internal/pathutil"
	"github.com/xymaxim/ypb/internal/playback"
	"github.com/xymaxim/ypb/internal/urlutil"
)

func NewDownloadCommand(a *app.App) *cli.Command {
	return &cli.Command{
		Name:  "download",
		Usage: "Download live stream excerpts",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name: "video-id",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "interval",
				Usage: "interval to rewind",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.StringArg("video-id") == "" {
				return fmt.Errorf("%s requires a video ID", cmd.FullName())
			}
			return runDownload(a, ctx, cmd)
		},
	}
}

func runDownload(a *app.App, _ context.Context, cmd *cli.Command) error {
	intervalInput := cmd.String("interval")
	if !cmd.IsSet("interval") {
		return fmt.Errorf("%s requires the --interval flag", cmd.FullName())
	}

	start, end, err := input.ParseInterval(intervalInput)
	if err != nil {
		return fmt.Errorf("parsing input interval: w%", err)
	}
	if err := validateInputMoments(start, end); err != nil {
		return fmt.Errorf("bad input interval: %w", err)
	}

	videoID := cmd.StringArg("video-id")
	videoURL := urlutil.BuildVideoLiveURL(videoID)
	fmt.Printf("(<<) Collecting info about %s...\n", videoURL)

	cfg := &app.Config{Port: cmd.Int("port")}
	if err := a.Initialize(videoID, cfg); err != nil {
		return fmt.Errorf("initializing app: %w", err)
	}

	fmt.Printf("Stream '%s' is alive!\n", a.Playback.Info().Title)

	fmt.Println("(<<) Locating start and end moments...")
	locateContext, err := actions.NewLocateContext(a.Playback, nil)
	if err != nil {
		return cli.Exit(fmt.Errorf("building locate context: %w", err), 1)
	}

	interval, outputContext, err := actions.LocateInterval(
		a.Playback,
		start,
		end,
		locateContext,
	)
	if err != nil {
		return cli.Exit(fmt.Errorf("locating interval: %w", err), 1)
	}

	fmt.Printf(
		"Actual start: %s, sq=%d\n",
		interval.Start.ActualTime.Format(time.RFC1123Z),
		interval.Start.Metadata.SequenceNumber,
	)
	fmt.Printf(
		"  Actual end: %s, sq=%d\n",
		interval.End.ActualTime.Format(time.RFC1123Z),
		interval.End.Metadata.SequenceNumber,
	)

	http.HandleFunc("/mpd", func(w http.ResponseWriter, r *http.Request) {
		out, err := actions.ComposeStatic(
			a.Playback,
			interval,
			urlutil.FormatServerAddress(a.Server.Addr),
		)
		if err != nil {
			http.Error(w, "Error composing MPD", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/dash+xml")
		_, err = w.Write(out)
		if err != nil {
			http.Error(w, "Error writing manifest", http.StatusInternalServerError)
			return
		}
	})
	http.HandleFunc("/videoplayback/", a.SegmentHandler)

	go func() {
		slog.Debug("starting server", "addr", a.Server.Addr)
		err = a.Server.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
	}()

	u, err := url.JoinPath(urlutil.FormatServerAddress(a.Server.Addr), "mpd")
	if err != nil {
		return cli.Exit(fmt.Errorf("building URL: %w", err), 1)
	}
	err = a.YtdlpRunner.Run(u, "--newline", "--output", buildOutputName(outputContext))
	if err != nil {
		return cli.Exit(fmt.Errorf("downloading failed: %w", err), 1)
	}

	return nil
}

func buildOutputName(ctx *actions.LocateOutputContext) string {
	return fmt.Sprintf(
		"%s_%s_%s_%s.%%(ext)s",
		pathutil.AdjustForFilename(ctx.Title, 0),
		ctx.ID,
		pathutil.FormatTime(ctx.InputStartTime),
		pathutil.FormatDuration(ctx.InputDuration),
	)
}

// validateInputMoments performs preliminary validation on start and end moment values
// to catch obvious errors before locating them.
func validateInputMoments(start, end input.MomentValue) error {
	switch s := start.(type) {
	case time.Time:
		if e, ok := end.(time.Time); ok && s.After(e) {
			return fmt.Errorf("start time is after end time: %v > %v", s, e)
		}

	case playback.SequenceNumber:
		if e, ok := end.(playback.SequenceNumber); ok && s > e {
			return fmt.Errorf(
				"start segment is after end segment: %d > %d", s, e)
		}

	case time.Duration:
		if _, ok := end.(time.Duration); ok {
			return errors.New("both start and end cannot be durations")
		}

	case input.MomentKeyword:
		if s == input.NowKeyword {
			return fmt.Errorf("'%s' cannot be used at start", input.NowKeyword)
		}
	}

	return nil
}
