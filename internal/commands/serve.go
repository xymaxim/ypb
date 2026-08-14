package commands

import (
	"fmt"
	"net/http"

	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/urlutil"
)

type Serve struct {
	CommonFlags
	UI     bool   `help:"Also serve the web player"`
	Stream string `help:"YouTube video ID"          arg:"" required:""`
	YtdlpOptionsFlag
}

func (c *Serve) Run() error {
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
	if c.UI {
		mux.Handle("/{$}", apppkg.WithError((&apppkg.PlayHandler{}).ServeRoot))
		mux.Handle("/", apppkg.WithError((&apppkg.PlayHandler{}).ServeHTTP))
		mux.Handle("/{interval}", apppkg.WithError((&apppkg.PlayHandler{}).ServePage))
	}
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
		"(<<) Playback started and listening on %s...\n",
		urlutil.FormatServerAddress(app.Server.Addr),
	)

	if c.UI {
		fmt.Printf(
			":::: Open %s/now in your browser to play\n",
			urlutil.FormatServerAddress(app.Server.Addr),
		)
	}

	return app.Server.ListenAndServe()
}
