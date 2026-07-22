package app

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/xymaxim/ypb/fetchers"
	"github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/playback"
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
	FFmpegRunner  exec.Runner
	FFprobeRunner exec.Runner
	YtdlpRunner   exec.Runner
	Server        *http.Server
}

func InitApp(id string, port int, ytdlpOptions []string) (*App, error) {
	ytdlpRunner := exec.NewCommandRunner(YtdlpBinaryPath)
	ffmpegRunner := exec.NewCommandRunner(FFmpegBinaryPath)
	ffprobeRunner := exec.NewCommandRunner(FFprobeBinaryPath)

	fetcher := &fetchers.YtdlpFetcher{
		VideoID:      id,
		Runner:       ytdlpRunner,
		YtdlpOptions: ytdlpOptions,
	}

	pb, err := playback.NewPlayback(context.Background(), id, fetcher, nil)
	if err != nil {
		return nil, fmt.Errorf("creating playback: %w", err)
	}

	server := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		ReadHeaderTimeout: 20 * time.Second,
	}

	return &App{
		Playback:      pb,
		FFmpegRunner:  ffmpegRunner,
		FFprobeRunner: ffprobeRunner,
		YtdlpRunner:   ytdlpRunner,
		Server:        server,
	}, nil
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
