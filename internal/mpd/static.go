package mpd

import (
	"strconv"
	"time"

	"github.com/xymaxim/ypb/internal/playback/info"
)

const (
	mpdProfilesStatic = "urn:mpeg:dash:profile:isoff-main:2011"
	mpdTypeStatic     = "static"
)

type StaticOptions struct {
	CommonOptions
	MediaDuration time.Duration
	SegmentCount  int
}

// ComposeStatic builds a static MPD manifest.
func ComposeStatic(opts StaticOptions, videoInfo info.VideoInformation) (string, error) {
	m := newMPD(opts.BaseURL, videoInfo)
	m.Type = mpdTypeStatic
	m.Profiles = mpdProfilesStatic
	m.MediaPresentationDuration = formatDuration(opts.MediaDuration)
	m.Periods[0].AdaptationSets = buildAdaptationSets(
		buildStaticSegmentTemplate(opts),
		videoInfo,
	)
	return marshal(m)
}

func buildStaticSegmentTemplate(opts StaticOptions) SegmentTemplate {
	t := baseSegmentTemplate(opts.CommonOptions)
	t.SegmentTimeline = &SegmentTimeline{
		Timeline: []S{
			{
				T: t.PresentationTimeOffset,
				D: strconv.FormatInt(opts.SegmentDuration.Milliseconds(), 10),
				R: strconv.Itoa(opts.SegmentCount - 1),
			},
		},
	}
	return t
}
