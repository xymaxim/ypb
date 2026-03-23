package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xymaxim/ypb/internal/actions"
	"github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/internal/input"
	"github.com/xymaxim/ypb/internal/playback"
	"github.com/xymaxim/ypb/internal/urlutil"
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
}

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
	startParsed, endParsed, err := input.ParseInterval(param)
	if err != nil {
		return fmt.Errorf("parsing interval parameter %q: %w", param, err)
	}

	if err := input.ValidateMoments(startParsed, endParsed); err != nil {
		return fmt.Errorf("bad input interval: %w", err)
	}

	locateCtx, err := actions.NewLocateContext(h.Playback, nil, nil)
	if err != nil {
		return fmt.Errorf("building locate context: %w", err)
	}

	rewindInterval, _, err := actions.LocateInterval(
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
	})
}

func (h *MPDHandler) respondDynamicMPD(w http.ResponseWriter, r *http.Request, param string) error {
	parsed, err := input.ParseIntervalPart(param)
	if err != nil {
		return fmt.Errorf("parsing interval parameter %q: %w", param, err)
	}

	locateCtx, err := actions.NewLocateContext(h.Playback, nil, nil)
	if err != nil {
		return fmt.Errorf("building locate context: %w", err)
	}

	rewindMoment, err := actions.LocateMoment(h.Playback, parsed, locateCtx)
	if err != nil {
		return fmt.Errorf("locating moment: %w", err)
	}

	out, err := actions.ComposeDynamic(
		h.Playback,
		rewindMoment,
		urlutil.FormatServerAddress(h.ServerAddr),
		h.FFprobeRunner,
	)
	if err != nil {
		return fmt.Errorf("composing dynamic mpd: %w", err)
	}
	return h.serveMPD(w, r, out, intervalInfo{
		StartActualTime: rewindMoment.ActualTime.UTC(),
		StartTargetTime: rewindMoment.TargetTime.UTC(),
	})
}

func (h *MPDHandler) serveMPD(
	w http.ResponseWriter,
	r *http.Request,
	mpd []byte,
	info intervalInfo,
) error {
	metadata := mpdMetadata{
		VideoTitle:      h.Playback.Info().Title,
		VideoURL:        urlutil.BuildVideoLiveURL(h.Playback.Info().ID),
		StartActualTime: info.StartActualTime,
		StartTargetTime: info.StartTargetTime,
		EndActualTime:   info.EndActualTime,
		EndTargetTime:   info.EndTargetTime,
	}

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
