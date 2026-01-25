package actions

import (
	"errors"
	"fmt"
	"time"

	"github.com/xymaxim/ypb/internal/input"
	"github.com/xymaxim/ypb/internal/playback"
	"github.com/xymaxim/ypb/internal/playback/segment"
)

type LocateOutputContext struct {
	ID                  string
	Title               string
	StartSequenceNumber playback.SequenceNumber
	EndSequenceNumber   playback.SequenceNumber
	ActualStartTime     time.Time
	ActualEndTime       time.Time
	ActualDuration      time.Duration
	InputStartTime      time.Time
	InputEndTime        time.Time
	InputDuration       time.Duration
}

func LocateMoment(
	pb *playback.Playback,
	value input.MomentValue,
	reference segment.Metadata,
) (*playback.RewindMoment, error) {
	switch v := value.(type) {
	case time.Time:
		out, err := pb.LocateMoment(v, reference, false)
		if err != nil {
			return nil, fmt.Errorf("locating moment: %w", err)
		}
		return out, nil
	case playback.SequenceNumber:
		sm, err := pb.FetchSegmentMetadata(pb.GetReferenceItag(), v)
		if err != nil {
			return nil, fmt.Errorf("fetching segment metadata, sq=%d: %w", v, err)
		}
		return playback.NewRewindMoment(sm.Time(), sm, false, false), nil
	default:
		return nil, fmt.Errorf("got unallowed type %T: %v", v, v)
	}
}

func LocateInterval(
	pb *playback.Playback,
	start, end input.MomentValue,
	reference segment.Metadata,
) (*playback.RewindInterval, *LocateOutputContext, error) {
	interval, err := locateInterval(pb, start, end, reference)
	if err != nil {
		return nil, nil, fmt.Errorf("locating interval: %w", err)
	}

	context := &LocateOutputContext{
		Title:               pb.Info.Title,
		ID:                  pb.Info.ID,
		StartSequenceNumber: interval.Start.Metadata.SequenceNumber,
		EndSequenceNumber:   interval.End.Metadata.SequenceNumber,
		ActualStartTime:     interval.Start.ActualTime,
		ActualEndTime:       interval.End.ActualTime,
		ActualDuration:      interval.End.ActualTime.Sub(interval.Start.ActualTime),
		InputStartTime:      interval.Start.TargetTime,
		InputEndTime:        interval.End.TargetTime,
		InputDuration:       interval.End.TargetTime.Sub(interval.Start.TargetTime),
	}

	return interval, context, err
}

// LocateInterval finds start and end moments corresponding to the target times.
func locateInterval(
	pb *playback.Playback,
	start, end input.MomentValue,
	reference segment.Metadata,
) (*playback.RewindInterval, error) {
	switch s := start.(type) {
	case time.Time, playback.SequenceNumber:
		startMoment, err := resolveMoment(pb, s, reference, false)
		if err != nil {
			return nil, fmt.Errorf("resolving start moment: %w", err)
		}
		switch e := end.(type) {
		case time.Duration:
			endTime := startMoment.TargetTime.Add(e)
			endMoment, locErr := pb.LocateMoment(endTime, reference, true)
			if locErr != nil {
				return nil, fmt.Errorf("locating end moment: %w", locErr)
			}
			return &playback.RewindInterval{
				Start: startMoment,
				End:   endMoment,
			}, nil
		case time.Time, playback.SequenceNumber:
			endMoment, err := resolveMoment(pb, e, reference, true)
			if err != nil {
				return nil, fmt.Errorf("resolving end moment: %w", err)
			}
			return &playback.RewindInterval{
				Start: startMoment,
				End:   endMoment,
			}, nil
		}
	case time.Duration:
		switch e := end.(type) {
		case time.Time, playback.SequenceNumber:
			endMoment, err := resolveMoment(pb, e, reference, true)
			if err != nil {
				return nil, fmt.Errorf("resolving end moment: %w", err)
			}
			startTime := endMoment.TargetTime.Add(-s)
			startMoment, locErr := pb.LocateMoment(startTime, reference, false)
			if locErr != nil {
				return nil, fmt.Errorf("locating start moment: %w", locErr)
			}
			return &playback.RewindInterval{
				Start: startMoment,
				End:   endMoment,
			}, nil
		case time.Duration:
			return nil, errors.New("two durations are not allowed")
		}
	}
	return nil, errors.New("resolving start and end")
}

func resolveMoment(
	pb *playback.Playback,
	value any,
	reference segment.Metadata,
	isEnd bool,
) (*playback.RewindMoment, error) {
	switch v := value.(type) {
	case time.Time:
		m, err := pb.LocateMoment(v, reference, isEnd)
		if err != nil {
			return nil, fmt.Errorf("locating moment: %w", err)
		}
		return m, nil
	case playback.SequenceNumber:
		sm, err := pb.FetchSegmentMetadata(pb.GetReferenceItag(), v)
		if err != nil {
			return nil, fmt.Errorf("fetching segment metadata: %w", err)
		}
		var actualTime time.Time
		if isEnd {
			actualTime = sm.Time().Add(pb.Info.SegmentDuration)
		} else {
			actualTime = sm.Time()
		}
		return &playback.RewindMoment{
			Metadata:   sm,
			ActualTime: actualTime,
			TargetTime: actualTime,
			InGap:      false,
		}, nil
	default:
		return nil, fmt.Errorf("got unexpected type %T: %v", v, v)
	}
}
