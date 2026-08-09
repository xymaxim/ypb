package testutil

import (
	"time"

	"github.com/xymaxim/ypb/info"
	"github.com/xymaxim/ypb/internal/mpd"
)

func SampleVideoInfo() info.VideoInformation {
	return info.VideoInformation{
		ID:    "abcdefgh123",
		Title: "Sample stream",
		AudioStreams: []info.AudioStream{
			{
				CommonStream: info.CommonStream{
					Itag:     "140",
					Codecs:   "mp4a.40.2",
					MimeType: "audio/mp4",
					Tbr:      128,
				},
				AudioSamplingRate: 44100,
			},
		},
		VideoStreams: []info.VideoStream{
			{
				CommonStream: info.CommonStream{
					Itag:     "160",
					Codecs:   "avc1.42c00b",
					MimeType: "video/mp4",
					Tbr:      100,
				},
				Width: 256, Height: 144, FrameRate: 15,
			},
			{
				CommonStream: info.CommonStream{
					Itag:     "137",
					Codecs:   "avc1.640028",
					MimeType: "video/mp4",
					Tbr:      1500,
				},
				Width: 1920, Height: 1080, FrameRate: 30,
			},
			{
				CommonStream: info.CommonStream{
					Itag:     "394",
					Codecs:   "av01.0.00M.08",
					MimeType: "video/mp4",
					Tbr:      80,
				},
				Width: 256, Height: 144, FrameRate: 30,
			},
			{
				CommonStream: info.CommonStream{
					Itag:     "399",
					Codecs:   "av01.0.08M.08",
					MimeType: "video/mp4",
					Tbr:      500,
				},
				Width: 1920, Height: 1080, FrameRate: 30,
			},
			{
				CommonStream: info.CommonStream{
					Itag:     "248",
					Codecs:   "vp9",
					MimeType: "video/webm",
					Tbr:      1000,
				},
				Width: 1920, Height: 1080, FrameRate: 30,
			},
		},
	}
}

func SampleCommonOptions() mpd.CommonOptions {
	return mpd.CommonOptions{
		BaseURL:         "http://localhost:9000",
		StartNumber:     123,
		SegmentDuration: 5 * time.Second,
		AudioPTS:        0,
		VideoPTS:        0,
	}
}
