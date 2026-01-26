package actions_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/xymaxim/ypb/internal/actions"
	"github.com/xymaxim/ypb/internal/input"
	"github.com/xymaxim/ypb/internal/playback"
	"github.com/xymaxim/ypb/internal/playback/info"
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

func (pb *fakePlayback) Info() info.VideoInformation {
	return info.VideoInformation{
		ID:    "abcdefgh123",
		Title: "Test title",
	}
}

func (pb *fakePlayback) ProbeItag() string {
	return ""
}

func (pb *fakePlayback) RequestHeadSeqNum() (int, error) {
	return pb.fakeMetadata[len(pb.fakeMetadata)-1].SequenceNumber, nil
}

func (pb *fakePlayback) FetchSegmentMetadata(
	_ string,
	sq playback.SequenceNumber,
) (*segment.Metadata, error) {
	return pb.fakeMetadata[sq], nil
}

// LocateMoment returns the rewind moment corresponds the target time. For tests
// only. For example, it does not handle timeline gaps.
//
// When isEnd is false, segments are treated as closed at the start and open at
// the end [start, end).  A time exactly on a segment boundary belongs to the
// segment starting at that time.
//
// When isEnd is true, segments are treated as open at the start and closed at
// the end (start, end].  A time exactly on a segment boundary belongs to the
// segment ending at that time.

func (pb *fakePlayback) LocateMoment(
	t time.Time,
	reference segment.Metadata,
	isEnd bool,
) (*playback.RewindMoment, error) {
	timeDiff := t.Sub(reference.Time()).Nanoseconds()
	segmentDuration := reference.Duration.Nanoseconds()

	segmentOffset := timeDiff / segmentDuration
	timeRemainder := timeDiff % segmentDuration

	// Adjust for negative remainders (time before reference)
	if timeRemainder < 0 {
		segmentOffset--
		timeRemainder += segmentDuration
	}

	// Handle boundary conditions for segment end times
	if isEnd && timeRemainder == 0 {
		// Exact boundary belongs to the previous segment
		segmentOffset--
	}

	sq := reference.SequenceNumber + int(segmentOffset)
	m, ok := pb.fakeMetadata[sq]
	if !ok {
		panic(fmt.Sprintf("segment not found: %d", sq))
	}

	return playback.NewRewindMoment(t, m, isEnd, false), nil
}

func TestLocateMoment(t *testing.T) {
	t.Parallel()

	fakeMetadata := testutil.GenerateFakeSegmentMetadata(3, 2*time.Second)
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
		{
			name:  "now",
			value: "now",
			expected: &playback.RewindMoment{
				Metadata:   fakeMetadata[2],
				ActualTime: time.Date(2026, 1, 2, 10, 20, 34, 0, time.UTC),
				TargetTime: time.Date(2026, 1, 2, 10, 20, 34, 0, time.UTC),
				InGap:      false,
			},
		},
	}

	pb := newFakePlayback(fakeMetadata)
	reference := *fakeMetadata[2]
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			moment, err := actions.LocateMoment(pb, tc.value, reference)
			require.NoError(t, err)
			if diff := cmp.Diff(tc.expected, moment); diff != "" {
				t.Fatalf("Mismatch (- expected, + actual):\n%s", diff)
			}
		})
	}
}

func TestLocateInterval(t *testing.T) {
	t.Parallel()

	fakeMetadata := testutil.GenerateFakeSegmentMetadata(2, 2*time.Second)

	expectedInterval := &playback.RewindInterval{
		Start: &playback.RewindMoment{
			Metadata:   fakeMetadata[0],
			ActualTime: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
			TargetTime: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
			InGap:      false,
		},
		End: &playback.RewindMoment{
			Metadata:   fakeMetadata[1],
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
			name:             "sequence number and now",
			start:            0,
			end:              "now",
			expectedInterval: expectedInterval,
			expectedContext:  expectedContext,
		},
	}

	pb := newFakePlayback(fakeMetadata)
	reference := *fakeMetadata[len(fakeMetadata)-1]
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
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
