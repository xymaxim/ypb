package commands

import (
	"fmt"
	"net/http"

	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/player"
	"github.com/xymaxim/ypb/internal/urlutil"
)

type Play struct {
	ServerFlags
	Stream string `arg:"" help:"YouTube video ID" required:""`
	YtdlpOptionsFlag
}

func (c *Play) Run() error {
	if err := CheckYtdlp(); err != nil {
		return err
	}

	ytdlpOptions := NormalizeYtdlpOptions(c.YtdlpOptions)
	app, err := apppkg.InitApp(c.Stream, string(c.Host), int(c.Port), ytdlpOptions)
	if err != nil {
		return err
	}

	ln, err := apppkg.Listen(string(c.Host), int(c.Port))
	if err != nil {
		return err
	}
	app.Server.Addr = ln.Addr().String()

	fmt.Printf("(<<) Stream '%s' is alive!\n", app.Playback.Info().Title)

	mux := http.NewServeMux()
	mux.Handle("/{$}", apppkg.WithError((&player.PlayHandler{}).ServeRoot))
	mux.Handle("/", apppkg.WithError((&player.PlayHandler{}).ServeHTTP))
	mux.Handle("/{interval}", apppkg.WithError((&player.PlayHandler{}).ServePage))
	mux.HandleFunc(apppkg.InfoPath, apppkg.WithError(
		(&apppkg.InfoHandler{Info: app.Playback.Info()}).ServeHTTP,
	),
	)
	mux.HandleFunc(apppkg.MPDPath, apppkg.WithError(
		(&apppkg.MPDHandler{
			Playback:      app.Playback,
			FFprobeRunner: app.FFprobeRunner,
			ServerAddr:    app.Server.Addr,
		}).ServeHTTP,
	),
	)
	mux.HandleFunc(apppkg.SegmentPath, apppkg.WithError(
		(&apppkg.SegmentHandler{Playback: app.Playback}).ServeHTTP,
	),
	)

	app.Server.Handler = apppkg.WithHostCheck(
		app.Server.Addr,
		string(c.Host),
		mux,
	)

	fmt.Printf(
		"(<<) Player started and listening on %s...\n",
		urlutil.FormatServerAddress(app.Server.Addr),
	)

	return app.Server.Serve(ln)
}
