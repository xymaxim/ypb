package info

import (
	"time"
)

type VideoInformation struct {
	ID              string
	Title           string
	Channel         string
	SegmentDuration time.Duration
	AudioStreams    []AudioStream
	VideoStreams    []VideoStream
}

func (i VideoInformation) BestVideo() *VideoStream {
	if len(i.VideoStreams) == 0 {
		return nil
	}

	best := i.VideoStreams[0]
	for _, s := range i.VideoStreams[1:] {
		if s.Height > best.Height {
			best = s
		} else if s.Height == best.Height && s.FrameRate > best.FrameRate {
			best = s
		}
	}

	return &best
}

type CommonStream struct {
	BaseURL  string
	Codecs   string
	Itag     string
	MimeType string
}

type AudioStream struct {
	CommonStream
	AudioSamplingRate int
}

type VideoStream struct {
	CommonStream
	Width     int
	Height    int
	FrameRate int
}
