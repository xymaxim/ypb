package actions_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"

	"github.com/xymaxim/ypb/info"
	"github.com/xymaxim/ypb/internal/actions"
	"github.com/xymaxim/ypb/internal/input"
	"github.com/xymaxim/ypb/internal/testutil"
	"github.com/xymaxim/ypb/playback"
	"github.com/xymaxim/ypb/segment"
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
	m, ok := pb.fakeMetadata[sq]
	if !ok {
		return nil, fmt.Errorf("fetching segment metadata, sq=%d", sq)
	}
	return &m, nil
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
			value: input.NowKeyword,
			expected: &playback.RewindMoment{
				Metadata:   fakeMetadata[2],
				ActualTime: time.Date(2026, 1, 2, 10, 20, 34, 0, time.UTC),
				TargetTime: time.Date(2026, 1, 2, 10, 20, 36, 0, time.UTC),
				InGap:      false,
			},
		},

		// Arithmetic expressions
		{
			name: "time minus duration",
			value: input.MomentExpression{
				Left:     time.Date(2026, 1, 2, 10, 20, 36, 0, time.UTC),
				Operator: input.OpMinus,
				Right:    time.Second,
			},
			expected: &playback.RewindMoment{
				Metadata:   fakeMetadata[2],
				ActualTime: time.Date(2026, 1, 2, 10, 20, 34, 0, time.UTC),
				TargetTime: time.Date(2026, 1, 2, 10, 20, 35, 0, time.UTC),
				InGap:      false,
			},
		},
		{
			name: "now minus duration",
			value: input.MomentExpression{
				Left:     input.NowKeyword,
				Operator: input.OpMinus,
				Right:    time.Second,
			},
			expected: &playback.RewindMoment{
				Metadata:   fakeMetadata[2],
				ActualTime: time.Date(2026, 1, 2, 10, 20, 34, 0, time.UTC),
				TargetTime: time.Date(2026, 1, 2, 10, 20, 35, 0, time.UTC),
				InGap:      false,
			},
		},
	}

	pb := newFakePlayback(fakeMetadata)
	now := fakeMetadata[len(fakeMetadata)-1]
	ctx := &actions.LocateContext{Head: now, Reference: now}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			moment, err := actions.LocateMoment(pb, tc.value, ctx)
			require.NoError(t, err)
			if diff := cmp.Diff(tc.expected, moment); diff != "" {
				t.Fatalf("Mismatch (- expected, + actual):\n%s", diff)
			}
		})
	}
}

func TestLocateMoment_BadMomentType(t *testing.T) {
	t.Parallel()

	fakeMetadata := testutil.GenerateFakeSegmentMetadata(3, 2*time.Second)
	testCases := []struct {
		name    string
		value   input.MomentValue
		wantErr *actions.BadMomentTypeError
	}{
		{
			name:    "duration",
			value:   time.Second,
			wantErr: actions.NewBadMomentTypeError(time.Second, ""),
		},
		{
			name:    "any string",
			value:   "abc",
			wantErr: actions.NewBadMomentTypeError("abc", ""),
		},
	}

	pb := newFakePlayback(fakeMetadata)
	now := fakeMetadata[len(fakeMetadata)-1]
	ctx := &actions.LocateContext{Head: now, Reference: now}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := actions.LocateMoment(pb, tc.value, ctx)

			var gotErr *actions.BadMomentTypeError
			if !errors.As(err, &gotErr) {
				t.Fatalf("expected BadMomentTypeError, got %T: %v", err, err)
			}
			if diff := cmp.Diff(gotErr, tc.wantErr); diff != "" {
				t.Fatalf("error mismatch (- got, + want):\n%s", diff)
			}
		})
	}
}

func TestLocateMoment_NowPlusWithoutPinnedTime(t *testing.T) {
	t.Parallel()

	fakeMetadata := testutil.GenerateFakeSegmentMetadata(3, 2*time.Second)
	pb := newFakePlayback(fakeMetadata)
	now := fakeMetadata[len(fakeMetadata)-1]
	ctx := &actions.LocateContext{Head: now, Reference: now}

	value := input.MomentExpression{
		Left:     input.NowKeyword,
		Operator: input.OpPlus,
		Right:    time.Hour,
	}
	_, err := actions.LocateMoment(pb, value, ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be used with plus")
}

func TestLocateMoment_Latency(t *testing.T) {
	t.Parallel()

	fakeMetadata := testutil.GenerateFakeSegmentMetadata(10, 2*time.Second)

	latency := 4 * time.Second

	testCases := []struct {
		name     string
		value    input.MomentValue
		expected *playback.RewindMoment
		wantErr  string
	}{
		{
			// The now keyword is at the live edge: with latency, the content
			// on air is not yet ingested, so locating it fails.
			name:    "now keyword at the live edge",
			value:   input.NowKeyword,
			wantErr: "cannot locate 'now' with latency",
		},
		{
			// A moment within the latency window of the live edge cannot be
			// located either: 10:20:49 located at 10:20:53, after head.
			name:    "time within the latency window",
			value:   time.Date(2026, 1, 2, 10, 20, 49, 0, time.UTC),
			wantErr: "with latency",
		},
		{
			name:  "time",
			value: time.Date(2026, 1, 2, 10, 20, 44, 0, time.UTC),
			expected: &playback.RewindMoment{
				Metadata:   fakeMetadata[9],
				ActualTime: time.Date(2026, 1, 2, 10, 20, 44, 0, time.UTC),
				TargetTime: time.Date(2026, 1, 2, 10, 20, 44, 0, time.UTC),
				InGap:      false,
			},
		},
		{
			// Latency shifts the search time forward by 4s: 10:20:42 is located
			// at 10:20:46, landing on sq 8 (a plain lookup at 10:20:42 would
			// hit sq 6).
			name:  "search time shifted by latency",
			value: time.Date(2026, 1, 2, 10, 20, 42, 0, time.UTC),
			expected: &playback.RewindMoment{
				Metadata:   fakeMetadata[8],
				ActualTime: time.Date(2026, 1, 2, 10, 20, 42, 0, time.UTC),
				TargetTime: time.Date(2026, 1, 2, 10, 20, 42, 0, time.UTC),
				InGap:      false,
			},
		},
		{
			name: "now minus duration",
			value: input.MomentExpression{
				Left:     input.NowKeyword,
				Operator: input.OpMinus,
				Right:    10 * time.Second,
			},
			expected: &playback.RewindMoment{
				Metadata:   fakeMetadata[7],
				ActualTime: time.Date(2026, 1, 2, 10, 20, 40, 0, time.UTC),
				TargetTime: time.Date(2026, 1, 2, 10, 20, 40, 0, time.UTC),
				InGap:      false,
			},
		},
	}

	pb := newFakePlayback(fakeMetadata)
	now := fakeMetadata[len(fakeMetadata)-1]
	ctx := &actions.LocateContext{Head: now, Reference: now, Latency: latency}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			moment, err := actions.LocateMoment(pb, tc.value, ctx)
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
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
		// Time at start
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
			name:             "time and sequence number",
			start:            time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
			end:              1,
			expectedInterval: expectedInterval,
			expectedContext:  expectedContext,
		},

		// Sequence number at start
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

		// Duration at start
		{
			name:             "duration and time",
			start:            4 * time.Second,
			end:              time.Date(2026, 1, 2, 10, 20, 34, 0, time.UTC),
			expectedInterval: expectedInterval,
			expectedContext:  expectedContext,
		},

		// 'Now' at end
		{
			name:             "time and now",
			start:            time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
			end:              input.NowKeyword,
			expectedInterval: expectedInterval,
			expectedContext:  expectedContext,
		},
		{
			name:             "sequence number and now",
			start:            0,
			end:              input.NowKeyword,
			expectedInterval: expectedInterval,
			expectedContext:  expectedContext,
		},
		{
			name:             "duration and now",
			start:            4 * time.Second,
			end:              input.NowKeyword,
			expectedInterval: expectedInterval,
			expectedContext:  expectedContext,
		},
	}

	pb := newFakePlayback(fakeMetadata)
	now := fakeMetadata[len(fakeMetadata)-1]
	ctx := &actions.LocateContext{Head: now, Reference: now}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			interval, context, err := actions.LocateInterval(
				pb,
				tc.start,
				tc.end,
				ctx,
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

func TestLocateInterval_Failure(t *testing.T) {
	t.Parallel()

	fakeMetadata := testutil.GenerateFakeSegmentMetadata(3, 2*time.Second)

	testCases := []struct {
		name    string
		start   input.MomentValue
		end     input.MomentValue
		wantErr error
	}{
		{
			name:  "end time after current moment",
			start: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
			end:   time.Date(2026, 1, 2, 23, 59, 59, 0, time.UTC),
			wantErr: actions.NewResolveMomentError(
				time.Date(2026, 1, 2, 23, 59, 59, 0, time.UTC),
				true,
				nil,
			),
		},
	}

	pb := newFakePlayback(fakeMetadata)
	now := fakeMetadata[len(fakeMetadata)-1]
	ctx := &actions.LocateContext{Head: now, Reference: now}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := actions.LocateInterval(pb, tc.start, tc.end, ctx)

			var gotErr *actions.ResolveMomentError
			if !errors.As(err, &gotErr) {
				t.Fatalf("expected ResolveMomentError, got %T: %v", err, err)
			}
			opts := cmpopts.IgnoreFields(actions.ResolveMomentError{}, "Err")
			if diff := cmp.Diff(gotErr, tc.wantErr, opts); diff != "" {
				t.Fatalf("error mismatch (- got, + want):\n%s", diff)
			}
		})
	}
}

func TestLocateInterval_Latency(t *testing.T) {
	t.Parallel()

	fakeMetadata := testutil.GenerateFakeSegmentMetadata(5, 2*time.Second)

	expectedInterval := &playback.RewindInterval{
		Start: &playback.RewindMoment{
			Metadata:   fakeMetadata[1],
			ActualTime: time.Date(2026, 1, 2, 10, 20, 31, 0, time.UTC),
			TargetTime: time.Date(2026, 1, 2, 10, 20, 32, 0, time.UTC),
			InGap:      false,
		},
		End: &playback.RewindMoment{
			Metadata:   fakeMetadata[4],
			ActualTime: time.Date(2026, 1, 2, 10, 20, 39, 0, time.UTC),
			TargetTime: time.Date(2026, 1, 2, 10, 20, 38, 0, time.UTC),
			InGap:      false,
		},
	}
	expectedContext := &actions.LocateOutputContext{
		ID:                  testutil.TestVideoID,
		Title:               "Test title",
		StartSequenceNumber: 1,
		EndSequenceNumber:   4,
		ActualStartTime:     time.Date(2026, 1, 2, 10, 20, 31, 0, time.UTC),
		ActualEndTime:       time.Date(2026, 1, 2, 10, 20, 39, 0, time.UTC),
		ActualDuration:      8 * time.Second,
		InputStartTime:      time.Date(2026, 1, 2, 10, 20, 32, 0, time.UTC),
		InputEndTime:        time.Date(2026, 1, 2, 10, 20, 38, 0, time.UTC),
		InputDuration:       6 * time.Second,
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
			start:            time.Date(2026, 1, 2, 10, 20, 32, 0, time.UTC),
			end:              time.Date(2026, 1, 2, 10, 20, 38, 0, time.UTC),
			expectedInterval: expectedInterval,
			expectedContext:  expectedContext,
		},
		{
			name:  "time and duration",
			start: time.Date(2026, 1, 2, 10, 20, 32, 0, time.UTC),
			end:   4 * time.Second,
			expectedInterval: &playback.RewindInterval{
				Start: expectedInterval.Start,
				End: &playback.RewindMoment{
					Metadata:   fakeMetadata[3],
					ActualTime: time.Date(2026, 1, 2, 10, 20, 37, 0, time.UTC),
					TargetTime: time.Date(2026, 1, 2, 10, 20, 36, 0, time.UTC),
					InGap:      false,
				},
			},
			expectedContext: &actions.LocateOutputContext{
				ID:                  testutil.TestVideoID,
				Title:               "Test title",
				StartSequenceNumber: 1,
				EndSequenceNumber:   3,
				ActualStartTime:     time.Date(2026, 1, 2, 10, 20, 31, 0, time.UTC),
				ActualEndTime:       time.Date(2026, 1, 2, 10, 20, 37, 0, time.UTC),
				ActualDuration:      6 * time.Second,
				InputStartTime:      time.Date(2026, 1, 2, 10, 20, 32, 0, time.UTC),
				InputEndTime:        time.Date(2026, 1, 2, 10, 20, 36, 0, time.UTC),
				InputDuration:       4 * time.Second,
			},
		},
	}

	pb := newFakePlayback(fakeMetadata)
	now := fakeMetadata[len(fakeMetadata)-1]
	ctx := &actions.LocateContext{Head: now, Reference: now, Latency: time.Second}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			interval, context, err := actions.LocateInterval(
				pb,
				tc.start,
				tc.end,
				ctx,
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

func TestLocateInterval_Latency_LiveEdge(t *testing.T) {
	t.Parallel()

	fakeMetadata := testutil.GenerateFakeSegmentMetadata(5, 2*time.Second)

	// A 10s duration end from 10:20:30 lands exactly on the head end
	// (10:20:40). With a 1s latency the located end would be 10:20:41, which
	// is not yet ingested, so the interval must be rejected with a
	// latency-aware message.
	testCases := []struct {
		name    string
		start   input.MomentValue
		end     input.MomentValue
		wantErr string
	}{
		{
			name:  "duration end with latency after head",
			start: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
			end:   10 * time.Second,
			wantErr: "end time 2026-01-02 10:20:40 +0000 UTC with latency 1s is after " +
				"current moment: 2026-01-02 10:20:41 +0000 UTC > " +
				"2026-01-02 10:20:40 +0000 UTC",
		},
	}

	pb := newFakePlayback(fakeMetadata)
	now := fakeMetadata[len(fakeMetadata)-1]
	ctx := &actions.LocateContext{Head: now, Reference: now, Latency: time.Second}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := actions.LocateInterval(pb, tc.start, tc.end, ctx)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}
}
