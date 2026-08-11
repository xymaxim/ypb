package mpd

import (
	"strconv"
	"strings"
	"time"

	"github.com/xymaxim/ypb/info"
)

const (
	mpdProfilesLive = "urn:mpeg:dash:profile:isoff-live:2011"
	mpdTypeDynamic  = "dynamic"
)

type DynamicOptions struct {
	CommonOptions
	Location                   string
	AvailabilityStartTime      time.Time
	TimeShiftBufferDepth       time.Duration
	SuggestedPresentationDelay time.Duration
	MinimumUpdatePeriod        time.Duration
	PublishTime                time.Time
	SegmentRepeatCount         int
}

// ComposeDynamic builds a dynamic MPD manifest.
func ComposeDynamic(opts DynamicOptions, videoInfo info.VideoInformation) (string, error) {
	m := newMPD(opts.BaseURL, videoInfo)
	m.Location = opts.Location
	m.Type = mpdTypeDynamic
	m.Profiles = mpdProfilesLive
	m.AvailabilityStartTime = opts.AvailabilityStartTime.UTC().Format(time.RFC3339)
	if opts.TimeShiftBufferDepth > 0 {
		m.TimeShiftBufferDepth = formatDuration(opts.TimeShiftBufferDepth)
	}
	if opts.SuggestedPresentationDelay > 0 {
		m.SuggestedPresentationDelay = formatDuration(opts.SuggestedPresentationDelay)
	}
	if opts.MinimumUpdatePeriod > 0 {
		m.MinimumUpdatePeriod = formatDuration(opts.MinimumUpdatePeriod)
	}
	if !opts.PublishTime.IsZero() {
		m.PublishTime = opts.PublishTime.UTC().Format(time.RFC3339)
	}
	m.Periods[0].AdaptationSets = buildAdaptationSets(
		buildDynamicSegmentTemplate(opts, opts.AudioPTS),
		buildDynamicSegmentTemplate(opts, opts.VideoPTS),
		videoInfo,
	)
	output, err := marshal(m)
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(output, "></S>", "/>"), nil
}

func buildDynamicSegmentTemplate(opts DynamicOptions, pts int64) SegmentTemplate {
	t := baseSegmentTemplate(opts.CommonOptions, pts)
	t.SegmentTimeline = &SegmentTimeline{
		Timeline: []*S{
			{
				T: t.PresentationTimeOffset,
				D: strconv.FormatInt(opts.SegmentDuration.Milliseconds(), 10),
				R: strconv.Itoa(opts.SegmentRepeatCount),
			},
		},
	}
	return t
}
