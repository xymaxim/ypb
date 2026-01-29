// This file extends Playback with segment locating given a target time.
//
// The search algorithm is based on the original ytpb's implementation [1] and
// consists of three steps: (1) a "look around", jump-based search to find a
// segment directly or outline a search domain (the jump length is based on the
// time difference [2] and constant duration of segments); (2) search for a
// segment in the outlined domain using a binary search if a segment is not
// found in the previous step; (3) check whether a target time falls inside a
// gap or not.
//
// Three steps are required for accurate results since a stream timeline due
// to instability may contain numerous gaps, which leads to under- or
// overestimation of rewind timings.
//
// References:
//
// 1. See https://github.com/xymaxim/ytpb/pull/3
// 2. For time comparison, we rely on the 'Ingestion-Walltime-US' metadata (see
//    https://github.com/xymaxim/ytpb/tree/main/notebooks on why this value was
//    chosen and different edge cases)

package playback

import (
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/xymaxim/ypb/internal/playback/segment"
)

// timeDiffTolerance is the absolute time difference tolerance. See
// https://github.com/xymaxim/ytpb/issues/5.
const timeDiffTolerance = 50 * time.Millisecond

// RewindMoment describes a specific point in time mapped to a segment-locate result.
type RewindMoment struct {
	Metadata   *segment.Metadata
	ActualTime time.Time
	TargetTime time.Time
	InGap      bool
}

func NewRewindMoment(
	target time.Time,
	metadata *segment.Metadata,
	isEnd, inGap bool,
) *RewindMoment {
	var actual time.Time
	if isEnd {
		actual = metadata.EndTime()
	} else {
		actual = metadata.Time()
	}
	return &RewindMoment{
		Metadata:   metadata,
		ActualTime: actual,
		TargetTime: target,
		InGap:      inGap,
	}
}

func (m *RewindMoment) TimeDifference() time.Duration {
	return m.TargetTime.Sub(m.ActualTime)
}

// RewindInterval holds start and end rewind moments.
type RewindInterval struct {
	Start *RewindMoment
	End   *RewindMoment
}

// LocateMoment finds the RewindMoment corresponding to a targetTime.
//
// The search begins from a reference point (typically the head segment or the
// closest known segment to the target). If isEnd is true, the search moment is
// treated as an interval end.
func (pb *Playback) LocateMoment(
	targetTime time.Time,
	reference segment.Metadata,
	isEnd bool,
) (*RewindMoment, error) {
	slog.Info(
		"locating moment",
		slog.Time("time", targetTime.In(time.UTC)),
		slog.Group(
			"reference",
			slog.Int("sq", reference.SequenceNumber),
			slog.Time("time", reference.Time()),
		),
	)

	// Step 1: Jump-based search to quickly locate a segment or narrow the
	// search domain for next steps.
	var track []SequenceNumber

	initialTimeDiff := targetTime.Sub(reference.Time())
	initialDirection := math.Copysign(1, initialTimeDiff.Seconds())

	currentSeqNum := reference.SequenceNumber
	candidate, err := fetchSegmentMetadata(pb, currentSeqNum)
	if err != nil {
		return nil, err
	}

	directionChanged := false

	for {
		track = append(track, currentSeqNum)
		candidateTimeDiff := targetTime.Sub(candidate.Time())
		slog.Debug(
			"jump search step",
			slog.Int("sq", currentSeqNum),
			slog.Duration("diff", candidateTimeDiff),
			slog.Time("time", candidate.Time().UTC()),
		)

		// Check if target time falls within current segment
		maxAllowed := candidate.Duration + timeDiffTolerance
		if 0 <= candidateTimeDiff && candidateTimeDiff <= maxAllowed {
			slog.Debug(
				"segment located via jump search",
				slog.Int("sq", currentSeqNum),
			)
			return NewRewindMoment(targetTime, candidate, isEnd, false), nil
		}

		// Check if a search domain has been outlined
		currentDirection := math.Copysign(1, candidateTimeDiff.Seconds())
		if !directionChanged {
			directionChanged = currentDirection*initialDirection < 0
		}
		if directionChanged && currentDirection == initialDirection {
			break
		}

		// Jump to next candidate segment
		currentSeqNum += calculateSegmentOffset(targetTime, candidate, isEnd)
		candidate, err = fetchSegmentMetadata(pb, currentSeqNum)
		if err != nil {
			return nil, err
		}
	}

	// Step 2 and 3: Binary search within discovered domain and gap detection
	var moment *RewindMoment
	startSeqNum, endSeqNum := track[len(track)-2], track[len(track)-1]
	moment, err = pb.searchInRange(targetTime, startSeqNum, endSeqNum, isEnd)
	if err != nil {
		return nil, fmt.Errorf("searching in range: %w", err)
	}

	slog.Debug(
		"segment located via binary search",
		slog.Int("sq", moment.Metadata.SequenceNumber),
	)

	return moment, nil
}

// calculateSegmentOffset calculates the sequence number offset of the segment
// that contains time t relative to the provided reference. If isEnd is true, an
// exact boundary time is treated as belonging to the previous segment.
func calculateSegmentOffset(t time.Time, reference *segment.Metadata, isEnd bool) SequenceNumber {
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

	return int(segmentOffset)
}

// searchInRange performs binary search within the specified domain and handles
// gaps. This implements Step 2 and Step 3 of the search algorithm.
func (pb *Playback) searchInRange(
	targetTime time.Time,
	startSeqNum, endSeqNum int,
	isEnd bool,
) (*RewindMoment, error) {
	slog.Debug(
		"start binary search",
		slog.Int("start", startSeqNum),
		slog.Int("end", endSeqNum),
	)

	// Find the segment whose time is >= targetTime
	foundIndex := sort.Search(endSeqNum-startSeqNum+1, func(k int) bool {
		sq := startSeqNum + k
		metadata, err := fetchSegmentMetadata(pb, sq)
		if err != nil {
			slog.Error(
				"fetching during binary search",
				slog.Int("sq", sq),
				slog.Any("error", err),
			)
			return false
		}

		slog.Debug(
			"bisect step",
			slog.Int("sq", sq),
			slog.Duration("diff", targetTime.Sub(metadata.Time())),
			slog.Time("time", metadata.Time().UTC()),
		)

		return !metadata.Time().Before(targetTime)
	})

	// The target segment is just before the found index
	candidate, err := fetchSegmentMetadata(pb, startSeqNum+foundIndex-1)
	if err != nil {
		return nil, err
	}

	// After above the time difference is always positive
	timeDiff := targetTime.Sub(candidate.Time())

	// Step 3: Detect and handle gaps
	if timeDiff-timeDiffTolerance <= candidate.Duration {
		return NewRewindMoment(targetTime, candidate, isEnd, false), nil
	}

	slog.Info(
		"target time falls inside a gap",
		slog.Int("sq", candidate.SequenceNumber),
		slog.Duration("diff", timeDiff),
	)

	// For interval starts, use the next segment after the gap
	if !isEnd {
		next, err := fetchSegmentMetadata(pb, candidate.SequenceNumber+1)
		if err != nil {
			return nil, err
		}

		timeDiff = targetTime.Sub(next.Time())
		slog.Debug(
			"using next segment after gap",
			slog.Int("sq", next.SequenceNumber),
			slog.Duration("diff", timeDiff),
			slog.Time("time", next.Time().UTC()),
		)

		candidate = next
	}

	return NewRewindMoment(targetTime, candidate, isEnd, true), nil
}

// fetchSegmentMetadata fetches segment metadata and wraps errors consistently.
func fetchSegmentMetadata(pb *Playback, sq SequenceNumber) (*segment.Metadata, error) {
	metadata, err := pb.FetchSegmentMetadata(pb.ProbeItag(), sq)
	if err != nil {
		return nil, NewSegmentMetadataFetchError(sq, err)
	}
	return metadata, nil
}
