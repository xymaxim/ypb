package playback_test

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xymaxim/ypb/internal/playback"
	"github.com/xymaxim/ypb/internal/segment"
	"github.com/xymaxim/ypb/internal/testutil"
	"github.com/xymaxim/ypb/internal/urlutil"
)

type TestSegmentMetadata struct {
	SequenceNumber    int
	IngestionWalltime time.Time
	Duration          time.Duration
}

func readGapCaseMetadata(t *testing.T, path string) map[int]*TestSegmentMetadata {
	t.Helper()

	f, err := os.Open(path) // #nosec G304
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	reader := csv.NewReader(f)

	_, err = reader.Read()
	if err != nil && errors.Is(err, io.EOF) {
		t.Fatal(err)
	}

	mapping := make(map[int]*TestSegmentMetadata)
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		sequenceNumber, err := strconv.Atoi(record[0])
		if err != nil {
			t.Fatal(err)
		}
		ingestionWalltimeUs, err := strconv.ParseInt(record[1], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		duration, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			t.Fatal(err)
		}

		mapping[sequenceNumber] = &TestSegmentMetadata{
			SequenceNumber:    sequenceNumber,
			IngestionWalltime: time.Unix(0, ingestionWalltimeUs*1e3).In(time.UTC),
			Duration:          time.Duration(duration * float64(time.Second)),
		}
	}

	return mapping
}

func makeGapCaseHandler(
	t *testing.T,
	gc map[int]*TestSegmentMetadata,
) func(w http.ResponseWriter, r *http.Request) {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		sq, err := strconv.Atoi(urlutil.ExtractParameter(r.URL.RawPath, "sq"))
		if err != nil {
			t.Fatalf("parsing sq from request URL: %v", err)
		}
		testMetadata := gc[sq]
		if testMetadata == nil {
			t.Fatalf("no test metadata for sq=%d", sq)
		}
		w.Write(
			generateMetadataBytes(
				t,
				testMetadata.SequenceNumber,
				testMetadata.IngestionWalltime,
			),
		)
	}
}

func generateMetadataBytes(t *testing.T, sq int, ingestionWalltime time.Time) []byte {
	t.Helper()
	return fmt.Appendf(
		nil,
		`Sequence-Number: %d
Ingestion-Walltime-Us: %d`,
		sq,
		ingestionWalltime.UnixMicro(),
	)
}

//nolint:tparallel
func TestPlayback_LocateMoment_GapCase1(t *testing.T) {
	t.Parallel()

	gapCase := readGapCaseMetadata(t, "testdata/gap-case-1.csv")

	ts := httptest.NewServer(http.HandlerFunc(makeGapCaseHandler(t, gapCase)))
	defer ts.Close()

	fetcher := &testutil.MockFetcher{VideoID: testutil.TestVideoID}
	pb, _ := playback.NewPlayback(testutil.TestVideoID, fetcher, testutil.NewClient(ts.URL))

	testCases := []struct {
		name            string
		targetSeconds   float64
		referenceSeqNum int
		isEnd           bool
		expected        *playback.RewindMoment
	}{
		{
			name:            "S1",
			targetSeconds:   1679788193.600278,
			referenceSeqNum: 7959630,
			isEnd:           false,
			expected: &playback.RewindMoment{
				SequenceNumber: 7959599,
				InGap:          false,
			},
		},
		{
			name:            "E1",
			targetSeconds:   1679788193.600278,
			referenceSeqNum: 7959630,
			isEnd:           true,
			expected: &playback.RewindMoment{
				SequenceNumber: 7959599,
				InGap:          false,
			},
		},
		{
			name:            "S2",
			targetSeconds:   1679788196.600287,
			referenceSeqNum: 7959600,
			isEnd:           false,
			expected: &playback.RewindMoment{
				SequenceNumber: 7959600,
				InGap:          false,
			},
		},
		// For S3 cases, two segments are possibly formally valid,
		// depending on the chosen reference.
		{
			name:            "S3_1",
			targetSeconds:   1679788198.599000,
			referenceSeqNum: 7959601,
			isEnd:           false,
			expected: &playback.RewindMoment{
				SequenceNumber: 7959601,
				InGap:          false,
			},
		},
		{
			name:            "S3_2",
			targetSeconds:   1679788198.599000,
			referenceSeqNum: 7959600,
			isEnd:           false,
			expected: &playback.RewindMoment{
				SequenceNumber: 7959602,
				InGap:          false,
			},
		},
	}

	//nolint:paralleltest
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetTime := time.Unix(0, int64(tc.targetSeconds*1e9)).In(time.UTC)
			referenceTime := gapCase[tc.referenceSeqNum].IngestionWalltime
			actual, err := pb.LocateMoment(
				targetTime,
				segment.Metadata{
					SequenceNumber:    tc.referenceSeqNum,
					IngestionWalltime: referenceTime,
				},
				tc.isEnd,
			)
			require.NoError(t, err)
			assert.Equal(t, tc.expected.SequenceNumber, actual.SequenceNumber)
			assert.Equal(
				t,
				gapCase[tc.expected.SequenceNumber].IngestionWalltime,
				actual.Time,
			)
			assert.Equal(t, tc.expected.InGap, actual.InGap)
		})
	}
}

//nolint:tparallel
func TestPlayback_LocateMoment_GapCase2(t *testing.T) {
	t.Parallel()

	gapCase := readGapCaseMetadata(t, "testdata/gap-case-2.csv")

	ts := httptest.NewServer(http.HandlerFunc(makeGapCaseHandler(t, gapCase)))
	defer ts.Close()

	fetcher := &testutil.MockFetcher{VideoID: testutil.TestVideoID}
	pb, _ := playback.NewPlayback(testutil.TestVideoID, fetcher, testutil.NewClient(ts.URL))

	testCases := []struct {
		name            string
		targetSeconds   float64
		referenceSeqNum int
		isEnd           bool
		expected        *playback.RewindMoment
	}{
		{
			name:            "S1",
			targetSeconds:   1679763599.262686,
			referenceSeqNum: 7947346,
			isEnd:           false,
			expected: &playback.RewindMoment{
				SequenceNumber: 7947333,
				InGap:          false,
			},
		},
		{
			name:            "S2",
			targetSeconds:   1679763599.262686,
			referenceSeqNum: 7947346,
			isEnd:           false,
			expected: &playback.RewindMoment{
				SequenceNumber: 7947333,
				InGap:          false,
			},
		},
		{
			name:            "S3",
			targetSeconds:   1679763611.742391,
			referenceSeqNum: 7947346,
			isEnd:           false,
			expected: &playback.RewindMoment{
				SequenceNumber: 7947335,
				InGap:          true,
			},
		},
		{
			name:            "E3",
			targetSeconds:   1679763611.742391,
			referenceSeqNum: 7947346,
			isEnd:           true,
			expected: &playback.RewindMoment{
				SequenceNumber: 7947334,
				InGap:          true,
			},
		},
	}

	//nolint:paralleltest
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetTime := time.Unix(0, int64(tc.targetSeconds*1e9)).In(time.UTC)
			referenceTime := gapCase[tc.referenceSeqNum].IngestionWalltime
			actual, err := pb.LocateMoment(
				targetTime,
				segment.Metadata{
					SequenceNumber:    tc.referenceSeqNum,
					IngestionWalltime: referenceTime,
				},
				tc.isEnd,
			)
			require.NoError(t, err)
			assert.Equal(t, tc.expected.SequenceNumber, actual.SequenceNumber)
			assert.Equal(
				t,
				gapCase[tc.expected.SequenceNumber].IngestionWalltime,
				actual.Time,
			)
			assert.Equal(t, tc.expected.InGap, actual.InGap)
		})
	}
}

//nolint:tparallel
func TestPlayback_LocateMoment_GapCase3(t *testing.T) {
	t.Parallel()

	gapCase := readGapCaseMetadata(t, "testdata/gap-case-3.csv")

	ts := httptest.NewServer(http.HandlerFunc(makeGapCaseHandler(t, gapCase)))
	defer ts.Close()

	fetcher := &testutil.MockFetcher{VideoID: testutil.TestVideoID}
	pb, _ := playback.NewPlayback(testutil.TestVideoID, fetcher, testutil.NewClient(ts.URL))

	testCases := []struct {
		name            string
		targetSeconds   float64
		referenceSeqNum int
		isEnd           bool
		expected        *playback.RewindMoment
	}{
		{
			name:            "S1",
			targetSeconds:   1679785199.451019,
			referenceSeqNum: 7958122,
			isEnd:           false,
			expected: &playback.RewindMoment{
				SequenceNumber: 7958102,
				InGap:          false,
			},
		},
		{
			name:            "S2",
			targetSeconds:   1679785201.449813,
			referenceSeqNum: 7958122,
			isEnd:           false,
			expected: &playback.RewindMoment{
				SequenceNumber: 7958103,
				InGap:          false,
			},
		},
		{
			name:            "S3",
			targetSeconds:   1679785204.623643,
			referenceSeqNum: 7958122,
			isEnd:           false,
			expected: &playback.RewindMoment{
				SequenceNumber: 7958104,
				InGap:          true,
			},
		},
		{
			name:            "E3",
			targetSeconds:   1679785204.623643,
			referenceSeqNum: 7958122,
			isEnd:           true,
			expected: &playback.RewindMoment{
				SequenceNumber: 7958103,
				InGap:          true,
			},
		},
		{
			name:            "S4",
			targetSeconds:   1679785208.850441,
			referenceSeqNum: 7958122,
			isEnd:           false,
			expected: &playback.RewindMoment{
				SequenceNumber: 7958104,
				InGap:          false,
			},
		},
		{
			name:            "S5",
			targetSeconds:   1679785208.903407,
			referenceSeqNum: 7958122,
			isEnd:           false,
			expected: &playback.RewindMoment{
				SequenceNumber: 7958106,
				InGap:          false,
			},
		},
	}

	//nolint:paralleltest
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetTime := time.Unix(0, int64(tc.targetSeconds*1e9)).In(time.UTC)
			referenceTime := gapCase[tc.referenceSeqNum].IngestionWalltime
			actual, err := pb.LocateMoment(
				targetTime,
				segment.Metadata{
					SequenceNumber:    tc.referenceSeqNum,
					IngestionWalltime: referenceTime,
				},
				tc.isEnd,
			)
			require.NoError(t, err)
			assert.Equal(t, tc.expected.SequenceNumber, actual.SequenceNumber)
			assert.Equal(
				t,
				gapCase[tc.expected.SequenceNumber].IngestionWalltime,
				actual.Time,
			)
			assert.Equal(t, tc.expected.InGap, actual.InGap)
		})
	}
}
