package fetchers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/internal/playback/info"
	"github.com/xymaxim/ypb/internal/urlutil"
)

type YtdlpFetcher struct {
	VideoID string
	Runner  exec.Runner
}

type YtdlpAdditionals struct {
	UserAgent string
}

type jsonDump struct {
	Title   string   `json:"title"`
	Channel string   `json:"channel"`
	Formats []format `json:"formats"`
}

type format struct {
	FragmentBaseURL   string            `json:"fragment_base_url"`
	FormatID          string            `json:"format_id"`
	AudioCodec        string            `json:"acodec"`
	VideoCodec        string            `json:"vcodec"`
	AudioSamplingRate *int              `json:"asr"`
	Width             *int              `json:"width"`
	Height            *int              `json:"height"`
	FrameRate         *int              `json:"fps"`
	HTTPHeaders       map[string]string `json:"http_headers"`
}

func (fetcher *YtdlpFetcher) FetchInfo() (*info.VideoInformation, Additionals, error) {
	out, err := fetcher.runDumpJSON()
	if err != nil {
		return nil, nil, fmt.Errorf("dumping video info: %w", err)
	}

	var dump jsonDump
	if err := json.Unmarshal(out, &dump); err != nil {
		return nil, nil, fmt.Errorf("parsing info dump: %w", err)
	}

	audioStreams := []info.AudioStream{}
	videoStreams := []info.VideoStream{}
	for _, f := range dump.Formats {
		mimeTypeRaw := urlutil.ExtractParameter(f.FragmentBaseURL, "mime")
		if mimeTypeRaw == "" {
			return nil, nil, fmt.Errorf(
				"missing mime type parameter in base URL: %s",
				f.FragmentBaseURL,
			)
		}
		mimeType, err := url.PathUnescape(mimeTypeRaw)
		if err != nil {
			return nil, nil, fmt.Errorf("unescaping mime type: %w", err)
		}
		common := info.CommonStream{
			BaseURL:  f.FragmentBaseURL,
			Itag:     f.FormatID,
			MimeType: mimeType,
		}
		if f.VideoCodec == "none" {
			common.Codecs = f.AudioCodec
			audioStreams = append(
				audioStreams,
				info.AudioStream{
					CommonStream:      common,
					AudioSamplingRate: *f.AudioSamplingRate,
				},
			)
		} else {
			common.Codecs = f.VideoCodec
			videoStreams = append(
				videoStreams,
				info.VideoStream{
					CommonStream: common,
					Width:        *f.Width,
					Height:       *f.Height,
					FrameRate:    *f.FrameRate,
				},
			)
		}
	}

	someBaseURL := videoStreams[0].BaseURL
	segmentDurationRaw := urlutil.ExtractParameter(someBaseURL, "dur")
	if segmentDurationRaw == "" {
		return nil, nil, fmt.Errorf("no 'dur' parameter in base URL: %s", someBaseURL)
	}
	segmentDurationNumber, err := strconv.ParseFloat(segmentDurationRaw, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing segment duration: %w", err)
	}
	segmentDuration := time.Duration(segmentDurationNumber * float64(time.Second))

	information := &info.VideoInformation{
		ID:              fetcher.VideoID,
		Title:           dump.Title,
		Channel:         dump.Channel,
		SegmentDuration: segmentDuration,
		AudioStreams:    audioStreams,
		VideoStreams:    videoStreams,
	}
	additionals := YtdlpAdditionals{
		UserAgent: dump.Formats[0].HTTPHeaders["User-Agent"],
	}

	return information, additionals, nil
}

func (fetcher *YtdlpFetcher) FetchBaseURLs() (map[string]string, error) {
	out, err := fetcher.runDumpJSON()
	if err != nil {
		return nil, fmt.Errorf("dumping video info: %w", err)
	}

	var dump struct {
		Formats []struct {
			FormatID        string `json:"format_id"`
			FragmentBaseURL string `json:"fragment_base_url"`
		} `json:"formats"`
	}
	if err := json.Unmarshal(out, &dump); err != nil {
		return nil, fmt.Errorf("parsing info dump: %w", err)
	}

	baseURLs := make(map[string]string)
	for _, f := range dump.Formats {
		baseURLs[f.FormatID] = f.FragmentBaseURL
	}

	return baseURLs, nil
}

func (fetcher *YtdlpFetcher) runDumpJSON() ([]byte, error) {
	var outBuf bytes.Buffer

	_, err := fetcher.Runner.RunWith(
		[]exec.Option{
			exec.WithStdoutMode(exec.StreamRaw),
			exec.WithCallbacks(
				func(chunk []byte) { outBuf.Write(chunk) },
				fetcher.Runner.(*exec.CommandRunner).PrintCallback(),
			),
		},
		"--dump-json",
		"--live-from-start",
		fetcher.VideoID,
	)
	if err != nil {
		return nil, err
	}

	return outBuf.Bytes(), nil
}
