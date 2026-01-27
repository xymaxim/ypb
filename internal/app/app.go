package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
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

var segmentPatternRe = regexp.MustCompile(`^/videoplayback/itag/([0-9]+)/sq/([0-9]+)/?$`)

type App struct {
	Playback    playback.Playbacker
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

	locateCtx, err := actions.NewLocateContext(a.Playback, nil)
	if err != nil {
		slog.Error("building locate context", "err", err)
		writeError(w, "Error building locate context", http.StatusInternalServerError)
		return
	}

	interval, _, err := actions.LocateInterval(a.Playback, startTime, endTime, locateCtx)
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

	itag, sqRaw := matches[1], matches[2]
	sq, err := strconv.Atoi(sqRaw)
	if err != nil {
		slog.Error("parsing sq parameter", "value", sqRaw, "err", err)
		writeError(w, "parsing sq parameter value: "+sqRaw, http.StatusInternalServerError)
		return
	}

	segment, err := a.Playback.DownloadSegment(itag, sq)
	if err != nil {
		slog.Error("downloading segment", "sq", sq, "err", err)
		writeError(
			w,
			fmt.Sprintf("Error downloading segment sq=%d", sq),
			http.StatusInternalServerError,
		)
		return
	}
	w.Write(segment)
}

func writeError(w http.ResponseWriter, msg string, code int) {
	http.Error(w, fmt.Sprintf("%d %s", code, msg), code)
}
