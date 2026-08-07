package commands

import (
	"fmt"
	"net/http"

	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/urlutil"
)

type Play struct {
	CommonFlags
	Stream string `arg:"" help:"YouTube video ID" required:""`
	YtdlpOptionsFlag
}

func (c *Play) Run() error {
	if err := checkYtdlp(); err != nil {
		return err
	}

	ytdlpOptions := NormalizeYtdlpOptions(c.YtdlpOptions)
	app, err := apppkg.InitApp(c.Stream, c.Port, ytdlpOptions)
	if err != nil {
		return err
	}

	fmt.Printf("(<<) Stream '%s' is alive!\n", app.Playback.Info().Title)

	mux := http.NewServeMux()
	mux.Handle("/{$}", apppkg.WithError((&apppkg.PlayHandler{}).ServeHTTP))
	mux.HandleFunc(apppkg.InfoPath, apppkg.WithError(
		(&apppkg.InfoHandler{Info: app.Playback.Info()}).ServeHTTP),
	)
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

	app.Server.Handler = apppkg.WithCORS(mux)

	fmt.Printf(
		"(<<) Player started and listening on %s...\n",
		urlutil.FormatServerAddress(app.Server.Addr),
	)

	return app.Server.ListenAndServe()
}
