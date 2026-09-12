package fetchers //nolint:testpackage

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xymaxim/ypb/info"
)

func videoStreams(itags ...string) []info.VideoStream {
	streams := make([]info.VideoStream, 0, len(itags))
	for _, itag := range itags {
		streams = append(streams, info.VideoStream{
			CommonStream: info.CommonStream{Itag: itag},
		})
	}
	return streams
}

func TestResolveVideoItag(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name         string
		formatID     string
		videoStreams []info.VideoStream
		expected     string
	}{
		{
			name:     "single video format",
			formatID: "137",
			videoStreams: videoStreams(
				"136",
				"137",
			),
			expected: "137",
		},
		{
			name:     "audio before video",
			formatID: "140+137",
			videoStreams: videoStreams(
				"136",
				"137",
			),
			expected: "137",
		},
		{
			name:     "video before audio",
			formatID: "137+140",
			videoStreams: videoStreams(
				"136",
				"137",
			),
			expected: "137",
		},
		{
			name:     "audio only",
			formatID: "140",
			videoStreams: videoStreams(
				"136",
				"137",
			),
			expected: "",
		},
		{
			name:     "muxed format",
			formatID: "18",
			videoStreams: videoStreams(
				"18",
			),
			expected: "18",
		},
		{
			name:         "empty format id",
			formatID:     "",
			videoStreams: videoStreams("137"),
			expected:     "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(
				t,
				tc.expected,
				resolveVideoItag(tc.formatID, tc.videoStreams),
			)
		})
	}
}
