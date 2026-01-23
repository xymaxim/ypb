package playback

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/xymaxim/ypb/internal/fetchers"
	"github.com/xymaxim/ypb/internal/playback/info"
	"github.com/xymaxim/ypb/internal/segment"
	"github.com/xymaxim/ypb/internal/urlutil"
)

type SequenceNumber = int

type Playback struct {
	BaseURLs map[string]string
	Client   *http.Client
	Fetcher  fetchers.Fetcher
	Info     info.VideoInformation
}

func NewPlayback(videoID string, fetcher fetchers.Fetcher, client *http.Client) (*Playback, error) {
	information, _, err := fetcher.FetchInfo()
	if err != nil {
		return nil, fmt.Errorf("fetching video info for playback: %w", err)
	}
	baseURLs := make(map[string]string)
	for _, s := range information.AudioStreams {
		baseURLs[s.Itag] = s.BaseURL
	}
	for _, s := range information.VideoStreams {
		baseURLs[s.Itag] = s.BaseURL
	}

	var pb *Playback
	if client == nil {
		client = NewClient(pb)
	}
	pb = &Playback{
		BaseURLs: baseURLs,
		Client:   client,
		Fetcher:  fetcher,
		Info:     *information,
	}

	return pb, nil
}

func (pb *Playback) RefreshBaseURLs() error {
	slog.Info("refreshing base URLs")
	baseURLs, err := pb.Fetcher.FetchBaseURLs()
	if err != nil {
		return fmt.Errorf("fetching base URLs: %w", err)
	}

	pb.BaseURLs = baseURLs

	return nil
}

func (pb *Playback) RequestHeadSeqNum() (int, error) {
	baseURL := pb.BaseURLs[pb.GetReferenceItag()]
	resp, err := pb.Client.Head(baseURL)
	if err != nil {
		return -1, fmt.Errorf("doing request: %w", err)
	}
	defer resp.Body.Close()

	seqNumRaw := resp.Header.Get("X-Head-Seqnum")
	if seqNumRaw == "" {
		return -1, errors.New("missing 'X-Head-Seqnum' header")
	}

	result, err := strconv.Atoi(seqNumRaw)
	if err != nil {
		return -1, fmt.Errorf("converting head sequence number: %w", err)
	}

	return result, nil
}

func (pb *Playback) DownloadSegment(itag string, sq SequenceNumber) ([]byte, error) {
	return pb.downloadSegmentPartial(itag, sq, 0)
}

func (pb *Playback) FetchSegmentMetadata(
	itag string,
	sq SequenceNumber,
) (*segment.Metadata, error) {
	b, err := pb.downloadSegmentPartial(itag, sq, segment.MetadataLength)
	if err != nil {
		return nil, fmt.Errorf("downloading segment sq=%d metadata: %w", sq, err)
	}

	sm, err := segment.ParseMetadata(b)
	if err != nil {
		return nil, fmt.Errorf("parsing metadata: %w", err)
	}

	return sm, nil
}

// func (pb *Playback) ResolveInterval(start, end MomentValue) {

// }

func (pb *Playback) GetReferenceItag() string {
	return pb.Info.VideoStreams[0].Itag
}

func (pb *Playback) downloadSegmentPartial(
	itag string,
	sq SequenceNumber,
	length int64,
) ([]byte, error) {
	baseURL := pb.BaseURLs[itag]
	if baseURL == "" {
		return nil, fmt.Errorf("missing base URL for itag '%s'", itag)
	}
	u, err := urlutil.BuildSegmentURL(baseURL, sq)
	if err != nil {
		return nil, fmt.Errorf("building segment URL: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating new request: %w", err)
	}

	if length > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=0-%d", length-1))
	}
	resp, err := pb.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting segment: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusPartialContent:
		var body []byte
		var readErr error
		if resp.StatusCode == http.StatusPartialContent {
			reader := &io.LimitedReader{R: resp.Body, N: length}
			body, readErr = io.ReadAll(reader)
		} else {
			body, readErr = io.ReadAll(resp.Body)
		}
		return body, readErr
	default:
		return nil, fmt.Errorf("got unexpected status: %s", resp.Status)
	}
}
