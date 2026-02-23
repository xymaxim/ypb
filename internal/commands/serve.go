package commands

import (
	"fmt"
	"net/http"

	"github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/urlutil"
)

type Serve struct {
	CommonFlags
	Stream string `arg:"" help:"YouTube video ID" required:""`
}

func (c *Serve) Run() error {
	if err := checkYtdlp(); err != nil {
		return err
	}

	a := app.NewApp()

	if err := CollectVideoInfo(c.Stream, a, c.Port); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc(app.MPDPath, app.WithError(
		(&app.MPDHandler{
			Playback:      a.Playback,
			FFprobeRunner: a.FFprobeRunner,
			ServerAddr:    a.Server.Addr,
		}).ServeHTTP),
	)
	mux.HandleFunc(app.SegmentPath, app.WithError(
		(&app.SegmentHandler{Playback: a.Playback}).ServeHTTP),
	)

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
