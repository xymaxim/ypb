package playback_test

import (
	"encoding/csv"
	"errors"
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
	"github.com/xymaxim/ypb/internal/playback/segment"
	"github.com/xymaxim/ypb/internal/testutil"
)

func makeGapCaseHandler(
	t *testing.T,
	data map[playback.SequenceNumber]*segment.Metadata,
) func(w http.ResponseWriter, r *http.Request) {
	t.Helper()
	return testutil.MakeSegmentMetadataHandler(t, data)
}

func readGapCaseMetadata(t *testing.T, path string) map[playback.SequenceNumber]*segment.Metadata {
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

	mapping := make(map[int]*segment.Metadata)
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

		mapping[sequenceNumber] = &segment.Metadata{
			SequenceNumber:    sequenceNumber,
			IngestionWalltime: time.Unix(0, ingestionWalltimeUs*1e3).In(time.UTC),
		}
	}

	return mapping
}

//nolint:tparallel
func TestPlayback_LocateMoment_Synthetic(t *testing.T) {
	t.Parallel()

	// Synthetic test data
	metadataMapping := map[playback.SequenceNumber]*segment.Metadata{
		0: {
			SequenceNumber:    0,
			IngestionWalltime: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
		},
		1: {
			SequenceNumber:    1,
			IngestionWalltime: time.Date(2026, 1, 2, 10, 20, 32, 0, time.UTC),
		},
	}

	// Setup
	ts := httptest.NewServer(
		http.HandlerFunc(
			testutil.MakeSegmentMetadataHandler(
				t,
				metadataMapping,
			),
		),
	)
	defer ts.Close()

	pb, err := playback.NewPlayback(
		testutil.TestVideoID,
		&testutil.MockFetcher{
			VideoID: testutil.TestVideoID,
		},
		testutil.NewClient(ts.URL),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Test cases
	testCases := []struct {
		name     string
		target   time.Time
		isEnd    bool
		expected *playback.RewindMoment
	}{
		{
			name:   "start moment at start edge",
			target: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
			isEnd:  false,
			expected: &playback.RewindMoment{
				Metadata:   metadataMapping[0],
				ActualTime: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
				TargetTime: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
				InGap:      false,
			},
		},
		{
			name:   "start moment near start edge",
			target: time.Date(2026, 1, 2, 10, 20, 30, 500_000_000, time.UTC),
			isEnd:  false,
			expected: &playback.RewindMoment{
				Metadata:   metadataMapping[0],
				ActualTime: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
				TargetTime: time.Date(
					2026,
					1,
					2,
					10,
					20,
					30,
					500_000_000,
					time.UTC,
				),
				InGap: false,
			},
		},
		{
			name:   "end moment near start edge",
			target: time.Date(2026, 1, 2, 10, 20, 30, 500000000, time.UTC),
			isEnd:  true,
			expected: &playback.RewindMoment{
				Metadata:   metadataMapping[0],
				ActualTime: time.Date(2026, 1, 2, 10, 20, 32, 0, time.UTC),
				TargetTime: time.Date(
					2026,
					1,
					2,
					10,
					20,
					30,
					500_000_000,
					time.UTC,
				),
				InGap: false,
			},
		},
	}

	reference := *metadataMapping[1]
	for _, tc := range testCases { //nolint:paralleltest
		t.Run(tc.name, func(t *testing.T) {
			rm, err := pb.LocateMoment(tc.target, reference, tc.isEnd)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, rm)
		})
	}
}

//nolint:tparallel
func TestPlayback_LocateMoment_GapCase1(t *testing.T) {
	t.Parallel()

	// Read test data
	gapCase := readGapCaseMetadata(t, "testdata/gap-case-1.csv")

	// Setup
	ts := httptest.NewServer(http.HandlerFunc(makeGapCaseHandler(t, gapCase)))
	defer ts.Close()

	fetcher := &testutil.MockFetcher{VideoID: testutil.TestVideoID}
	pb, _ := playback.NewPlayback(testutil.TestVideoID, fetcher, testutil.NewClient(ts.URL))

	// Test cases
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
				Metadata: &segment.Metadata{
					SequenceNumber: 7959599,
				},
				InGap: false,
			},
		},
		{
			name:            "E1",
			targetSeconds:   1679788193.600278,
			referenceSeqNum: 7959630,
			isEnd:           true,
			expected: &playback.RewindMoment{
				Metadata: &segment.Metadata{
					SequenceNumber: 7959599,
				},
				InGap: false,
			},
		},
		{
			name:            "S2",
			targetSeconds:   1679788196.600287,
			referenceSeqNum: 7959600,
			isEnd:           false,
			expected: &playback.RewindMoment{
				Metadata: &segment.Metadata{
					SequenceNumber: 7959600,
				},
				InGap: false,
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
				Metadata: &segment.Metadata{
					SequenceNumber: 7959601,
				},
				InGap: false,
			},
		},
		{
			name:            "S3_2",
			targetSeconds:   1679788198.599000,
			referenceSeqNum: 7959600,
			isEnd:           false,
			expected: &playback.RewindMoment{
				Metadata: &segment.Metadata{
					SequenceNumber: 7959602,
				},
				InGap: false,
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
			assert.Equal(
				t,
				tc.expected.Metadata.SequenceNumber,
				actual.Metadata.SequenceNumber,
			)
			assert.Equal(t, tc.expected.InGap, actual.InGap)
		})
	}
}

//nolint:tparallel
func TestPlayback_LocateMoment_GapCase2(t *testing.T) {
	t.Parallel()

	// Read test data
	gapCase := readGapCaseMetadata(t, "testdata/gap-case-2.csv")

	// Setup
	ts := httptest.NewServer(http.HandlerFunc(makeGapCaseHandler(t, gapCase)))
	defer ts.Close()

	fetcher := &testutil.MockFetcher{VideoID: testutil.TestVideoID}
	pb, _ := playback.NewPlayback(testutil.TestVideoID, fetcher, testutil.NewClient(ts.URL))

	// Test cases
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
				Metadata: &segment.Metadata{
					SequenceNumber: 7947333,
				},
				InGap: false,
			},
		},
		{
			name:            "S2",
			targetSeconds:   1679763599.262686,
			referenceSeqNum: 7947346,
			isEnd:           false,
			expected: &playback.RewindMoment{
				Metadata: &segment.Metadata{
					SequenceNumber: 7947333,
				},
				InGap: false,
			},
		},
		{
			name:            "S3",
			targetSeconds:   1679763611.742391,
			referenceSeqNum: 7947346,
			isEnd:           false,
			expected: &playback.RewindMoment{
				Metadata: &segment.Metadata{
					SequenceNumber: 7947335,
				},
				InGap: true,
			},
		},
		{
			name:            "E3",
			targetSeconds:   1679763611.742391,
			referenceSeqNum: 7947346,
			isEnd:           true,
			expected: &playback.RewindMoment{
				Metadata: &segment.Metadata{
					SequenceNumber: 7947334,
				},
				InGap: true,
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
			assert.Equal(
				t,
				tc.expected.Metadata.SequenceNumber,
				actual.Metadata.SequenceNumber,
			)
			assert.Equal(t, tc.expected.InGap, actual.InGap)
		})
	}
}

//nolint:tparallel
func TestPlayback_LocateMoment_GapCase3(t *testing.T) {
	t.Parallel()

	// Read test data
	gapCase := readGapCaseMetadata(t, "testdata/gap-case-3.csv")

	// Setup
	ts := httptest.NewServer(http.HandlerFunc(makeGapCaseHandler(t, gapCase)))
	defer ts.Close()

	fetcher := &testutil.MockFetcher{VideoID: testutil.TestVideoID}
	pb, _ := playback.NewPlayback(testutil.TestVideoID, fetcher, testutil.NewClient(ts.URL))

	// Test cases
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
				Metadata: &segment.Metadata{
					SequenceNumber: 7958102,
				},
				InGap: false,
			},
		},
		{
			name:            "S2",
			targetSeconds:   1679785201.449813,
			referenceSeqNum: 7958122,
			isEnd:           false,
			expected: &playback.RewindMoment{
				Metadata: &segment.Metadata{
					SequenceNumber: 7958103,
				},
				InGap: false,
			},
		},
		{
			name:            "S3",
			targetSeconds:   1679785204.623643,
			referenceSeqNum: 7958122,
			isEnd:           false,
			expected: &playback.RewindMoment{
				Metadata: &segment.Metadata{
					SequenceNumber: 7958104,
				},
				InGap: true,
			},
		},
		{
			name:            "E3",
			targetSeconds:   1679785204.623643,
			referenceSeqNum: 7958122,
			isEnd:           true,
			expected: &playback.RewindMoment{
				Metadata: &segment.Metadata{
					SequenceNumber: 7958103,
				},
				InGap: true,
			},
		},
		{
			name:            "S4",
			targetSeconds:   1679785208.850441,
			referenceSeqNum: 7958122,
			isEnd:           false,
			expected: &playback.RewindMoment{
				Metadata: &segment.Metadata{
					SequenceNumber: 7958104,
				},
				InGap: false,
			},
		},
		{
			name:            "S5",
			targetSeconds:   1679785208.903407,
			referenceSeqNum: 7958122,
			isEnd:           false,
			expected: &playback.RewindMoment{
				Metadata: &segment.Metadata{
					SequenceNumber: 7958106,
				},
				InGap: false,
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
			assert.Equal(
				t,
				tc.expected.Metadata.SequenceNumber,
				actual.Metadata.SequenceNumber,
			)
			assert.Equal(t, tc.expected.InGap, actual.InGap)
		})
	}
}
