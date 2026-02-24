package actions

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/xymaxim/ypb/internal/input"
	"github.com/xymaxim/ypb/internal/playback"
	"github.com/xymaxim/ypb/internal/playback/segment"
)

// BadMomentTypeError indicates that an unsupported moment value type was encountered.
type BadMomentTypeError struct {
	Value  any
	Origin string
}

// NewBadMomentTypeError creates a new BadMomentTypeError.
func NewBadMomentTypeError(value any, origin string) *BadMomentTypeError {
	return &BadMomentTypeError{
		Value:  value,
		Origin: origin,
	}
}

func (e *BadMomentTypeError) Error() string {
	if e.Origin != "" {
		return fmt.Sprintf("unsupported type for %s: %T", e.Origin, e.Value)
	}
	return fmt.Sprintf("unsupported moment type: %T", e.Value)
}

// ResolveMomentError wraps errors that occur when resolving a moment value.
type ResolveMomentError struct {
	Moment input.MomentValue
	IsEnd  bool
	Err    error
}

// NewResolveMomentError create a ResolveMomentError.
func NewResolveMomentError(m input.MomentValue, isEnd bool, err error) *ResolveMomentError {
	return &ResolveMomentError{Moment: m, IsEnd: isEnd, Err: err}
}

func (e *ResolveMomentError) Error() string {
	position := "start"
	if e.IsEnd {
		position = "end"
	}
	return fmt.Sprintf("resolving %s moment '%v': %v", position, e.Moment, e.Err)
}

func (e *ResolveMomentError) Unwrap() error {
	return e.Err
}

// LocateOutputContext contains the resolved details of a located interval.
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

// LocateContext holds reference points used to locate and resolve moments.
//
// Head is the most recent segment available in the stream. Reference serves as
// the base for relative time calculations. PinnedTime represents the time of
// the 'now' keyword: in strict mode (downloads and capture), it is set to the
// app start-up time; in non-strict mode (serve), it is nil and 'now' falls back
// to the end of the most recent segment.
type LocateContext struct {
	Head         segment.Metadata
	Reference    segment.Metadata
	PinnedTime   *time.Time
	PinnedMoment *playback.RewindMoment
}

// NewLocateContext creates a new LocateContext.
//
// If reference is nil, the most recent (head) segment will be used as the
// reference.
func NewLocateContext(
	pb playback.Playbacker,
	reference *segment.Metadata,
	pinnedTime *time.Time,
) (*LocateContext, error) {
	head, err := fetchHeadMetadata(pb)
	if err != nil {
		return nil, fmt.Errorf("fetching head segment metadata: %w", err)
	}

	if reference == nil {
		reference = head
	}

	if pinnedTime != nil {
		slog.Info("pinned time", slog.Time("time", *pinnedTime))
	}

	return &LocateContext{
		Head:       *head,
		Reference:  *reference,
		PinnedTime: pinnedTime,
	}, nil
}

// LocateMoment locates a single moment.
func LocateMoment(
	pb playback.Playbacker,
	value input.MomentValue,
	ctx *LocateContext,
) (*playback.RewindMoment, error) {
	out, err := resolveMoment(pb, value, ctx, false)
	if err != nil {
		return nil, NewResolveMomentError(value, false, err)
	}
	return out, nil
}

// LocateInterval locates start and end moments of an interval.
func LocateInterval(
	pb playback.Playbacker,
	start, end input.MomentValue,
	ctx *LocateContext,
) (*playback.RewindInterval, *LocateOutputContext, error) {
	slog.Info("locating interval", "start", start, "end", end)

	if err := validateMoments(start, end, ctx); err != nil {
		return nil, nil, err
	}

	interval, err := locateStartAndEnd(pb, start, end, ctx)
	if err != nil {
		return nil, nil, err
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

	return interval, context, nil
}

func fetchHeadMetadata(pb playback.Playbacker) (*segment.Metadata, error) {
	sq, err := pb.RequestHeadSeqNum()
	if err != nil {
		return nil, fmt.Errorf("requesting head segment: %w", err)
	}
	m, err := pb.FetchSegmentMetadata(pb.ProbeItag(), sq)
	if err != nil {
		return nil, playback.NewSegmentMetadataFetchError(sq, err)
	}
	return m, nil
}

func validateMoments(start, end input.MomentValue, ctx *LocateContext) error {
	switch s := start.(type) {
	case time.Time:
		if s.After(ctx.Head.EndTime()) {
			return fmt.Errorf(
				"start time is after head segment: %v > %v",
				s,
				ctx.Head.EndTime(),
			)
		}
	case playback.SequenceNumber:
		if s > ctx.Head.SequenceNumber {
			return fmt.Errorf(
				"start segment %d is after head one %d",
				s,
				ctx.Head.SequenceNumber,
			)
		}
	}
	return nil
}

// locateStartAndEnd resolves both start and end moments into a RewindInterval.
func locateStartAndEnd(
	pb playback.Playbacker,
	start, end input.MomentValue,
	ctx *LocateContext,
) (*playback.RewindInterval, error) {
	if isAbsoluteMoment(start) {
		return locateWithAbsoluteStart(pb, start, end, ctx)
	}
	if duration, ok := start.(time.Duration); ok {
		return locateWithDurationStart(pb, duration, end, ctx)
	}
	return nil, NewBadMomentTypeError(start, "start moment")
}

// isAbsoluteMoment reports whether the value represents an absolute point in time.
func isAbsoluteMoment(value input.MomentValue) bool {
	_, ok := value.(time.Duration)
	return !ok
}

// locateWithAbsoluteStart handles intervals where the start is an absolute moment.
func locateWithAbsoluteStart(
	pb playback.Playbacker,
	start, end input.MomentValue,
	ctx *LocateContext,
) (*playback.RewindInterval, error) {
	startMoment, err := resolveMoment(pb, start, ctx, false)
	if err != nil {
		return nil, NewResolveMomentError(start, false, err)
	}

	// Handle absolute end
	if isAbsoluteMoment(end) {
		endMoment, err := resolveMoment(pb, end, ctx, true)
		if err != nil {
			return nil, NewResolveMomentError(end, true, err)
		}

		if startMoment.TargetTime.After(endMoment.ActualTime) {
			return nil, NewResolveMomentError(
				start,
				false,
				errors.New("start is after end"),
			)
		}

		return &playback.RewindInterval{Start: startMoment, End: endMoment}, nil
	}

	// Handle duration end
	if duration, ok := end.(time.Duration); ok {
		endTime := startMoment.TargetTime.Add(duration)
		endMoment, err := pb.LocateMoment(endTime, ctx.Reference, true)
		if err != nil {
			return nil, fmt.Errorf("locating end moment: %w", err)
		}
		return &playback.RewindInterval{Start: startMoment, End: endMoment}, nil
	}

	return nil, NewBadMomentTypeError(end, "end moment (with absolute start)")
}

// locateWithDurationStart handles intervals where the start is a duration.
func locateWithDurationStart(
	pb playback.Playbacker,
	startDuration time.Duration,
	end input.MomentValue,
	ctx *LocateContext,
) (*playback.RewindInterval, error) {
	if _, ok := end.(time.Duration); ok {
		return nil, errors.New("both start and end cannot be durations")
	}
	if isAbsoluteMoment(end) {
		endMoment, err := resolveMoment(pb, end, ctx, true)
		if err != nil {
			return nil, NewResolveMomentError(end, true, err)
		}
		startTime := endMoment.TargetTime.Add(-startDuration)
		startMoment, err := pb.LocateMoment(startTime, ctx.Reference, false)
		if err != nil {
			return nil, fmt.Errorf("locating start moment: %w", err)
		}
		return &playback.RewindInterval{Start: startMoment, End: endMoment}, nil
	}
	return nil, NewBadMomentTypeError(end, "end moment (with duration start)")
}

// resolveMoment resolves any MomentValue into a RewindMoment.
func resolveMoment(
	pb playback.Playbacker,
	value input.MomentValue,
	ctx *LocateContext,
	isEnd bool,
) (*playback.RewindMoment, error) {
	switch v := value.(type) {
	case time.Time:
		return resolveTime(pb, v, ctx, isEnd)
	case playback.SequenceNumber:
		return resolveSequenceNumber(pb, v, ctx, isEnd)
	case input.MomentKeyword:
		return resolveKeyword(pb, v, ctx, isEnd)
	case input.MomentExpression:
		return resolveExpression(pb, v, ctx, isEnd)
	default:
		return nil, NewBadMomentTypeError(v, "")
	}
}

// resolveTime resolves the target time t into a RewindMoment.
func resolveTime(
	pb playback.Playbacker,
	t time.Time,
	ctx *LocateContext,
	isEnd bool,
) (*playback.RewindMoment, error) {
	if t.After(ctx.Head.EndTime()) {
		return nil, fmt.Errorf("time %v is after current moment", t)
	}
	moment, err := pb.LocateMoment(t, ctx.Reference, isEnd)
	if err != nil {
		return nil, fmt.Errorf("locating moment at %v: %w", t, err)
	}
	return moment, nil
}

// resolveSequenceNumber resolves the sequence number sq into a RewindMoment.
func resolveSequenceNumber(
	pb playback.Playbacker,
	sq playback.SequenceNumber,
	ctx *LocateContext,
	isEnd bool,
) (*playback.RewindMoment, error) {
	if sq > ctx.Head.SequenceNumber {
		return nil, fmt.Errorf(
			"segment %d is not yet available, current: %d",
			sq,
			ctx.Head.SequenceNumber,
		)
	}

	metadata, err := pb.FetchSegmentMetadata(pb.ProbeItag(), sq)
	if err != nil {
		return nil, playback.NewSegmentMetadataFetchError(sq, err)
	}

	targetTime := metadata.Time()
	if isEnd {
		targetTime = metadata.EndTime()
	}

	return playback.NewRewindMoment(targetTime, *metadata, isEnd, false), nil
}

// resolveKeywordMoment resolves a keyword into a RewindMoment.
func resolveKeyword(
	pb playback.Playbacker,
	keyword input.MomentKeyword,
	ctx *LocateContext,
	isEnd bool,
) (*playback.RewindMoment, error) {
	switch keyword {
	case input.NowKeyword:
		if ctx.PinnedMoment != nil {
			return ctx.PinnedMoment, nil
		}

		if ctx.PinnedTime != nil {
			m, err := resolveTime(pb, *ctx.PinnedTime, ctx, isEnd)
			if err != nil {
				return nil, fmt.Errorf(
					"resolving pinned time %q: %w",
					ctx.PinnedTime,
					err,
				)
			}
			ctx.PinnedMoment = m
		} else {
			ctx.PinnedMoment = playback.NewRewindMoment(
				ctx.Head.EndTime(),
				ctx.Head,
				isEnd,
				false,
			)
		}

		slog.Debug(
			"resolved now keyword",
			slog.Int("sq", ctx.PinnedMoment.Metadata.SequenceNumber),
			slog.Time("time", ctx.PinnedMoment.TargetTime),
		)

		return ctx.PinnedMoment, nil

	default:
		return nil, fmt.Errorf("unknown keyword: '%s'", keyword)
	}
}

// resolveExpression evaluates the moment expression expr into a RewindMoment.
func resolveExpression(
	pb playback.Playbacker,
	expr input.MomentExpression,
	ctx *LocateContext,
	isEnd bool,
) (*playback.RewindMoment, error) {
	// Resolve left operand to a concrete time
	var leftTime time.Time
	if expr.Left == input.NowKeyword {
		if expr.Operator == input.OpPlus {
			return nil, fmt.Errorf("'%s' cannot be used with plus", input.NowKeyword)
		}
		moment, err := resolveMoment(pb, expr.Left, ctx, false)
		if err != nil {
			return nil, NewResolveMomentError(input.NowKeyword, isEnd, err)
		}
		leftTime = moment.TargetTime
	} else {
		leftTime = expr.Left.(time.Time)
	}

	// Apply the operator to calculate target time
	var targetTime time.Time
	switch expr.Operator {
	case input.OpPlus:
		targetTime = leftTime.Add(expr.Right)
	case input.OpMinus:
		targetTime = leftTime.Add(-expr.Right)
	}

	// Resolve and return the moment
	moment, err := resolveMoment(pb, targetTime, ctx, isEnd)
	if err != nil {
		return nil, fmt.Errorf("locating time '%v': %w", targetTime, err)
	}

	return moment, nil
}
