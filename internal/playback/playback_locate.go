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
	"log"
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
		actual = metadata.Time().Add(metadata.Duration)
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

// LocateMoment finds the RewindMoment corresponding to a target time.
//
// The function runs the search relative to the arbitrary reference point
// specified by a sequence number and time. Usually, its choice comes down to
// the head segment, but the closest segment to a target time is preferable. If
// isEnd is true, the search moment is treated as an interval end.
func (pb *Playback) LocateMoment(
	targetTime time.Time,
	reference segment.Metadata,
	isEnd bool,
) (*RewindMoment, error) {
	hasSegmentFound := false
	hasDomainDiscovered := false
	hasSignChanged := false

	var track []SequenceNumber

	currentTimeDiff := targetTime.Sub(reference.Time())
	startDirection := math.Copysign(1, currentTimeDiff.Seconds())

	slog.Info(
		"locating moment",
		slog.Time("time", targetTime.In(time.UTC)),
		slog.Group(
			"reference",
			slog.Int("sq", reference.SequenceNumber),
			slog.Time("time", reference.Time()),
		),
	)

	candidateSeqNum := reference.SequenceNumber
	candidateMetadata, err := pb.FetchSegmentMetadata(pb.ProbeItag(), candidateSeqNum)
	if err != nil {
		return nil, fmt.Errorf(
			"fetching segment metadata for sq=%d: %w",
			candidateSeqNum,
			err,
		)
	}

	// Step 1
	for !hasDomainDiscovered {
		track = append(track, candidateSeqNum)
		slog.Debug(
			"jump search step",
			slog.Int("sq", candidateSeqNum),
			slog.Duration("diff", currentTimeDiff),
			slog.Time("time", candidateMetadata.Time().In(time.UTC)),
		)

		if currentTimeDiff >= 0 {
			if currentTimeDiff <= pb.Info().SegmentDuration+timeDiffTolerance {
				hasSegmentFound = true
				break
			}
		}

		direction := math.Copysign(1, currentTimeDiff.Seconds())
		if !hasSignChanged {
			hasSignChanged = direction*startDirection < 0
		}

		hasDomainDiscovered = hasSignChanged && (direction == startDirection)

		jumpSizeSec := currentTimeDiff.Seconds() / pb.Info().SegmentDuration.Seconds()
		jumpSize := int(math.Floor(jumpSizeSec))
		candidateSeqNum += jumpSize
		candidateMetadata, err = pb.FetchSegmentMetadata(
			pb.ProbeItag(),
			candidateSeqNum,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"fetching segment metadata, sq=%d: %w",
				candidateSeqNum,
				err,
			)
		}

		currentTimeDiff = targetTime.Sub(candidateMetadata.Time())
	}

	// Step 2
	var result *RewindMoment
	if hasSegmentFound {
		result = NewRewindMoment(targetTime, candidateMetadata, isEnd, false)
	} else {
		startSeqNum, endSeqNum := track[len(track)-2], track[len(track)-1]
		result, err = pb.searchInRange(targetTime, startSeqNum, endSeqNum, isEnd)
	}

	slog.Debug(
		"moment located",
		slog.Int("sq", candidateSeqNum),
		slog.Duration("diff", currentTimeDiff),
		slog.Time("time", candidateMetadata.Time().In(time.UTC)),
	)

	return result, err
}

// searchinRange performs a binary search within a search domain.
func (pb *Playback) searchInRange(
	targetTime time.Time,
	startSeqNum, endSeqNum int,
	isEnd bool,
) (*RewindMoment, error) {
	getBisectedTime := func(seqNum SequenceNumber, targetTime time.Time) (time.Time, error) {
		metadata, err := pb.FetchSegmentMetadata(pb.ProbeItag(), seqNum)
		if err != nil {
			return time.Time{}, fmt.Errorf(
				"fetching segment metadata for sq=%d: %w",
				seqNum,
				err,
			)
		}
		slog.Debug(
			"Bisect step",
			slog.Int("sq", seqNum),
			slog.Duration("diff", targetTime.Sub(metadata.Time())),
			slog.Time("time", metadata.Time()),
		)
		return metadata.Time(), nil
	}

	foundIndex := sort.Search(endSeqNum-startSeqNum+1, func(k int) bool {
		t, err := getBisectedTime(startSeqNum+k, targetTime)
		if err != nil {
			log.Fatalf("getting bisected time for sq=%d: %v", startSeqNum+k, err)
		}
		return !t.Before(targetTime)
	})

	candidateSeqNum := startSeqNum + foundIndex - 1
	candidateMetadata, err := pb.FetchSegmentMetadata(pb.ProbeItag(), candidateSeqNum)
	if err != nil {
		return nil, fmt.Errorf(
			"fetching segment metadata for sq=%d: %w",
			candidateSeqNum,
			err,
		)
	}

	// After Step 2 the time difference is always positive
	timeDiff := targetTime.Sub(candidateMetadata.Time())

	// Step 3
	isInGap := false
	if pb.Info().SegmentDuration < timeDiff-timeDiffTolerance {
		slog.Info("target time falls inside a gap")
		isInGap = true
		if !isEnd {
			candidateSeqNum += 1
			candidateMetadata, err = pb.FetchSegmentMetadata(
				pb.ProbeItag(),
				candidateSeqNum,
			)
			if err != nil {
				return nil, fmt.Errorf(
					"fetching segment metadata, sq=%d: %w",
					candidateSeqNum,
					err,
				)
			}
			timeDiff = targetTime.Sub(candidateMetadata.Time())
			slog.Debug(
				"took next segment",
				slog.Duration("diff", timeDiff),
				slog.Int("sq", candidateSeqNum),
				slog.Time("time", candidateMetadata.Time()),
			)
		}
	}

	moment := NewRewindMoment(targetTime, candidateMetadata, isEnd, isInGap)

	return moment, nil
}
