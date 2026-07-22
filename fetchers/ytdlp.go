package fetchers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/xymaxim/ypb/info"
	"github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/internal/urlutil"
)

type YtdlpFetcher struct {
	VideoID      string
	Runner       exec.Runner
	YtdlpOptions []string
	OnPrint      func([]byte)
}

type YtdlpAdditionals struct {
	UserAgent string
}

type jsonDump struct {
	Title            string   `json:"title"`
	ChannelID        string   `json:"channel_id"`
	ChannelTitle     string   `json:"channel"`
	ReleaseTimestamp int64    `json:"release_timestamp"`
	Formats          []format `json:"formats"`
}

type format struct {
	URL               string            `json:"url"`
	FormatID          string            `json:"format_id"`
	TargetDuration    int               `json:"target_duration"`
	AudioCodec        string            `json:"acodec"`
	VideoCodec        string            `json:"vcodec"`
	AudioSamplingRate *int              `json:"asr"`
	Width             *int              `json:"width"`
	Height            *int              `json:"height"`
	FrameRate         *int              `json:"fps"`
	HTTPHeaders       map[string]string `json:"http_headers"`
}

func (fetcher *YtdlpFetcher) FetchInfo(
	ctx context.Context,
) (*info.VideoInformation, Additionals, error) {
	out, err := fetcher.runDumpJSON(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("dumping video info: %w", err)
	}

	var dump jsonDump
	if err := json.Unmarshal([]byte(out), &dump); err != nil {
		return nil, nil, fmt.Errorf("parsing info dump: %w", err)
	}

	audioStreams := []info.AudioStream{}
	videoStreams := []info.VideoStream{}
	for _, f := range dump.Formats {
		mimeTypeRaw := urlutil.ExtractParameter(f.URL, "mime")
		if mimeTypeRaw == "" {
			return nil, nil, fmt.Errorf(
				"missing mime type parameter in base URL: %s",
				f.URL,
			)
		}
		mimeType, err := url.PathUnescape(mimeTypeRaw)
		if err != nil {
			return nil, nil, fmt.Errorf("unescaping mime type: %w", err)
		}
		common := info.CommonStream{
			BaseURL:  f.URL,
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

	segmentDuration := time.Duration(float64(dump.Formats[0].TargetDuration)) * time.Second
	information := &info.VideoInformation{
		ID:              fetcher.VideoID,
		Title:           dump.Title,
		ChannelID:       dump.ChannelID,
		ChannelTitle:    dump.ChannelTitle,
		ActualStartTime: time.Unix(dump.ReleaseTimestamp, 0).UTC(),
		SegmentDuration: segmentDuration,
		AudioStreams:    audioStreams,
		VideoStreams:    videoStreams,
	}

	additionals := YtdlpAdditionals{
		UserAgent: dump.Formats[0].HTTPHeaders["User-Agent"],
	}

	return information, additionals, nil
}

func (fetcher *YtdlpFetcher) FetchBaseURLs(ctx context.Context) (map[string]string, error) {
	out, err := fetcher.runDumpJSON(ctx)
	if err != nil {
		return nil, fmt.Errorf("dumping video info: %w", err)
	}

	var dump struct {
		Formats []struct {
			FormatID string `json:"format_id"`
			URL      string `json:"url"`
		} `json:"formats"`
	}
	if err := json.Unmarshal([]byte(out), &dump); err != nil {
		return nil, fmt.Errorf("parsing info dump: %w", err)
	}

	baseURLs := make(map[string]string)
	for _, f := range dump.Formats {
		baseURLs[f.FormatID] = f.URL
	}

	return baseURLs, nil
}

func (fetcher *YtdlpFetcher) runDumpJSON(ctx context.Context) (string, error) {
	printCallback := func(chunk []byte) {
		fetcher.Runner.(*exec.CommandRunner).PrintCallback(chunk)
		if fetcher.OnPrint != nil {
			fetcher.OnPrint(chunk)
		}
	}

	tempFile, err := os.CreateTemp("", "ypb-*")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer tempFile.Close()

	// yt-dlp appends a suffix to the output filename for some reason
	jsonPath := tempFile.Name() + ".info.json"
	defer os.Remove(tempFile.Name())
	defer os.Remove(jsonPath)

	args := []string{
		"--live-from-start",
		"--skip-download",
		"--write-info-json",
		"-o", "infojson:" + tempFile.Name(),
	}
	args = append(args, fetcher.YtdlpOptions...)
	args = append(args, fetcher.VideoID)

	_, err = fetcher.Runner.RunWith(
		ctx,
		[]exec.Option{
			exec.WithCallbacks(printCallback, printCallback),
		},
		args...,
	)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(jsonPath)
	if err != nil {
		return "", fmt.Errorf("reading output file: %w", err)
	}

	return string(content), nil
}
