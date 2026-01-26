package commands

import (
	"context"
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
			videoID := cmd.StringArg("video-id")
			if videoID == "" {
				return cli.Exit("ypb download requires a video ID", 2)
			}

			fmt.Printf(
				"(<<) Collecting info about %s...\n",
				urlutil.BuildVideoLiveURL(videoID),
			)
			cfg := &app.Config{
				Port: cmd.Int("port"),
			}
			if err := a.Initialize(videoID, cfg); err != nil {
				return fmt.Errorf("initializing app: %w", err)
			}

			fmt.Printf("Stream '%s' is alive!\n", a.Playback.Info().Title)

			return runDownload(a, ctx, cmd)
		},
	}
}

func runDownload(a *app.App, _ context.Context, cmd *cli.Command) error {
	intervalInput := cmd.String("interval")
	start, end, err := input.ParseInterval(intervalInput)
	if err != nil {
		slog.Error("parsing input interval", "value", intervalInput, "err", err)
		return err
	}

	fmt.Println("(<<) Locating start and end moments...")

	referenceSeqNum, err := a.Playback.RequestHeadSeqNum()
	if err != nil {
		return cli.Exit(err, 1)
	}
	reference, err := a.Playback.FetchSegmentMetadata(
		a.Playback.ProbeItag(),
		referenceSeqNum,
	)
	if err != nil {
		return cli.Exit(err, 1)
	}

	interval, actionContext, err := actions.LocateInterval(
		a.Playback,
		start,
		end,
		*reference,
	)
	if err != nil {
		return cli.Exit(err, 1)
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
	err = a.YtdlpRunner.Run(u, "--newline", "--output", buildOutputName(actionContext))
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
