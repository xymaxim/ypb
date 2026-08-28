package commands

import (
	"fmt"
	"net/http"

	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/player"
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
		player.RegisterPlayRoutes(mux)
	}
	apppkg.RegisterInfoRoute(mux, app)
	apppkg.RegisterSegmentRoute(mux, app)
	apppkg.RegisterMPDRoute(mux, app)

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
