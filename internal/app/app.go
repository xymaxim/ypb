package app

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/internal/playback"
	"github.com/xymaxim/ypb/internal/playback/fetchers"
)

const (
	InfoPath    = "/info"
	MPDPath     = "/mpd/{interval}"
	SegmentPath = "/segments/itag/{itag}/sq/{sq}"
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

func WithError(fn func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := fn(w, r)
		if err != nil {
			msg := fmt.Sprintf("%d %s", http.StatusInternalServerError, err.Error())
			http.Error(w, msg, http.StatusInternalServerError)
		}
	})
}
