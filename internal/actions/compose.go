package actions

import (
	"strconv"
	"time"

	"github.com/xymaxim/ypb/internal/mpd"
	"github.com/xymaxim/ypb/internal/playback"
)

const segmentMediaURL = "videoplayback/itag/$RepresentationID$/sq/$Number$"

func ComposeStatic(
	pb playback.Playbacker,
	interval *playback.RewindInterval,
	baseURL string,
) ([]byte, error) {
	timescale := time.Millisecond
	segmentDuration := pb.Info().SegmentDuration
	mpdInfo := mpd.Information{
		AvailabilityStartTime:     interval.Start.Metadata.Time(),
		MediaPresentationDuration: interval.Duration(),
		RepresentationBaseURL:     baseURL,
		SegmentTemplate: &mpd.SegmentTemplate{
			Media:       segmentMediaURL,
			StartNumber: interval.Start.Metadata.SequenceNumber,
			Duration:    strconv.Itoa(int(segmentDuration / timescale)),
			Timescale:   strconv.Itoa(int(time.Second / timescale)),
		},
	}

	out := mpd.ComposeStaticMPD(mpdInfo, pb.Info())

	return []byte(out), nil
}
