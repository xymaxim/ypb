package commands

import (
	"fmt"
	"net/http"

	apppkg "github.com/xymaxim/ypb/internal/app"
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

	app := apppkg.NewApp()

	if err := CollectVideoInfo(c.Stream, app, c.Port); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc(apppkg.MPDPath, apppkg.WithError(
		(&apppkg.MPDHandler{
			Playback:      app.Playback,
			FFprobeRunner: app.FFprobeRunner,
			ServerAddr:    app.Server.Addr,
		}).ServeHTTP),
	)
	mux.HandleFunc(apppkg.SegmentPath, apppkg.WithError(
		(&apppkg.SegmentHandler{Playback: app.Playback}).ServeHTTP),
	)

	app.Server.Handler = mux

	fmt.Printf(
		"(<<) Playback started and listening on %s...\n",
		urlutil.FormatServerAddress(app.Server.Addr),
	)

	return app.Server.ListenAndServe()
}
