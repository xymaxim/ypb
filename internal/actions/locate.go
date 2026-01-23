package actions

import (
	"fmt"
	"time"

	"github.com/xymaxim/ypb/internal/playback"
)

type LocateActionContext struct {
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

func Locate(
	pb *playback.Playback,
	start time.Time,
	end time.Time,
) (*playback.RewindInterval, *LocateActionContext, error) {
	referenceSeqNum, err := pb.RequestHeadSeqNum()
	if err != nil {
		return nil, nil, fmt.Errorf("requesting reference sequence number: %w", err)
	}
	reference, err := pb.FetchSegmentMetadata(pb.GetReferenceItag(), referenceSeqNum)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"fetching segment metadata, sq=%d: %w",
			reference.SequenceNumber,
			err,
		)
	}

	interval, err := pb.LocateInterval(start, end, *reference)
	if err != nil {
		return nil, nil, fmt.Errorf("locating interval: %w", err)
	}

	context := &LocateActionContext{
		Title:               pb.Info.Title,
		ID:                  pb.Info.ID,
		StartSequenceNumber: interval.Start.SequenceNumber,
		EndSequenceNumber:   interval.Start.SequenceNumber,
		ActualStartTime:     interval.Start.Time,
		ActualEndTime:       interval.End.Time,
		ActualDuration:      interval.End.Time.Sub(interval.Start.Time),
		InputStartTime:      start,
		InputEndTime:        end,
		InputDuration:       end.Sub(start),
	}

	return interval, context, err
}
