package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xymaxim/ypb/internal/actions"
	"github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/internal/input"
	"github.com/xymaxim/ypb/internal/urlutil"
	"github.com/xymaxim/ypb/playback"
)

type intervalInfo struct {
	StartActualTime time.Time
	StartTargetTime time.Time
	EndActualTime   *time.Time
	EndTargetTime   *time.Time
}

type mpdMetadata struct {
	VideoTitle      string     `json:"videoTitle"`
	VideoURL        string     `json:"videoUrl"`
	OutputName      string     `json:"outputName,omitempty"`
	StartActualTime time.Time  `json:"startActualTime"`
	StartTargetTime time.Time  `json:"startTargetTime"`
	EndActualTime   *time.Time `json:"endActualTime,omitempty"`
	EndTargetTime   *time.Time `json:"endTargetTime,omitempty"`
}

type mpdResponse struct {
	Metadata mpdMetadata `json:"metadata"`
	MPD      string      `json:"mpd"`
}

type MPDHandler struct {
	Playback      playback.Playbacker
	ServerAddr    string
	FFprobeRunner exec.Runner
	dynamicCache  dynamicCache
}

// dynamicCache caches results for dynamic MPD requests, keyed by
// normalized target time, to avoid repeated locate and probe operations.
type dynamicCache struct {
	mu    sync.Mutex
	items map[string]dynamicCacheEntry
}

// dynamicCacheEntry holds the cached data from a dynamic MPD request.
type dynamicCacheEntry struct {
	AvailabilityStartTime time.Time
	StartActualTime       time.Time
	StartTargetTime       time.Time
	LastAccess            time.Time
	probe                 actions.DynamicMomentProbe
}

const (
	dynamicCacheMaxEntries  = 1000
	dynamicCacheIdleTimeout = 24 * time.Hour
)

func (h *MPDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	param, err := url.PathUnescape(r.PathValue("interval"))
	if err != nil {
		return fmt.Errorf("unescaping interval parameter: %w", err)
	}

	if !strings.Contains(param, "/") && !strings.Contains(param, "--") {
		return h.respondDynamicMPD(w, r, param)
	}
	return h.respondStaticMPD(w, r, param)
}

func (h *MPDHandler) respondStaticMPD(w http.ResponseWriter, r *http.Request, param string) error {
	nowTime := time.Now().UTC()

	startParsed, endParsed, err := input.ParseInterval(param, nil)
	if err != nil {
		return fmt.Errorf("parsing interval parameter %q: %w", param, err)
	}

	if err := input.ValidateMoments(startParsed, endParsed, nowTime); err != nil {
		return fmt.Errorf("bad input interval: %w", err)
	}

	latency, err := parseLatencyParam(r)
	if err != nil {
		return err
	}
	for _, mv := range []input.MomentValue{startParsed, endParsed} {
		if err := input.ValidateMoment(mv, latency, nowTime); err != nil {
			return fmt.Errorf("bad input interval: %w", err)
		}
	}

	locateCtx, err := actions.NewLocateContext(h.Playback, nil, nil)
	if err != nil {
		return fmt.Errorf("building locate context: %w", err)
	}
	locateCtx.Latency = latency

	rewindInterval, outputContext, err := actions.LocateInterval(
		h.Playback,
		startParsed,
		endParsed,
		locateCtx,
	)
	if err != nil {
		return fmt.Errorf("locating interval: %w", err)
	}

	mpd, err := actions.ComposeStatic(
		h.Playback,
		rewindInterval,
		urlutil.FormatServerAddress(h.ServerAddr),
		h.FFprobeRunner,
	)
	if err != nil {
		return fmt.Errorf("composing static mpd: %w", err)
	}

	ea := rewindInterval.End.ActualTime.UTC()
	et := rewindInterval.End.TargetTime.UTC()

	return h.serveMPD(w, r, mpd, intervalInfo{
		StartActualTime: rewindInterval.Start.ActualTime.UTC(),
		StartTargetTime: rewindInterval.Start.TargetTime.UTC(),
		EndActualTime:   &ea,
		EndTargetTime:   &et,
	}, outputContext)
}

func (h *MPDHandler) respondDynamicMPD(w http.ResponseWriter, r *http.Request, param string) error {
	nowTime := time.Now().UTC()

	parsed, err := input.ParseIntervalPart(param, nil)
	if err != nil {
		return fmt.Errorf("parsing interval parameter %q: %w", param, err)
	}

	latency, err := parseLatencyParam(r)
	if err != nil {
		return err
	}
	if err := input.ValidateMoment(parsed, latency, nowTime); err != nil {
		return fmt.Errorf("bad input moment: %w", err)
	}

	key, err := dynamicCacheKey(parsed, latency)
	if err != nil {
		return fmt.Errorf("building dynamic cache key: %w", err)
	}

	if entry, ok := h.dynamicCache.get(key); ok {
		out, err := actions.ComposeDynamic(
			h.Playback,
			entry.probe,
			urlutil.FormatServerAddress(h.ServerAddr),
			entry.AvailabilityStartTime,
			nowTime,
		)
		if err != nil {
			return fmt.Errorf("composing dynamic mpd: %w", err)
		}
		return h.serveMPD(w, r, out, intervalInfo{
			StartActualTime: entry.StartActualTime,
			StartTargetTime: entry.StartTargetTime,
		}, nil)
	}

	locateCtx, err := actions.NewLocateContext(h.Playback, nil, nil)
	if err != nil {
		return fmt.Errorf("building locate context: %w", err)
	}
	locateCtx.Latency = latency

	rewindMoment, err := actions.LocateMoment(h.Playback, parsed, locateCtx)
	if err != nil {
		return fmt.Errorf("locating moment: %w", err)
	}

	probe, err := actions.ProbeDynamicMoment(h.Playback, rewindMoment, h.FFprobeRunner)
	if err != nil {
		return fmt.Errorf("probing dynamic moment: %w", err)
	}

	h.dynamicCache.set(key, dynamicCacheEntry{
		AvailabilityStartTime: nowTime,
		StartActualTime:       rewindMoment.ActualTime.UTC(),
		StartTargetTime:       rewindMoment.TargetTime.UTC(),
		probe:                 probe,
	})

	out, err := actions.ComposeDynamic(
		h.Playback,
		probe,
		urlutil.FormatServerAddress(h.ServerAddr),
		nowTime,
		nowTime,
	)
	if err != nil {
		return fmt.Errorf("composing dynamic mpd: %w", err)
	}
	return h.serveMPD(w, r, out, intervalInfo{
		StartActualTime: rewindMoment.ActualTime.UTC(),
		StartTargetTime: rewindMoment.TargetTime.UTC(),
	}, nil)
}

func (h *MPDHandler) serveMPD(
	w http.ResponseWriter,
	r *http.Request,
	mpd []byte,
	info intervalInfo,
	outputContext *actions.LocateOutputContext,
) error {
	metadata := mpdMetadata{
		VideoTitle:      h.Playback.Info().Title,
		VideoURL:        urlutil.BuildVideoLiveURL(h.Playback.Info().ID),
		StartActualTime: info.StartActualTime,
		StartTargetTime: info.StartTargetTime,
		EndActualTime:   info.EndActualTime,
		EndTargetTime:   info.EndTargetTime,
	}
	if outputContext != nil {
		metadata.OutputName = actions.BuildOutputStem(outputContext)
	}

	w.Header().Set("Cache-Control", "no-store")

	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(
			mpdResponse{Metadata: metadata, MPD: string(mpd)},
		)
		if err != nil {
			return fmt.Errorf("writing json response: %w", err)
		}
		return nil
	}

	w.Header().Set("Content-Type", "application/dash+xml")
	if _, err := w.Write(mpd); err != nil {
		return fmt.Errorf("writing mpd: %w", err)
	}
	return nil
}

func parseLatencyParam(r *http.Request) (time.Duration, error) {
	raw := r.URL.Query().Get("latency")
	if raw == "" {
		raw = r.URL.Query().Get("l")
	}
	if raw == "" {
		return 0, nil
	}

	latency, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing latency parameter %q: %w", raw, err)
	}

	if latency < 0 {
		return 0, fmt.Errorf(
			"latency parameter %q must be a non-negative number of seconds",
			raw,
		)
	}

	return time.Duration(latency * float64(time.Second)), nil
}

// dynamicCacheKey derives a cache key for a dynamic MPD request from the parsed
// moment value and the latency.
func dynamicCacheKey(v input.MomentValue, latency time.Duration) (string, error) {
	latencyKey := strconv.FormatInt(int64(latency), 10)

	switch mv := v.(type) {
	case time.Time:
		return fmt.Sprintf(
			"t:%d|%s",
			mv.UTC().Truncate(time.Second).Unix(),
			latencyKey,
		), nil
	case playback.SequenceNumber:
		return fmt.Sprintf("sq:%d|%s", mv, latencyKey), nil
	case input.MomentKeyword:
		return fmt.Sprintf("k:%s|%s", mv, latencyKey), nil
	case input.MomentExpression:
		if mv.Left == input.NowKeyword {
			return fmt.Sprintf(
				"e:%c%s|%s",
				mv.Operator,
				strconv.FormatInt(int64(mv.Right), 10),
				latencyKey,
			), nil
		}
		left, ok := mv.Left.(time.Time)
		if !ok {
			return "", fmt.Errorf(
				"unsupported moment expression left operand type %T",
				mv.Left,
			)
		}
		var target time.Time
		switch mv.Operator {
		case input.OpPlus:
			target = left.Add(mv.Right)
		case input.OpMinus:
			target = left.Add(-mv.Right)
		default:
			return "", fmt.Errorf("unsupported operator %q", mv.Operator)
		}
		return fmt.Sprintf(
			"t:%d|%s",
			target.UTC().Truncate(time.Second).Unix(),
			latencyKey,
		), nil
	default:
		return "", fmt.Errorf("unsupported moment value type %T", v)
	}
}

func (c *dynamicCache) get(key string) (dynamicCacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.items[key]
	if !ok {
		return dynamicCacheEntry{}, false
	}
	if time.Since(entry.LastAccess) > dynamicCacheIdleTimeout {
		delete(c.items, key)
		return dynamicCacheEntry{}, false
	}

	entry.LastAccess = time.Now()
	c.items[key] = entry
	return entry, true
}

func (c *dynamicCache) set(key string, entry dynamicCacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.items) >= dynamicCacheMaxEntries {
		c.items = make(map[string]dynamicCacheEntry)
	}
	if c.items == nil {
		c.items = make(map[string]dynamicCacheEntry)
	}

	entry.LastAccess = time.Now()
	c.items[key] = entry
}
