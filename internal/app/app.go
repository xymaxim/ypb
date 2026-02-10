package app

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/xymaxim/ypb/internal/actions"
	"github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/internal/input"
	"github.com/xymaxim/ypb/internal/playback"
	"github.com/xymaxim/ypb/internal/playback/fetchers"
	"github.com/xymaxim/ypb/internal/urlutil"
)

const (
	SegmentPath = "/videoplayback/itag/{itag}/sq/{sq}"
	RewindPath  = "/rewind/{rewind}"
)

type App struct {
	Playback    playback.Playbacker
	Server      *http.Server
	Config      *Config
	YtdlpRunner exec.Runner
}

type Config struct {
	Port int
}

func NewApp(ytdlpPath string) *App {
	return &App{
		Config:      &Config{},
		YtdlpRunner: exec.NewCommandRunner(ytdlpPath),
	}
}

func (a *App) Initialize(videoID string, cfg *Config) error {
	a.Config = cfg

	pb, err := playback.NewPlayback(
		videoID,
		&fetchers.YtdlpFetcher{
			VideoID: videoID,
			Runner:  a.YtdlpRunner,
		},
		nil,
	)
	if err != nil {
		return fmt.Errorf("starting playback: %w", err)
	}
	a.Playback = pb

	a.Server = &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		ReadHeaderTimeout: 20 * time.Second,
	}

	return nil
}

func (a *App) RewindHandler(w http.ResponseWriter, r *http.Request) error {
	rewindParam, err := url.PathUnescape(r.PathValue("rewind"))
	if err != nil {
		return fmt.Errorf("unescaping rewind parameter argument: %w", err)
	}

	startMoment, endMoment, err := input.ParseInterval(rewindParam)
	if err != nil {
		return fmt.Errorf("parsing rewind parameter argument: %w", err)
	}

	if err := input.ValidateMoments(startMoment, endMoment); err != nil {
		return fmt.Errorf("bad input interval: %w", err)
	}

	locateCtx, err := actions.NewLocateContext(a.Playback, nil)
	if err != nil {
		return fmt.Errorf("building locate context: %w", err)
	}

	interval, _, err := actions.LocateInterval(a.Playback, startMoment, endMoment, locateCtx)
	if err != nil {
		return fmt.Errorf("locating interval: %w", err)
	}

	out, err := actions.ComposeStatic(
		a.Playback,
		interval,
		urlutil.FormatServerAddress(a.Server.Addr),
	)
	if err != nil {
		return fmt.Errorf("composing manifest: %w", err)
	}

	w.Header().Set("Content-Type", "application/dash+xml")
	_, err = w.Write(out)
	if err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	return nil
}

func (a *App) SegmentHandler(w http.ResponseWriter, r *http.Request) error {
	sq, err := strconv.Atoi(r.PathValue("sq"))
	if err != nil {
		return fmt.Errorf("parsing sq parameter argument: %w", err)
	}

	err = a.Playback.StreamSegment(r.PathValue("itag"), sq, w)
	if err != nil {
		return fmt.Errorf("streaming segment, sq=%d: %w", sq, err)
	}

	return nil
}

func WithError(fn func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := fn(w, r)
		if err != nil {
			msg := fmt.Sprintf("%d %s", http.StatusInternalServerError, err.Error())
			http.Error(w, msg, http.StatusInternalServerError)
		}
	})
}
