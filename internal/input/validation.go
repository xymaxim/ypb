package input

import (
	"errors"
	"fmt"
	"time"

	"github.com/xymaxim/ypb/playback"
)

// ValidateMoments performs preliminary validation on parsed start and end
// moment values to catch obvious errors.
func ValidateMoments(start, end MomentValue) error {
	now := time.Now()

	switch s := start.(type) {
	case time.Time:
		if s.After(now) {
			return fmt.Errorf("start time is in the future: %v", s)
		}
		if e, ok := end.(time.Time); ok && s.After(e) {
			return fmt.Errorf("start time is after end time: %v > %v", s, e)
		}
		if d, ok := end.(time.Duration); ok {
			endTime := s.Add(d)
			if endTime.After(now) {
				return fmt.Errorf("end time is in the future: %v", endTime)
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

	if e, ok := end.(time.Time); ok && e.After(now) {
		return fmt.Errorf("end time is in the future: %v", e)
	}

	return nil
}

// ValidateNowLatency rejects the 'now' keyword when a non-zero latency is set.
func ValidateNowLatency(v MomentValue, latency time.Duration) error {
	if latency <= 0 {
		return nil
	}
	if k, ok := v.(MomentKeyword); ok && k == NowKeyword {
		return errors.New("cannot locate 'now' with latency")
	}
	return nil
}
