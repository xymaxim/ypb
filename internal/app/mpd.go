package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xymaxim/ypb/internal/actions"
	"github.com/xymaxim/ypb/internal/input"
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
	StartActualTime time.Time  `json:"startActual"`
	StartTargetTime time.Time  `json:"startTarget"`
	EndActualTime   *time.Time `json:"endActual,omitempty"`
	EndTargetTime   *time.Time `json:"endTarget,omitempty"`
}

type mpdResponse struct {
	Metadata mpdMetadata `json:"metadata"`
	MPD      string      `json:"mpd"`
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

	mpd, err := actions.ComposeStatic(
		a.Playback,
		rewindInterval,
		urlutil.FormatServerAddress(a.Server.Addr),
		a.FFprobeRunner,
	)
	if err != nil {
		return fmt.Errorf("composing static mpd: %w", err)
	}

	return a.serveMPD(w, r, mpd, intervalInfo{
		StartActualTime: rewindInterval.Start.ActualTime,
		StartTargetTime: rewindInterval.Start.TargetTime,
		EndActualTime:   &rewindInterval.End.ActualTime,
		EndTargetTime:   &rewindInterval.End.TargetTime,
	})
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

	return a.serveMPD(w, r, out, intervalInfo{
		StartActualTime: rewindMoment.ActualTime,
		StartTargetTime: rewindMoment.TargetTime,
	})
}

func (a *App) serveMPD(
	w http.ResponseWriter,
	r *http.Request,
	mpd []byte,
	info intervalInfo,
) error {
	metadata := mpdMetadata{
		VideoTitle:      a.Playback.Info().Title,
		VideoURL:        urlutil.BuildVideoLiveURL(a.Playback.Info().ID),
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
