package mpd

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xymaxim/ypb/internal/playback/info"
	"github.com/xymaxim/ypb/internal/urlutil"
)

const (
	mpdNamespace      = "urn:mpeg:DASH:schema:MPD:2011"
	mpdProfilesStatic = "urn:mpeg:dash:profile:isoff-main:2011"
	mpdProfilesLive   = "urn:mpeg:dash:profile:isoff-live:2011"
	segmentMediaURL   = "videoplayback/itag/$RepresentationID$/sq/$Number$"
)

type CommonOptions struct {
	BaseURL         string
	StartNumber     int
	SegmentDuration time.Duration
	PTS             float64
}

type StaticOptions struct {
	CommonOptions
	MediaDuration time.Duration
	SegmentCount  int
}

type DynamicOptions struct {
	CommonOptions
	AvailabilityStartTime time.Time
	TimeShiftBufferDepth  time.Duration
}

type MPD struct {
	XMLName                   xml.Name            `xml:"MPD"`
	Xmlns                     string              `xml:"xmlns,attr"`
	Profiles                  string              `xml:"profiles,attr"`
	Type                      string              `xml:"type,attr"`
	AvailabilityStartTime     string              `xml:"availabilityStartTime,attr,omitempty"`
	MediaPresentationDuration string              `xml:"mediaPresentationDuration,attr,omitempty"`
	TimeShiftBufferDepth      string              `xml:"timeShiftBufferDepth,attr,omitempty"`
	ProgramInformation        *ProgramInformation `xml:"ProgramInformation"`
	BaseURL                   string              `xml:"BaseURL"`
	Periods                   []Period            `xml:"Period"`
}

type ProgramInformation struct {
	XMLName xml.Name `xml:"ProgramInformation"`
	Title   string   `xml:"Title"`
	Source  string   `xml:"Source"`
}

type Period struct {
	AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
	ID              int              `xml:"id,attr"`
	MimeType        string           `xml:"mimeType,attr"`
	Representations []Representation `xml:"Representation"`
}

type Representation struct {
	ID                string          `xml:"id,attr"`
	Codecs            string          `xml:"codecs,attr"`
	AudioSamplingRate *int            `xml:"audioSamplingRate,attr,omitempty"`
	Width             *int            `xml:"width,attr,omitempty"`
	Height            *int            `xml:"height,attr,omitempty"`
	FrameRate         *int            `xml:"frameRate,attr,omitempty"`
	SegmentTemplate   SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentTemplate struct {
	Media                  string           `xml:"media,attr"`
	StartNumber            int              `xml:"startNumber,attr"`
	Timescale              string           `xml:"timescale,attr"`
	Duration               string           `xml:"duration,attr,omitempty"`
	PresentationTimeOffset string           `xml:"presentationTimeOffset,attr"`
	SegmentTimeline        *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
	Timeline []S `xml:"S"`
}

type S struct {
	T string `xml:"t,attr"`
	D string `xml:"d,attr"`
	R string `xml:"r,attr"`
}

func ComposeStatic(opts StaticOptions, videoInfo info.VideoInformation) (string, error) {
	m := newMPD(opts.BaseURL, videoInfo)
	m.Type = "static"
	m.Profiles = mpdProfilesStatic
	m.MediaPresentationDuration = formatDuration(opts.MediaDuration)
	m.Periods[0].AdaptationSets = buildAdaptationSets(
		buildStaticSegmentTemplate(opts),
		videoInfo,
	)
	return marshal(m)
}

func ComposeDynamic(opts DynamicOptions, videoInfo info.VideoInformation) (string, error) {
	m := newMPD(opts.BaseURL, videoInfo)
	m.Type = "dynamic"
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

func newMPD(baseURL string, videoInfo info.VideoInformation) MPD {
	return MPD{
		Xmlns:   mpdNamespace,
		BaseURL: baseURL,
		ProgramInformation: &ProgramInformation{
			Title:  videoInfo.Title,
			Source: urlutil.BuildVideoLiveURL(videoInfo.ID),
		},
		Periods: []Period{{}},
	}
}

func buildAdaptationSets(
	template SegmentTemplate,
	videoInfo info.VideoInformation,
) []AdaptationSet {
	period := Period{}

	for _, stream := range videoInfo.AudioStreams {
		set := period.getOrCreateAdaptationSet(stream.MimeType)
		set.Representations = append(set.Representations, Representation{
			ID:                stream.Itag,
			Codecs:            stream.Codecs,
			AudioSamplingRate: &stream.AudioSamplingRate,
			SegmentTemplate:   template,
		})
	}
	for _, stream := range videoInfo.VideoStreams {
		set := period.getOrCreateAdaptationSet(stream.MimeType)
		set.Representations = append(set.Representations, Representation{
			ID:              stream.Itag,
			Codecs:          stream.Codecs,
			Width:           &stream.Width,
			Height:          &stream.Height,
			FrameRate:       &stream.FrameRate,
			SegmentTemplate: template,
		})
	}

	return period.AdaptationSets
}

func baseSegmentTemplate(opts CommonOptions) SegmentTemplate {
	timescale := time.Second.Milliseconds()
	return SegmentTemplate{
		Media:                  segmentMediaURL,
		StartNumber:            opts.StartNumber,
		Timescale:              strconv.FormatInt(timescale, 10),
		PresentationTimeOffset: fmt.Sprintf("%.0f", opts.PTS*float64(timescale)),
	}
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

func buildDynamicSegmentTemplate(opts DynamicOptions) SegmentTemplate {
	t := baseSegmentTemplate(opts.CommonOptions)
	t.Duration = strconv.FormatInt(opts.SegmentDuration.Milliseconds(), 10)
	return t
}

func (period *Period) getOrCreateAdaptationSet(mimeType string) *AdaptationSet {
	for i := range period.AdaptationSets {
		if period.AdaptationSets[i].MimeType == mimeType {
			return &period.AdaptationSets[i]
		}
	}
	period.AdaptationSets = append(period.AdaptationSets, AdaptationSet{
		ID:       len(period.AdaptationSets),
		MimeType: mimeType,
	})
	return &period.AdaptationSets[len(period.AdaptationSets)-1]
}

func formatDuration(dur time.Duration) string {
	return "PT" + strings.ToUpper(dur.Round(100*time.Millisecond).String())
}

func marshal(m MPD) (string, error) {
	output, err := xml.MarshalIndent(m, " ", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling mpd: %w", err)
	}
	return fmt.Sprintf("%s%s\n", xml.Header, string(output)), nil
}
