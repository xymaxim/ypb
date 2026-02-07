package playback

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/xymaxim/ypb/internal/playback/fetchers"
	"github.com/xymaxim/ypb/internal/playback/info"
	"github.com/xymaxim/ypb/internal/playback/segment"
	"github.com/xymaxim/ypb/internal/urlutil"
)

type SequenceNumber = int

// SegmentMetadataFetchError wraps errors that occur when fetching segment metadata.
type SegmentMetadataFetchError struct {
	SequenceNumber SequenceNumber
	Err            error
}

func NewSegmentMetadataFetchError(sq SequenceNumber, err error) *SegmentMetadataFetchError {
	return &SegmentMetadataFetchError{SequenceNumber: sq, Err: err}
}

func (e *SegmentMetadataFetchError) Error() string {
	return fmt.Sprintf("fetching segment metadata (sq = %d): %v", e.SequenceNumber, e.Err)
}

func (e *SegmentMetadataFetchError) Unwrap() error {
	return e.Err
}

type Playbacker interface {
	BaseURLs() map[string]string
	DownloadSegment(itag string, sq SequenceNumber) ([]byte, error)
	FetchSegmentMetadata(itag string, sq SequenceNumber) (*segment.Metadata, error)
	Info() info.VideoInformation
	LocateMoment(time.Time, segment.Metadata, bool) (*RewindMoment, error)
	ProbeItag() string
	RefreshBaseURLs() error
	RequestHeadSeqNum() (int, error)
	StreamSegment(itag string, sq SequenceNumber, w io.Writer) error
}

var _ Playbacker = (*Playback)(nil)

type Playback struct {
	baseURLs map[string]string
	client   *http.Client
	fetcher  fetchers.Fetcher
	info     info.VideoInformation
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

	pb := &Playback{
		baseURLs: baseURLs,
		fetcher:  fetcher,
		info:     *information,
	}

	if client == nil {
		client = NewClient(pb).StandardClient()
	}
	pb.client = client

	return pb, nil
}

func (pb *Playback) BaseURLs() map[string]string {
	return pb.baseURLs
}

func (pb *Playback) Info() info.VideoInformation {
	return pb.info
}

func (pb *Playback) RefreshBaseURLs() error {
	slog.Debug("refreshing base URLs")
	baseURLs, err := pb.fetcher.FetchBaseURLs()
	if err != nil {
		return fmt.Errorf("fetching base URLs: %w", err)
	}

	pb.baseURLs = baseURLs

	return nil
}

func (pb *Playback) RequestHeadSeqNum() (int, error) {
	slog.Debug("requesting head sequence number")

	baseURL := pb.BaseURLs()[pb.ProbeItag()]
	resp, err := pb.client.Head(baseURL)
	if err != nil {
		return -1, fmt.Errorf("doing request: %w", err)
	}
	defer resp.Body.Close()

	seqNumRaw := resp.Header.Get("X-Head-Seqnum")
	if seqNumRaw == "" {
		return -1, errors.New("missing 'X-Head-Seqnum' header")
	}
	slog.Debug("got head sequence number", "sq", seqNumRaw)

	result, err := strconv.Atoi(seqNumRaw)
	if err != nil {
		return -1, fmt.Errorf("converting head sequence number: %w", err)
	}

	return result, nil
}

func (pb *Playback) ProbeItag() string {
	return pb.Info().VideoStreams[0].Itag
}

func (pb *Playback) StreamSegment(itag string, sq SequenceNumber, w io.Writer) error {
	return pb.streamSegmentPartial(itag, sq, 0, w)
}

func (pb *Playback) DownloadSegment(itag string, sq SequenceNumber) ([]byte, error) {
	var buf bytes.Buffer
	if err := pb.streamSegmentPartial(itag, sq, 0, &buf); err != nil {
		return nil, fmt.Errorf("streaming segment: %w", sq, err)
	}
	return buf.Bytes(), nil
}

func (pb *Playback) FetchSegmentMetadata(
	itag string,
	sq SequenceNumber,
) (*segment.Metadata, error) {
	var buf bytes.Buffer
	err := pb.streamSegmentPartial(itag, sq, segment.MetadataLength, &buf)
	if err != nil {
		return nil, fmt.Errorf("downloading segment metadata, sq=%d: %w", sq, err)
	}

	sm, err := segment.ParseMetadata(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("parsing metadata: %w", err)
	}

	return sm, nil
}

func (pb *Playback) streamSegmentPartial(
	itag string,
	sq SequenceNumber,
	length int64,
	w io.Writer,
) error {
	baseURL := pb.BaseURLs()[itag]
	if baseURL == "" {
		return fmt.Errorf("missing base URL for itag '%s'", itag)
	}
	u, err := urlutil.BuildSegmentURL(baseURL, strconv.Itoa(sq))
	if err != nil {
		return fmt.Errorf("building segment URL: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("creating new request: %w", err)
	}

	if length > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=0-%d", length-1))
	}
	resp, err := pb.client.Do(req)
	if err != nil {
		return fmt.Errorf("requesting segment: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusPartialContent:
		var copyErr error
		if resp.StatusCode == http.StatusPartialContent && length > 0 {
			reader := &io.LimitedReader{R: resp.Body, N: length}
			_, copyErr = io.Copy(w, reader)
		} else {
			_, copyErr = io.Copy(w, resp.Body)
		}
		return copyErr
	default:
		return fmt.Errorf("got unexpected status: %s", resp.Status)
	}
}
