// This file extends Playback with segment location functionality.
//
// The search algorithm is based on the original ytpb's implementation
// (https://github.com/xymaxim/ytpb/pull/3) and consists of three steps:
//
//  1. Jump-based search: Uses time differences to quickly find a segment or
//     narrow the search domain
//  2. Binary search: Refines the search within the discovered domain
//  3. Gap detection: Validates whether the target time falls within a gap
//
// This multi-step approach handles timeline instabilities and gaps that could
// otherwise cause incorrect rewind timing estimations.
//
// Time comparisons use 'Ingestion-Walltime-US' metadata. For details on this
// choice and edge cases, see:
// https://github.com/xymaxim/ytpb/tree/main/notebooks

package playback

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/xymaxim/ypb/internal/playback/segment"
)

// timeDiffTolerance defines the acceptable time difference when matching
// segments.  See https://github.com/xymaxim/ytpb/issues/5.
const timeDiffTolerance = 50 * time.Millisecond

// maxJumpSteps limits the number of jump search iterations.
const maxJumpSteps = 10

// RewindMoment describes a specific point in time mapped to a segment-locate result.
type RewindMoment struct {
	Metadata   *segment.Metadata
	ActualTime time.Time
	TargetTime time.Time
	InGap      bool
}

// NewRewindMoment creates a RewindMoment from segment metadata and target time.
// If isEnd is true, uses the segment's end time as the actual time.
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

// TimeDifference returns the duration between target and actual times.
func (m *RewindMoment) TimeDifference() time.Duration {
	return m.ActualTime.Sub(m.TargetTime)
}

// RewindInterval represents a time range with start and end moments.
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
		slog.Bool("end", isEnd),
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
		if len(track) >= maxJumpSteps {
			msg := "jump search exceeded max steps, exit"
			slog.Warn(msg, "track", track)
			return nil, errors.New(msg)
		}

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
			moment := NewRewindMoment(targetTime, candidate, isEnd, false)
			slog.Info(
				"moment located via jump search",
				slog.Int("sq", moment.Metadata.SequenceNumber),
				slog.Duration("diff", moment.TimeDifference()),
				slog.Time("time", moment.ActualTime.UTC()),
			)

			return moment, nil
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

	slog.Info(
		"moment located via binary search",
		slog.Int("sq", moment.Metadata.SequenceNumber),
		slog.Duration("diff", moment.TimeDifference()),
		slog.Time("time", moment.ActualTime.UTC()),
	)

	return moment, nil
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

// fetchSegmentMetadata fetches segment metadata and wraps errors consistently.
func fetchSegmentMetadata(pb *Playback, sq SequenceNumber) (*segment.Metadata, error) {
	metadata, err := pb.FetchSegmentMetadata(pb.ProbeItag(), sq)
	if err != nil {
		return nil, NewSegmentMetadataFetchError(sq, err)
	}
	return metadata, nil
}
