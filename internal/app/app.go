package app

import (
	"context"
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
	Port    int
	OnPrint func([]byte)
}

func NewApp() *App {
	return &App{
		Config:        &Config{},
		FFmpegRunner:  exec.NewCommandRunner(FFmpegBinaryPath),
		FFprobeRunner: exec.NewCommandRunner(FFprobeBinaryPath),
		YtdlpRunner:   exec.NewCommandRunner(YtdlpBinaryPath),
	}
}

func (a *App) Initialize(ctx context.Context, videoID string, cfg *Config, fetcher fetchers.Fetcher) error {
	a.Config = cfg

	pb, err := playback.NewPlayback(
		ctx,
		videoID,
		fetcher,
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

func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
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
