package testutil

import (
	"time"

	"github.com/xymaxim/ypb/internal/playback/fetchers"
	"github.com/xymaxim/ypb/internal/playback/info"
)

type MockFetcher struct {
	VideoID string
}

var (
	TestVideoID  = "abcdefgh123"
	TestBaseURLs = map[string]string{
		"136": "https://test/videoplayback/itag/136/mime/video%2Fmp4/dur/2.000/",
		"137": "https://test/videoplayback/itag/137/mime/video%2Fmp4/dur/2.000/",
		"140": "https://test/videoplayback/itag/140/mime/audio%2Fmp4/dur/2.000/",
	}
)

func (f *MockFetcher) FetchInfo() (*info.VideoInformation, fetchers.Additionals, error) {
	return &info.VideoInformation{
		ID:              TestVideoID,
		Title:           "Test tile",
		Channel:         "Test channel",
		SegmentDuration: 2 * time.Second,
		AudioStreams: []info.AudioStream{
			{
				CommonStream: info.CommonStream{
					BaseURL:  TestBaseURLs["140"],
					Codecs:   "mp4a.40.2",
					Itag:     "140",
					MimeType: "audio/mp4",
				},
				AudioSamplingRate: 44100,
			},
		},
		VideoStreams: []info.VideoStream{
			{
				CommonStream: info.CommonStream{
					BaseURL:  TestBaseURLs["136"],
					Codecs:   "avc1.4d401f",
					Itag:     "136",
					MimeType: "video/mp4",
				},
				Width:     1280,
				Height:    720,
				FrameRate: 30,
			},
			{
				CommonStream: info.CommonStream{
					BaseURL:  TestBaseURLs["137"],
					Codecs:   "avc1.640028",
					Itag:     "137",
					MimeType: "video/mp4",
				},
				Width:     1920,
				Height:    1080,
				FrameRate: 30,
			},
		},
	}, nil, nil
}

func (f *MockFetcher) FetchBaseURLs() (map[string]string, error) {
	return map[string]string{
		"136": "https://test/videoplayback/itag/136/mime/video%2Fmp4/dur/2.000/new",
		"137": "https://test/videoplayback/itag/137/mime/video%2Fmp4/dur/2.000/new",
		"140": "https://test/videoplayback/itag/140/mime/audio%2Fmp4/dur/2.000/new",
	}, nil
}
