package app

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xymaxim/ypb/internal/actions"
	"github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/internal/fetchers"
	"github.com/xymaxim/ypb/internal/input"
	"github.com/xymaxim/ypb/internal/playback"
	"github.com/xymaxim/ypb/internal/urlutil"
)

var segmentPatternRe = regexp.MustCompile(`^/videoplayback/itag/([0-9]+)/sq/([0-9]+)/?$`)

type App struct {
	Playback    *playback.Playback
	Server      *http.Server
	Config      *Config
	YtdlpRunner exec.Runner
}

type Config struct {
	Port int
}

func NewApp() *App {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		ReplaceAttr: nil,
		Level:       slog.LevelDebug,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return &App{
		Config:      &Config{},
		YtdlpRunner: exec.NewCommandRunner("yt-dlp"),
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

func (a *App) RewindHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimRight(r.URL.Path, "/")
	var params []string
	if path != "" {
		params = strings.Split(path, "/")
	}
	if len(params) == 1 {
		slog.Error("missing rewind parameter", "path", path)
		writeError(w, "Missing rewind parameter", http.StatusBadRequest)
		return
	}

	intervalString := params[1]
	_, _, err := input.ParseInterval(intervalString)
	if err != nil {
		slog.Error("parsing input interval", "value", intervalString, "err", err)
		writeError(w, "Error parsing rewind parameter", http.StatusInternalServerError)
		return
	}

	startTime := time.Now().Add(-time.Duration(78+2) * time.Hour)
	endTime := startTime.Add(time.Duration(12) * time.Hour)
	interval, _, err := actions.Locate(a.Playback, startTime, endTime)
	if err != nil {
		slog.Error("locating interval", "err", err)
		writeError(w, "Error locating rewind interval", http.StatusInternalServerError)
		return
	}

	out, _ := actions.ComposeStatic(
		a.Playback,
		interval,
		urlutil.FormatServerAddress(a.Server.Addr),
	)

	w.Header().Set("Content-Type", "application/dash+xml")
	_, err = w.Write(out)
	if err != nil {
		slog.Error("writing manifest", "err", err)
		writeError(w, "Error writing manifest", http.StatusInternalServerError)
		return
	}
}

func (a *App) SegmentHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimRight(r.URL.Path, "/")
	matches := segmentPatternRe.FindStringSubmatch(path)
	if len(matches) != 3 {
		return
	}

	itag, sq := matches[1], matches[2]
	segmentURL, err := url.JoinPath(a.Playback.BaseURLs[itag], "sq", sq)
	if err != nil {
		slog.Error("building segment URL", "itag", itag, "sq", sq)
		writeError(w, "Couldn't build segment URL", http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, segmentURL, r.Body)
	if err != nil {
		slog.Error("creating request", "url", segmentURL, "err", err)
		writeError(w, "Couldn't create segment request", http.StatusInternalServerError)
		return
	}

	resp, err := a.Playback.Client.Do(req)
	if err != nil {
		slog.Error("requesting segment", "url", segmentURL, "err", err)
		writeError(w, "Error requesting segment", http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()
	if _, err := io.Copy(w, resp.Body); err != nil {
		slog.Error("copying response data", "sq", sq, "err", err)
		writeError(w, "Error copying response data", http.StatusInternalServerError)
		return
	}
}

func writeError(w http.ResponseWriter, msg string, code int) {
	http.Error(w, fmt.Sprintf("%d %s", code, msg), code)
}
