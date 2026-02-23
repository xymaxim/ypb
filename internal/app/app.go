package app

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/xymaxim/ypb/internal/actions"
	"github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/internal/input"
	"github.com/xymaxim/ypb/internal/playback"
	"github.com/xymaxim/ypb/internal/playback/fetchers"
	"github.com/xymaxim/ypb/internal/urlutil"
)

const (
	SegmentPath = "/segments/itag/{itag}/sq/{sq}"
	MPDPath     = "/mpd/{interval}"
)

const (
	FFmpegBinaryPath  = "ffmpeg"
	FFprobeBinaryPath = "ffprobe"
	YtdlpBinaryPath   = "yt-dlp"
)

type App struct {
	Playback      playback.Playbacker
	Server        *http.Server
	Config        *Config
	FFmpegRunner  exec.Runner
	FFprobeRunner exec.Runner
	YtdlpRunner   exec.Runner
}

type Config struct {
	Port int
}

func NewApp() *App {
	return &App{
		Config:        &Config{},
		FFmpegRunner:  exec.NewCommandRunner(FFmpegBinaryPath),
		FFprobeRunner: exec.NewCommandRunner(FFprobeBinaryPath),
		YtdlpRunner:   exec.NewCommandRunner(YtdlpBinaryPath),
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

func (a *App) MPDHandler(w http.ResponseWriter, r *http.Request) error {
	param, err := url.PathUnescape(r.PathValue("interval"))
	if err != nil {
		return fmt.Errorf("unescaping interval parameter: %w", err)
	}

	if !strings.Contains(param, "/") && !strings.Contains(param, "--") {
		return a.respondDynamicMPD(w, r, param)
	}
	return a.respondStaticMPD(w, r, param)
}

func (a *App) respondStaticMPD(w http.ResponseWriter, r *http.Request, param string) error {
	startParsed, endParsed, err := input.ParseInterval(param)
	if err != nil {
		return fmt.Errorf("parsing interval parameter %q: %w", param, err)
	}

	if err := input.ValidateMoments(startParsed, endParsed); err != nil {
		return fmt.Errorf("bad input interval: %w", err)
	}

	locateCtx, err := actions.NewLocateContext(a.Playback, nil)
	if err != nil {
		return fmt.Errorf("building locate context: %w", err)
	}

	rewindInterval, _, err := actions.LocateInterval(
		a.Playback,
		startParsed,
		endParsed,
		locateCtx,
	)
	if err != nil {
		return fmt.Errorf("locating interval: %w", err)
	}

	out, err := actions.ComposeStatic(
		a.Playback,
		rewindInterval,
		urlutil.FormatServerAddress(a.Server.Addr),
		a.FFprobeRunner,
	)
	if err != nil {
		return fmt.Errorf("composing static mpd: %w", err)
	}

	return a.serveMPD(w, r, out)
}

func (a *App) respondDynamicMPD(w http.ResponseWriter, r *http.Request, param string) error {
	parsed, err := input.ParseIntervalPart(param)
	if err != nil {
		return fmt.Errorf("parsing interval parameter %q: %w", param, err)
	}

	locateCtx, err := actions.NewLocateContext(a.Playback, nil)
	if err != nil {
		return fmt.Errorf("building locate context: %w", err)
	}

	rewindMoment, err := actions.LocateMoment(a.Playback, parsed, locateCtx)
	if err != nil {
		return fmt.Errorf("locating moment: %w", err)
	}

	out, err := actions.ComposeDynamic(
		a.Playback,
		rewindMoment,
		urlutil.FormatServerAddress(a.Server.Addr),
		a.FFprobeRunner,
	)
	if err != nil {
		return fmt.Errorf("composing dynamic mpd: %w", err)
	}

	return a.serveMPD(w, r, out)
}

func (a *App) serveMPD(w http.ResponseWriter, r *http.Request, out []byte) error {
	w.Header().Set("Content-Type", "application/dash+xml")
	if _, err := w.Write(out); err != nil {
		return fmt.Errorf("writing mpd: %w", err)
	}
	return nil
}

func (a *App) SegmentHandler(w http.ResponseWriter, r *http.Request) error {
	sq, err := strconv.Atoi(r.PathValue("sq"))
	if err != nil {
		return fmt.Errorf("parsing sq parameter: %w", err)
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
