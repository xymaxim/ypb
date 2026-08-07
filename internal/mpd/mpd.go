package mpd

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xymaxim/ypb/info"
	"github.com/xymaxim/ypb/internal/urlutil"
)

const (
	mpdNamespace    = "urn:mpeg:DASH:schema:MPD:2011"
	segmentMediaURL = "segments/itag/$RepresentationID$/sq/$Number$"
)

type CommonOptions struct {
	BaseURL         string
	StartNumber     int
	SegmentDuration time.Duration
	PTS             float64
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
	Codecs          string           `xml:"codecs,attr,omitempty"`
	Representations []Representation `xml:"Representation"`

	family string // grouping key, e.g. "avc1"; not exported
}

type Representation struct {
	ID                string          `xml:"id,attr"`
	Bandwidth         int             `xml:"bandwidth,attr"`
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

// buildAdaptationSets groups audio/video streams into adaptation sets by
// codec family and labels each with its representative codecs attribute.
func buildAdaptationSets(
	template SegmentTemplate,
	videoInfo info.VideoInformation,
) []AdaptationSet {
	period := Period{}

	addAudioRepresentations(&period, videoInfo.AudioStreams, template)
	addVideoRepresentations(&period, videoInfo.VideoStreams, template)

	for i := range period.AdaptationSets {
		period.AdaptationSets[i].setRepresentativeCodecs()
	}

	return period.AdaptationSets
}

func addAudioRepresentations(period *Period, streams []info.AudioStream, template SegmentTemplate) {
	for _, stream := range streams {
		set := period.getOrCreateAdaptationSet(stream.MimeType, stream.Codecs)
		set.Representations = append(set.Representations, Representation{
			ID:                stream.Itag,
			Bandwidth:         int(stream.Tbr * 1000),
			Codecs:            stream.Codecs,
			AudioSamplingRate: &stream.AudioSamplingRate,
			SegmentTemplate:   template,
		})
	}
}

func addVideoRepresentations(period *Period, streams []info.VideoStream, template SegmentTemplate) {
	for _, stream := range streams {
		set := period.getOrCreateAdaptationSet(stream.MimeType, stream.Codecs)
		set.Representations = append(set.Representations, Representation{
			ID:              stream.Itag,
			Bandwidth:       int(stream.Tbr * 1000),
			Codecs:          stream.Codecs,
			Width:           &stream.Width,
			Height:          &stream.Height,
			FrameRate:       &stream.FrameRate,
			SegmentTemplate: template,
		})
	}
}

func (period *Period) getOrCreateAdaptationSet(mimeType, codecs string) *AdaptationSet {
	family := codecFamily(codecs)

	for i := range period.AdaptationSets {
		if period.AdaptationSets[i].MimeType == mimeType &&
			period.AdaptationSets[i].family == family {
			return &period.AdaptationSets[i]
		}
	}

	period.AdaptationSets = append(period.AdaptationSets, AdaptationSet{
		ID:       len(period.AdaptationSets),
		MimeType: mimeType,
		family:   family,
	})
	return &period.AdaptationSets[len(period.AdaptationSets)-1]
}

func (set *AdaptationSet) setRepresentativeCodecs() {
	if len(set.Representations) == 0 || set.Representations[0].Height == nil {
		return
	}

	best := set.Representations[0]
	for _, r := range set.Representations[1:] {
		if r.Height == nil {
			continue
		}
		switch {
		case *r.Height > *best.Height:
			best = r
		case *r.Height == *best.Height && r.FrameRate != nil &&
			(best.FrameRate == nil || *r.FrameRate > *best.FrameRate):
			best = r
		}
	}

	set.Codecs = best.Codecs
}

func codecFamily(codecs string) string {
	before, _, found := strings.Cut(codecs, ".")
	if found {
		return before
	}
	return codecs
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
