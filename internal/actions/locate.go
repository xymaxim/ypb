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
	pb playback.Playbacker,
	value input.MomentValue,
	reference segment.Metadata,
) (*playback.RewindMoment, error) {
	out, err := resolveMoment(pb, value, reference, false)
	if err != nil {
		return nil, fmt.Errorf("resolving moment: %w", err)
	}
	return out, nil
}

func LocateInterval(
	pb playback.Playbacker,
	start, end input.MomentValue,
	reference segment.Metadata,
) (*playback.RewindInterval, *LocateOutputContext, error) {
	interval, err := locateInterval(pb, start, end, reference)
	if err != nil {
		return nil, nil, fmt.Errorf("locating interval: %w", err)
	}

	context := &LocateOutputContext{
		Title:               pb.Info().Title,
		ID:                  pb.Info().ID,
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
	pb playback.Playbacker,
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
		case string:
			switch e {
			case "now":
				endMoment, err := resolveMoment(pb, e, reference, true)
				if err != nil {
					return nil, fmt.Errorf("resolving end moment: %w", err)
				}
				return &playback.RewindInterval{
					Start: startMoment,
					End:   endMoment,
				}, nil
			default:
				return nil, fmt.Errorf("got unknown keyword '%s'", e)
			}
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
		case string:
			switch e {
			case "now":
				endMoment, err := resolveMoment(pb, e, reference, true)
				if err != nil {
					return nil, fmt.Errorf("resolving end moment: %w", err)
				}
				targetTime := endMoment.Metadata.EndTime().Add(-s)
				startMoment, err := resolveMoment(pb, targetTime, reference, false)
				if err != nil {
					return nil, fmt.Errorf("resolving start moment: %w", err)
				}
				return &playback.RewindInterval{
					Start: startMoment,
					End:   endMoment,
				}, nil
			default:
				return nil, fmt.Errorf("got unknown keyword '%s'", e)
			}
		case time.Duration:
			return nil, errors.New("two durations are not allowed")
		}
	}
	return nil, errors.New("resolving start and end")
}

func resolveMoment(
	pb playback.Playbacker,
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
		sm, err := pb.FetchSegmentMetadata(pb.ProbeItag(), v)
		if err != nil {
			return nil, fmt.Errorf("fetching segment metadata: %w", err)
		}

		var targetTime time.Time
		if isEnd {
			targetTime = sm.EndTime()
		} else {
			targetTime = sm.Time()
		}

		return playback.NewRewindMoment(targetTime, sm, isEnd, false), nil
	case string:
		switch v {
		case "now":
			sq, err := pb.RequestHeadSeqNum()
			if err != nil {
				return nil, fmt.Errorf(
					"requesting head sequence number, sq=%d: %w",
					sq,
					err,
				)
			}
			now, err := pb.FetchSegmentMetadata(pb.ProbeItag(), sq)
			if err != nil {
				return nil, fmt.Errorf(
					"fetching segment metadata, sq=%d: %w",
					sq,
					err,
				)
			}
			return &playback.RewindMoment{
				Metadata:   now,
				TargetTime: now.EndTime(),
				ActualTime: now.EndTime(),
				InGap:      false,
			}, nil
		default:
			return nil, fmt.Errorf("got unknown keyword '%s'", v)
		}
	default:
		return nil, fmt.Errorf("got unexpected type %T: %v", v, v)
	}
}
