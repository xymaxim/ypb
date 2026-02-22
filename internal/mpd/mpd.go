package mpd

import (
	"encoding/xml"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/xymaxim/ypb/internal/playback/info"
	"github.com/xymaxim/ypb/internal/urlutil"
)

const (
	mpdNamespace = "urn:mpeg:DASH:schema:MPD:2011"
	mpdProfiles  = "urn:mpeg:dash:profile:isoff-main:2011"
)

type MPD struct {
	XMLName                   xml.Name            `xml:"MPD"`
	Xmlns                     string              `xml:"xmlns,attr"`
	Profiles                  string              `xml:"profiles,attr"`
	Type                      string              `xml:"type,attr"`
	AvailabilityStartTime     string              `xml:"availabilityStartTime,attr,omitempty"`
	MediaPresentationDuration string              `xml:"mediaPresentationDuration,attr"`
	ProgramInformation        *ProgramInformation `xml:"ProgramInformation"`
	Periods                   []Period            `xml:"Period"`
}

type ProgramInformation struct {
	XMLName xml.Name `xml:"ProgramInformation"`
	Title   string   `xml:"Title"`
	Source  string   `xml:"Source"`
}

type Period struct {
	Duration       string          `xml:"duration,attr"`
	AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
	ID                  int              `xml:"id,attr"`
	MimeType            string           `xml:"mimeType,attr"`
	SubsegmentAlignment string           `xml:"subsegmentAlignment,attr"`
	Representations     []Representation `xml:"Representation"`
}

type Representation struct {
	ID                string          `xml:"id,attr"`
	Codecs            string          `xml:"codecs,attr"`
	AudioSamplingRate *int            `xml:"audioSamplingRate,attr,omitempty"`
	Width             *int            `xml:"width,attr,omitempty"`
	Height            *int            `xml:"height,attr,omitempty"`
	FrameRate         *int            `xml:"frameRate,attr,omitempty"`
	BaseURL           string          `xml:"BaseURL"`
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
	Timescale []S `xml:"S"`
}

type S struct {
	D int `xml:"d,attr"`
	R int `xml:"r,attr"`
}

type Information struct {
	AvailabilityStartTime     time.Time
	MediaPresentationDuration time.Duration
	RepresentationBaseURL     string
	SegmentTemplate           *SegmentTemplate
}

func (period *Period) getOrCreateAdaptationSet(mimeType string) *AdaptationSet {
	for i := range period.AdaptationSets {
		set := &period.AdaptationSets[i]
		if set.MimeType == mimeType {
			return set
		}
	}
	set := AdaptationSet{
		ID:                  len(period.AdaptationSets),
		MimeType:            mimeType,
		SubsegmentAlignment: "true",
		Representations:     []Representation{},
	}
	period.AdaptationSets = append(period.AdaptationSets, set)
	return &period.AdaptationSets[len(period.AdaptationSets)-1]
}

func FormatDuration(dur time.Duration) string {
	asString := dur.Round(100 * time.Millisecond).String()
	return "PT" + strings.ToUpper(asString)
}

func ComposeStaticMPD(
	mpdInfo Information,
	videoInfo info.VideoInformation,
) string {
	mediaDuration := FormatDuration(mpdInfo.MediaPresentationDuration)
	mpd := MPD{
		Xmlns:                     mpdNamespace,
		Profiles:                  mpdProfiles,
		Type:                      "static",
		MediaPresentationDuration: mediaDuration,
		ProgramInformation: &ProgramInformation{
			Title:  videoInfo.Title,
			Source: urlutil.BuildVideoLiveURL(videoInfo.ID),
		},
		Periods: []Period{{Duration: mediaDuration}},
	}

	period := &mpd.Periods[0]
	for _, stream := range videoInfo.AudioStreams {
		set := period.getOrCreateAdaptationSet(stream.MimeType)
		set.Representations = append(
			set.Representations,
			Representation{
				ID:                stream.Itag,
				Codecs:            stream.Codecs,
				AudioSamplingRate: &stream.AudioSamplingRate,
				BaseURL:           mpdInfo.RepresentationBaseURL,
				SegmentTemplate:   *mpdInfo.SegmentTemplate,
			},
		)
	}
	for _, stream := range videoInfo.VideoStreams {
		set := period.getOrCreateAdaptationSet(stream.MimeType)
		set.Representations = append(
			set.Representations,
			Representation{
				ID:              stream.Itag,
				Codecs:          stream.Codecs,
				Width:           &stream.Width,
				Height:          &stream.Height,
				FrameRate:       &stream.FrameRate,
				BaseURL:         mpdInfo.RepresentationBaseURL,
				SegmentTemplate: *mpdInfo.SegmentTemplate,
			},
		)
	}

	output, err := xml.MarshalIndent(mpd, " ", " ")
	if err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%s%s\n", xml.Header, string(output))
}
