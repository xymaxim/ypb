package app

import (
	"fmt"
	"net/http"
	"strconv"
)

func (a *App) SegmentHandler(w http.ResponseWriter, r *http.Request) error {
	sq, err := strconv.Atoi(r.PathValue("sq"))
	if err != nil {
		return fmt.Errorf("parsing sq parameter: %w", err)
	}

	err = a.Playback.StreamSegment(r.PathValue("itag"), sq, w)
	if err != nil {
		return fmt.Errorf("streaming segment, sq=%d: %w", sq, err)
	}

	return nil
}
