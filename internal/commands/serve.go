package commands

import (
	"fmt"
	"net/http"

	"github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/urlutil"
)

type Serve struct {
	Stream    string `arg:"" help:"YouTube video ID"          required:""`
	Port      int    `       help:"Port to start playback on"             default:"8080"   short:"p"`
	YtdlpPath string `       help:"Path to yt-dlp binary"                 default:"yt-dlp"           type:"path"`
}

func (c *Serve) Run() error {
	if err := checkYtdlp(c.YtdlpPath); err != nil {
		return err
	}

	a := app.NewApp(c.YtdlpPath)

	if err := collectVideoInfo(c.Stream, a, c.Port); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc(app.RewindPath, app.WithError(a.RewindHandler))
	mux.HandleFunc(app.SegmentPath, app.WithError(a.SegmentHandler))

	a.Server.Handler = mux

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
