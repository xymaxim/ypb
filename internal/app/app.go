package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
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

// WithHostCheck rejects requests not addressed to a loopback host or to
// allowedHost when the server listens on a loopback address, which blocks DNS
// rebinding. On other addresses it lets every request through.
func WithHostCheck(listenAddr, allowedHost string, next http.Handler) http.Handler {
	if !isLoopback(requestHost(listenAddr)) {
		return next
	}

	allowedHost = requestHost(allowedHost)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := requestHost(r.Host)
		allowedHostMatch := allowedHost != "" && strings.EqualFold(host, allowedHost)
		if !isLoopback(host) && !allowedHostMatch {
			msg := fmt.Sprintf("unexpected Host %q", r.Host)
			http.Error(w, msg, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isLoopback reports whether addr, a host with an optional port, is
// localhost or a loopback IP address.
func isLoopback(addr string) bool {
	host := strings.TrimSuffix(strings.TrimPrefix(addr, "["), "]")
	if h, _, err := net.SplitHostPort(addr); err == nil {
		host = h
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.Unmap().IsLoopback()
}

// requestHost returns the host of a Host header, without the port or IPv6 brackets.
func requestHost(hostport string) string {
	if host, _, err := net.SplitHostPort(hostport); err == nil {
		return host
	}
	return strings.TrimSuffix(strings.TrimPrefix(hostport, "["), "]")
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
