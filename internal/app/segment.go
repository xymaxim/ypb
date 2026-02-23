package app

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/xymaxim/ypb/internal/playback"
)

type SegmentHandler struct {
	Playback playback.Playbacker
}

func (h *SegmentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	sq, err := strconv.Atoi(r.PathValue("sq"))
	if err != nil {
		return fmt.Errorf("parsing sq parameter: %w", err)
	}

	err = h.Playback.StreamSegment(r.PathValue("itag"), sq, w)
	if err != nil {
		return fmt.Errorf("streaming segment, sq=%d: %w", sq, err)
	}

	return nil
}
