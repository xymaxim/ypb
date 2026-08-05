package input

import (
	"errors"
	"fmt"
	"time"

	"github.com/xymaxim/ypb/playback"
)

// ValidateMoments checks start/end ordering and rejects duration ends in the
// future.
func ValidateMoments(start, end MomentValue, now time.Time) error {
	switch s := start.(type) {
	case time.Time:
		if e, ok := end.(time.Time); ok && s.After(e) {
			return fmt.Errorf("start time is after end time: %v > %v", s, e)
		}
		if d, ok := end.(time.Duration); ok {
			endTime := s.Add(d)
			if endTime.After(now) {
				return futureErr(endTime.String())
			}
		}
	case playback.SequenceNumber:
		if e, ok := end.(playback.SequenceNumber); ok && s > e {
			return fmt.Errorf("start segment is after end segment: %d > %d", s, e)
		}
	case time.Duration:
		if _, ok := end.(time.Duration); ok {
			return errors.New("both start and end cannot be durations")
		}
	}

	return nil
}

// ValidateMoment rejects moments that are in the future or in the latency window.
func ValidateMoment(v MomentValue, latency time.Duration, now time.Time) error {
	switch m := v.(type) {
	case MomentKeyword:
		if latency > 0 && m == NowKeyword {
			return latencyErr("'now'", latency)
		}
	case MomentExpression:
		if m.Left == NowKeyword {
			if m.Operator == OpPlus && m.Right > 0 {
				return futureErr(
					fmt.Sprintf("'now %c %s'", m.Operator, m.Right),
				)
			}
			if latency > 0 && m.Operator == OpMinus && m.Right < latency {
				return latencyErr(fmt.Sprintf("'now - %s'", m.Right), latency)
			}
			return nil
		}

		t, ok := m.Left.(time.Time)
		if !ok {
			return nil
		}
		var target time.Time
		if m.Operator == OpPlus {
			target = t.Add(m.Right)
		} else {
			target = t.Add(-m.Right)
		}
		if target.After(now) {
			return futureErr(target.String())
		}
		if latency > 0 && target.Add(latency).After(now) {
			return latencyErr(target.String(), latency)
		}
	case time.Time:
		if m.After(now) {
			return futureErr(m.String())
		}
		if latency > 0 && m.Add(latency).After(now) {
			return latencyErr(m.String(), latency)
		}
	}

	return nil
}

func latencyErr(what string, latency time.Duration) error {
	return fmt.Errorf(
		"cannot locate %s with latency %s: within the latency window, not yet ingested",
		what,
		latency,
	)
}

func futureErr(what string) error {
	return fmt.Errorf("cannot locate %s: in the future", what)
}
