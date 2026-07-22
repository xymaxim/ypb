package mpd

import (
	"strconv"
	"time"

	"github.com/xymaxim/ypb/info"
)

const (
	mpdProfilesLive = "urn:mpeg:dash:profile:isoff-live:2011"
	mpdTypeDynamic  = "dynamic"
)

type DynamicOptions struct {
	CommonOptions
	AvailabilityStartTime time.Time
	TimeShiftBufferDepth  time.Duration
}

// ComposeDynamic builds a dynamic MPD manifest.
func ComposeDynamic(opts DynamicOptions, videoInfo info.VideoInformation) (string, error) {
	m := newMPD(opts.BaseURL, videoInfo)
	m.Type = mpdTypeDynamic
	m.Profiles = mpdProfilesLive
	m.AvailabilityStartTime = opts.AvailabilityStartTime.UTC().Format(time.RFC3339)
	if opts.TimeShiftBufferDepth > 0 {
		m.TimeShiftBufferDepth = formatDuration(opts.TimeShiftBufferDepth)
	}
	m.Periods[0].AdaptationSets = buildAdaptationSets(
		buildDynamicSegmentTemplate(opts),
		videoInfo,
	)
	return marshal(m)
}

func buildDynamicSegmentTemplate(opts DynamicOptions) SegmentTemplate {
	t := baseSegmentTemplate(opts.CommonOptions)
	t.Duration = strconv.FormatInt(opts.SegmentDuration.Milliseconds(), 10)
	return t
}
