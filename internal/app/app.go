package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/xymaxim/ypb/fetchers"
	"github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/playback"
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

type statusWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (sw *statusWriter) WriteHeader(code int) {
	if sw.wroteHeader {
		return
	}
	sw.wroteHeader = true
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *statusWriter) Write(b []byte) (int, error) {
	if !sw.wroteHeader {
		sw.WriteHeader(http.StatusOK)
	}
	return sw.ResponseWriter.Write(b)
}

// ServerAddr returns the address for the playback server to listen on. An
// empty host means all interfaces; "127.0.0.1" keeps it to this computer.
func ServerAddr(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func InitApp(id, host string, port int, ytdlpOptions []string) (*App, error) {
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
		Addr:              ServerAddr(host, port),
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

// WithCORS allows the given origins to read responses. Origins are matched
// exactly and no wildcard origin is sent. An empty set disables CORS.
func WithCORS(allowed map[string]bool, next http.Handler) http.Handler {
	if len(allowed) == 0 {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Add("Vary", "Origin")

		origin := r.Header.Get("Origin")
		isPreflight := r.Method == http.MethodOptions &&
			origin != "" &&
			r.Header.Get("Access-Control-Request-Method") != ""

		if isPreflight {
			h.Add("Vary", "Access-Control-Request-Method")
			h.Add("Vary", "Access-Control-Request-Headers")
		}

		if origin != "" && allowed[origin] {
			h.Set("Access-Control-Allow-Origin", origin)

			if isPreflight {
				h.Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Accept, Content-Type, Range")
				h.Set("Access-Control-Max-Age", "600")
			} else {
				h.Set(
					"Access-Control-Expose-Headers",
					"Content-Range, Accept-Ranges",
				)
			}
		}

		if isPreflight {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func WithError(fn func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w}
		err := fn(sw, r)
		if err != nil {
			if sw.wroteHeader {
				slog.Debug("handler failed after writing response", "error", err)
				return
			}
			msg := fmt.Sprintf("%d %s", http.StatusInternalServerError, err.Error())
			http.Error(w, msg, http.StatusInternalServerError)
		}
	})
}
