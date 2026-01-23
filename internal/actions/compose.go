package actions

import (
	"strconv"

	"github.com/xymaxim/ypb/internal/mpd"
	"github.com/xymaxim/ypb/internal/playback"
)

const segmentMediaURL = "videoplayback/itag/$RepresentationID$/sq/$Number$"

func ComposeStatic(
	pb *playback.Playback,
	interval *playback.RewindInterval,
	baseURL string,
) ([]byte, error) {
	segmentDuration := strconv.FormatInt(pb.Info.SegmentDuration.Milliseconds(), 10)
	mpdInfo := mpd.Information{
		AvailabilityStartTime:     interval.Start.Time,
		MediaPresentationDuration: interval.End.Time.Sub(interval.Start.Time),
		RepresentationBaseURL:     baseURL,
		SegmentTemplate: &mpd.SegmentTemplate{
			Media:       segmentMediaURL,
			StartNumber: interval.Start.SequenceNumber,
			Duration:    segmentDuration,
			Timescale:   "1000",
		},
	}

	out := mpd.ComposeStaticMPD(mpdInfo, pb.Info)

	return []byte(out), nil
}
