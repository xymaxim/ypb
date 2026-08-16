package mockserver

import (
	"net/http"
	"time"

	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/playback"
)

func New(fixtureDir string, actualStartTime time.Time, addr string, ui bool) (http.Handler, error) {
	pb, err := playback.NewMockPlayback(fixtureDir, actualStartTime)
	if err != nil {
		return nil, err
	}

	ffprobeRunner := exec.NewCommandRunner(apppkg.FFprobeBinaryPath)

	app := &apppkg.App{
		Playback:      pb,
		FFprobeRunner: ffprobeRunner,
		Server:        &http.Server{Addr: addr},
	}

	mux := http.NewServeMux()
	if ui {
		apppkg.RegisterPlayRoutes(mux)
	}
	apppkg.RegisterInfoRoute(mux, app)
	apppkg.RegisterMPDRoute(mux, app)
	apppkg.RegisterSegmentRoute(mux, app)

	return apppkg.WithCORS(mux), nil
}
