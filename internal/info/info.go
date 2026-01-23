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
