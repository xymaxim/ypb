package playback_test

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xymaxim/ypb/playback"
	"github.com/xymaxim/ypb/segment"
)

type staticPlayback struct {
	playback.Playbacker

	metadata map[playback.SequenceNumber]segment.Metadata
	itag     string
}

func newStaticPlayback(metadata map[playback.SequenceNumber]segment.Metadata) *staticPlayback {
	return &staticPlayback{metadata: metadata, itag: "999"}
}

func (p *staticPlayback) FetchSegmentMetadata(
	_ string,
	sq playback.SequenceNumber,
) (*segment.Metadata, error) {
	m, ok := p.metadata[sq]
	if !ok {
		return nil, fmt.Errorf("no metadata for sq=%d", sq)
	}
	return &m, nil
}

func (p *staticPlayback) ProbeItag() string { return p.itag }

func (p *staticPlayback) LocateMoment(
	t time.Time,
	reference segment.Metadata,
	isEnd bool,
) (*playback.RewindMoment, error) {
	return playback.LocateMomentFor(p, t, reference, isEnd)
}

func readGapCaseMetadata(t *testing.T, path string) map[playback.SequenceNumber]segment.Metadata {
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

	mapping := make(map[int]segment.Metadata)
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

		mapping[sequenceNumber] = segment.Metadata{
			SequenceNumber:    sequenceNumber,
			IngestionWalltime: time.Unix(0, ingestionWalltimeUs*1e3).In(time.UTC),
			Duration:          2 * time.Second,
		}
	}

	return mapping
}

//nolint:tparallel
func TestLocateMomentFor_Synthetic(t *testing.T) {
	t.Parallel()

	// Synthetic test data
	metadataMapping := map[playback.SequenceNumber]segment.Metadata{
		0: {
			SequenceNumber:    0,
			IngestionWalltime: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
			Duration:          2 * time.Second,
		},
		1: {
			SequenceNumber:    1,
			IngestionWalltime: time.Date(2026, 1, 2, 10, 20, 32, 0, time.UTC),
			Duration:          2 * time.Second,
		},
	}

	pb := newStaticPlayback(metadataMapping)

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

	reference := metadataMapping[1]
	for _, tc := range testCases { //nolint:paralleltest
		t.Run(tc.name, func(t *testing.T) {
			moment, err := playback.LocateMomentFor(pb, tc.target, reference, tc.isEnd)
			require.NoError(t, err)
			if diff := cmp.Diff(tc.expected, moment); diff != "" {
				t.Fatal("Mismatch (- expected, + actual")
			}
		})
	}
}

//nolint:tparallel
func TestLocateMomentFor_GapCase1(t *testing.T) {
	t.Parallel()

	gapCase := readGapCaseMetadata(t, "testdata/gap-case-1.csv")
	pb := newStaticPlayback(gapCase)

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
				Metadata: segment.Metadata{SequenceNumber: 7959599},
				InGap:    false,
			},
		},
		{
			name:            "E1",
			targetSeconds:   1679788193.600278,
			referenceSeqNum: 7959630,
			isEnd:           true,
			expected: &playback.RewindMoment{
				Metadata: segment.Metadata{SequenceNumber: 7959599},
				InGap:    false,
			},
		},
		{
			name:            "S2",
			targetSeconds:   1679788196.600287,
			referenceSeqNum: 7959600,
			isEnd:           false,
			expected: &playback.RewindMoment{
				Metadata: segment.Metadata{SequenceNumber: 7959600},
				InGap:    false,
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
				Metadata: segment.Metadata{SequenceNumber: 7959601},
				InGap:    false,
			},
		},
		{
			name:            "S3_2",
			targetSeconds:   1679788198.599000,
			referenceSeqNum: 7959600,
			isEnd:           false,
			expected: &playback.RewindMoment{
				Metadata: segment.Metadata{SequenceNumber: 7959602},
				InGap:    false,
			},
		},
	}

	//nolint:paralleltest
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetTime := time.Unix(0, int64(tc.targetSeconds*1e9)).In(time.UTC)
			referenceTime := gapCase[tc.referenceSeqNum].IngestionWalltime
			actual, err := playback.LocateMomentFor(
				pb,
				targetTime,
				segment.Metadata{
					SequenceNumber:    tc.referenceSeqNum,
					IngestionWalltime: referenceTime,
					Duration:          2 * time.Second,
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
func TestLocateMomentFor_GapCase2(t *testing.T) {
	t.Parallel()

	gapCase := readGapCaseMetadata(t, "testdata/gap-case-2.csv")
	pb := newStaticPlayback(gapCase)

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
				Metadata: segment.Metadata{SequenceNumber: 7947333},
				InGap:    false,
			},
		},
		{
			name:            "S2",
			targetSeconds:   1679763599.262686,
			referenceSeqNum: 7947346,
			isEnd:           false,
			expected: &playback.RewindMoment{
				Metadata: segment.Metadata{SequenceNumber: 7947333},
				InGap:    false,
			},
		},
		{
			name:            "S3",
			targetSeconds:   1679763611.742391,
			referenceSeqNum: 7947346,
			isEnd:           false,
			expected: &playback.RewindMoment{
				Metadata: segment.Metadata{SequenceNumber: 7947335},
				InGap:    true,
			},
		},
		{
			name:            "E3",
			targetSeconds:   1679763611.742391,
			referenceSeqNum: 7947346,
			isEnd:           true,
			expected: &playback.RewindMoment{
				Metadata: segment.Metadata{SequenceNumber: 7947334},
				InGap:    true,
			},
		},
	}

	//nolint:paralleltest
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetTime := time.Unix(0, int64(tc.targetSeconds*1e9)).In(time.UTC)
			referenceTime := gapCase[tc.referenceSeqNum].IngestionWalltime
			actual, err := playback.LocateMomentFor(
				pb,
				targetTime,
				segment.Metadata{
					SequenceNumber:    tc.referenceSeqNum,
					IngestionWalltime: referenceTime,
					Duration:          2 * time.Second,
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
func TestLocateMomentFor_GapCase3(t *testing.T) {
	t.Parallel()

	gapCase := readGapCaseMetadata(t, "testdata/gap-case-3.csv")
	pb := newStaticPlayback(gapCase)

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
				Metadata: segment.Metadata{SequenceNumber: 7958102},
				InGap:    false,
			},
		},
		{
			name:            "S2",
			targetSeconds:   1679785201.449813,
			referenceSeqNum: 7958122,
			isEnd:           false,
			expected: &playback.RewindMoment{
				Metadata: segment.Metadata{SequenceNumber: 7958103},
				InGap:    false,
			},
		},
		{
			name:            "S3",
			targetSeconds:   1679785204.623643,
			referenceSeqNum: 7958122,
			isEnd:           false,
			expected: &playback.RewindMoment{
				Metadata: segment.Metadata{SequenceNumber: 7958104},
				InGap:    true,
			},
		},
		{
			name:            "E3",
			targetSeconds:   1679785204.623643,
			referenceSeqNum: 7958122,
			isEnd:           true,
			expected: &playback.RewindMoment{
				Metadata: segment.Metadata{SequenceNumber: 7958103},
				InGap:    true,
			},
		},
		{
			name:            "S4",
			targetSeconds:   1679785208.850441,
			referenceSeqNum: 7958122,
			isEnd:           false,
			expected: &playback.RewindMoment{
				Metadata: segment.Metadata{SequenceNumber: 7958104},
				InGap:    false,
			},
		},
		{
			name:            "S5",
			targetSeconds:   1679785208.903407,
			referenceSeqNum: 7958122,
			isEnd:           false,
			expected: &playback.RewindMoment{
				Metadata: segment.Metadata{SequenceNumber: 7958106},
				InGap:    false,
			},
		},
	}

	//nolint:paralleltest
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetTime := time.Unix(0, int64(tc.targetSeconds*1e9)).In(time.UTC)
			referenceTime := gapCase[tc.referenceSeqNum].IngestionWalltime
			actual, err := playback.LocateMomentFor(
				pb,
				targetTime,
				segment.Metadata{
					SequenceNumber:    tc.referenceSeqNum,
					IngestionWalltime: referenceTime,
					Duration:          2 * time.Second,
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
