package playback_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xymaxim/ypb/internal/playback"
	"github.com/xymaxim/ypb/internal/playback/segment"
	"github.com/xymaxim/ypb/internal/testutil"
)

func TestNewPlayback(t *testing.T) {
	t.Parallel()
	fetcher := &testutil.MockFetcher{VideoID: testutil.TestVideoID}
	pb, err := playback.NewPlayback(testutil.TestVideoID, fetcher, nil)
	require.NoError(t, err, "creating playback should not error")
	assert.Equal(t, testutil.TestBaseURLs, pb.BaseURLs())
}

func TestPlayback_RefreshBaseURLs(t *testing.T) {
	t.Parallel()
	fetcher := &testutil.MockFetcher{VideoID: testutil.TestVideoID}
	pb, _ := playback.NewPlayback(testutil.TestVideoID, fetcher, nil)
	require.NoError(t, pb.RefreshBaseURLs())
	assert.Equal(
		t,
		map[string]string{
			"136": "https://test/videoplayback/itag/136/mime/video%2Fmp4/dur/2.000/new",
			"137": "https://test/videoplayback/itag/137/mime/video%2Fmp4/dur/2.000/new",
			"140": "https://test/videoplayback/itag/140/mime/audio%2Fmp4/dur/2.000/new",
		},
		pb.BaseURLs(),
	)
}

func TestPlayback_RequestHeadSeqNum_Success(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodHead, r.Method)
			w.Header().Set("X-Head-Seqnum", "123")
		}),
	)
	defer ts.Close()

	fetcher := &testutil.MockFetcher{VideoID: testutil.TestVideoID}
	pb, _ := playback.NewPlayback(testutil.TestVideoID, fetcher, testutil.NewClient(ts.URL))

	actual, err := pb.RequestHeadSeqNum()
	require.NoError(t, err)
	assert.Equal(t, 123, actual)
}

func TestPlayback_RequestHeadSeqNum_MissingHeader(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(testutil.MakeDummyHandler())
	defer ts.Close()

	fetcher := &testutil.MockFetcher{VideoID: testutil.TestVideoID}
	pb, _ := playback.NewPlayback(testutil.TestVideoID, fetcher, testutil.NewClient(ts.URL))

	_, err := pb.RequestHeadSeqNum()
	if assert.Error(t, err) {
		assert.EqualError(t, err, "missing 'X-Head-Seqnum' header")
	}
}

func TestPlayback_DownloadSegment(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(
				t,
				"/videoplayback/itag/140/mime/audio%2Fmp4/dur/2.000/sq/123",
				r.URL.EscapedPath(),
			)
			w.Write([]byte("test"))
		}),
	)
	defer ts.Close()

	fetcher := &testutil.MockFetcher{VideoID: testutil.TestVideoID}
	pb, _ := playback.NewPlayback(testutil.TestVideoID, fetcher, testutil.NewClient(ts.URL))

	data, err := pb.DownloadSegment("140", 123)
	require.NoError(t, err)
	assert.Equal(t, []byte("test"), data)
}

func TestPlayback_DownloadSegment_UnknownItag(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(testutil.MakeDummyHandler())
	defer ts.Close()

	fetcher := &testutil.MockFetcher{VideoID: testutil.TestVideoID}
	pb, _ := playback.NewPlayback(testutil.TestVideoID, fetcher, testutil.NewClient(ts.URL))

	_, err := pb.DownloadSegment("unknown", 123)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestPlayback_FetchSegmentMetadata(t *testing.T) {
	t.Parallel()

	metadataBytes := `Sequence-Number: 7959120
Ingestion-Walltime-Us: 1679787234491176
Ingestion-Uncertainty-Us: 71
Target-Duration-Us: 2000000`
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(
				t,
				fmt.Sprintf("bytes=0-%d", segment.MetadataLength-1),
				r.Header.Get("Range"),
			)
			assert.Equal(
				t,
				"/videoplayback/itag/140/mime/audio%2Fmp4/dur/2.000/sq/123",
				r.URL.EscapedPath(),
			)
			w.Write([]byte(metadataBytes))
		}),
	)
	defer ts.Close()

	fetcher := &testutil.MockFetcher{VideoID: testutil.TestVideoID}
	pb, _ := playback.NewPlayback(testutil.TestVideoID, fetcher, testutil.NewClient(ts.URL))

	data, err := pb.FetchSegmentMetadata("140", 123)
	require.NoError(t, err)
	assert.Equal(
		t,
		&segment.Metadata{
			SequenceNumber:    7959120,
			IngestionWalltime: time.Unix(0, 1679787234491176*1e3).In(time.UTC),
			Duration:          2 * time.Second,
		},
		data,
	)
}
