package commands

import (
	"context"
	"fmt"
	"net/http"

	"github.com/urfave/cli/v3"

	"github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/urlutil"
)

func NewServeCommand(a *app.App) *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Start a playback server for a live stream",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name: "video-id",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			videoID := cmd.StringArg("video-id")
			if videoID == "" {
				return cli.Exit("ypb serve requires a video ID", 2)
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

			return runServe(a, ctx, cmd)
		},
	}
}

func runServe(a *app.App, _ context.Context, _ *cli.Command) error {
	http.HandleFunc("/rewind/", a.RewindHandler)
	http.HandleFunc("/videoplayback/", a.SegmentHandler)

	fmt.Printf(
		"(<<) Playback started and listening on %s...\n",
		urlutil.FormatServerAddress(a.Server.Addr),
	)
	err := a.Server.ListenAndServe()
	if err != nil {
		return err
	}

	return nil
}
