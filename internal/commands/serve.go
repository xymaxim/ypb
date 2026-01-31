package commands

import (
	"fmt"
	"net/http"

	"github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/urlutil"
)

type Serve struct {
	Stream string `arg:"" help:"YouTube video ID"          required:""`
	Port   int    `       help:"Port to start playback on"             short:"p" default:"8080"`
}

func (c *Serve) Run() error {
	a := app.NewApp()

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
