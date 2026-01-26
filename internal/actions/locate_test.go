package actions_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/xymaxim/ypb/internal/actions"
	"github.com/xymaxim/ypb/internal/input"
	"github.com/xymaxim/ypb/internal/playback"
	"github.com/xymaxim/ypb/internal/playback/segment"
	"github.com/xymaxim/ypb/internal/testutil"
)

type fakePlayback struct {
	*playback.Playback
	fakeMetadata testutil.MetadataMap
}

func newFakePlayback(data testutil.MetadataMap) *fakePlayback {
	return &fakePlayback{
		fakeMetadata: data,
	}
}

func (pb *fakePlayback) FetchSegmentMetadata(
	_ string,
	sq playback.SequenceNumber,
) (*segment.Metadata, error) {
	return pb.fakeMetadata[sq], nil
}

func (pb *fakePlayback) ProbeItag() string {
	return "100"
}

func (pb *fakePlayback) LocateMoment(
	t time.Time,
	_ segment.Metadata,
	_ bool,
) (*playback.RewindMoment, error) {
	return playback.NewRewindMoment(t, pb.fakeMetadata[0], false, false), nil
}

func TestLocateMoment(t *testing.T) {
	t.Parallel()
	fakeMetadata := testutil.MetadataMap{
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

	testCases := []struct {
		name     string
		value    input.MomentValue
		expected *playback.RewindMoment
	}{
		{
			name:  "time",
			value: time.Date(2026, 1, 2, 10, 20, 31, 0, time.UTC),
			expected: &playback.RewindMoment{
				Metadata:   fakeMetadata[0],
				ActualTime: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
				TargetTime: time.Date(2026, 1, 2, 10, 20, 31, 0, time.UTC),
				InGap:      false,
			},
		},
		{
			name:  "sequence number",
			value: 0,
			expected: &playback.RewindMoment{
				Metadata:   fakeMetadata[0],
				ActualTime: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
				TargetTime: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
				InGap:      false,
			},
		},
	}

	pb := newFakePlayback(fakeMetadata)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			moment, err := actions.LocateMoment(
				pb,
				tc.value,
				*fakeMetadata[1],
			)
			require.NoError(t, err)
			if diff := cmp.Diff(tc.expected, moment); diff != "" {
				t.Fatalf("Mismatch (- expected, + actual):\n%s", diff)
			}
		})
	}
}

//nolint:paralleltest
func TestLocateInterval(t *testing.T) {
	// Test data
	metadataMapping := map[playback.SequenceNumber]*segment.Metadata{
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
	expectedInterval := &playback.RewindInterval{
		Start: &playback.RewindMoment{
			Metadata:   metadataMapping[0],
			ActualTime: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
			TargetTime: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
			InGap:      false,
		},
		End: &playback.RewindMoment{
			Metadata:   metadataMapping[1],
			ActualTime: time.Date(2026, 1, 2, 10, 20, 34, 0, time.UTC),
			TargetTime: time.Date(2026, 1, 2, 10, 20, 34, 0, time.UTC),
			InGap:      false,
		},
	}
	expectedContext := &actions.LocateOutputContext{
		ID:                  testutil.TestVideoID,
		Title:               "Test title",
		StartSequenceNumber: 0,
		EndSequenceNumber:   1,
		ActualStartTime:     time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
		ActualEndTime:       time.Date(2026, 1, 2, 10, 20, 34, 0, time.UTC),
		ActualDuration:      4 * time.Second,
		InputStartTime:      time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
		InputEndTime:        time.Date(2026, 1, 2, 10, 20, 34, 0, time.UTC),
		InputDuration:       4 * time.Second,
	}

	testCases := []struct {
		name             string
		start            input.MomentValue
		end              input.MomentValue
		expectedInterval *playback.RewindInterval
		expectedContext  *actions.LocateOutputContext
	}{
		{
			name:             "time and time",
			start:            time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
			end:              time.Date(2026, 1, 2, 10, 20, 34, 0, time.UTC),
			expectedInterval: expectedInterval,
			expectedContext:  expectedContext,
		},
		{
			name:             "time and duration",
			start:            time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
			end:              4 * time.Second,
			expectedInterval: expectedInterval,
			expectedContext:  expectedContext,
		},
		{
			name:             "duration and time",
			start:            4 * time.Second,
			end:              time.Date(2026, 1, 2, 10, 20, 34, 0, time.UTC),
			expectedInterval: expectedInterval,
			expectedContext:  expectedContext,
		},
		{
			name:             "time and sequence number",
			start:            time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
			end:              1,
			expectedInterval: expectedInterval,
			expectedContext:  expectedContext,
		},
		{
			name:             "sequence number and time",
			start:            0,
			end:              time.Date(2026, 1, 2, 10, 20, 34, 0, time.UTC),
			expectedInterval: expectedInterval,
			expectedContext:  expectedContext,
		},
		{
			name:             "sequence number and sequence number",
			start:            0,
			end:              1,
			expectedInterval: expectedInterval,
			expectedContext:  expectedContext,
		},
		{
			name:  "time and time with non-zero difference",
			start: time.Date(2026, 1, 2, 10, 20, 31, 0, time.UTC),
			end:   time.Date(2026, 1, 2, 10, 20, 33, 0, time.UTC),
			expectedInterval: &playback.RewindInterval{
				Start: &playback.RewindMoment{
					Metadata:   metadataMapping[0],
					ActualTime: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
					TargetTime: time.Date(2026, 1, 2, 10, 20, 31, 0, time.UTC),
					InGap:      false,
				},
				End: &playback.RewindMoment{
					Metadata:   metadataMapping[1],
					ActualTime: time.Date(2026, 1, 2, 10, 20, 34, 0, time.UTC),
					TargetTime: time.Date(2026, 1, 2, 10, 20, 33, 0, time.UTC),
					InGap:      false,
				},
			},
			expectedContext: &actions.LocateOutputContext{
				ID:                  testutil.TestVideoID,
				Title:               "Test title",
				StartSequenceNumber: 0,
				EndSequenceNumber:   1,
				ActualStartTime:     time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
				ActualEndTime:       time.Date(2026, 1, 2, 10, 20, 34, 0, time.UTC),
				ActualDuration:      4 * time.Second,
				InputStartTime:      time.Date(2026, 1, 2, 10, 20, 31, 0, time.UTC),
				InputEndTime:        time.Date(2026, 1, 2, 10, 20, 33, 0, time.UTC),
				InputDuration:       2 * time.Second,
			},
		},
	}

	reference := *metadataMapping[1]
	for _, tc := range testCases { //nolint:paralleltest
		t.Run(tc.name, func(t *testing.T) {
			interval, context, err := actions.LocateInterval(
				pb,
				tc.start,
				tc.end,
				reference,
			)
			require.NoError(t, err)

			if diff := cmp.Diff(tc.expectedInterval, interval); diff != "" {
				t.Fatalf("Mismatch (- expected, + actual):\n%s", diff)
			}
			if diff := cmp.Diff(tc.expectedContext, context); diff != "" {
				t.Fatalf("Mismatch (- expected, + actual):\n%s", diff)
			}
		})
	}
}
