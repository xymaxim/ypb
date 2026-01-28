package input

import (
	"errors"
	"fmt"
	"time"

	"github.com/xymaxim/ypb/internal/playback"
)

// ValidateMoments performs preliminary validation on parsed start and end
// moment values to catch obvious errors.
func ValidateMoments(start, end MomentValue) error {
	switch s := start.(type) {
	case time.Time:
		if e, ok := end.(time.Time); ok && s.After(e) {
			return fmt.Errorf("start time is after end time: %v > %v", s, e)
		}
	case playback.SequenceNumber:
		if e, ok := end.(playback.SequenceNumber); ok && s > e {
			return fmt.Errorf("start segment is after end segment: %d > %d", s, e)
		}
	case time.Duration:
		if _, ok := end.(time.Duration); ok {
			return errors.New("both start and end cannot be durations")
		}
	case MomentKeyword:
		if s == NowKeyword {
			return fmt.Errorf("'%s' cannot be used at start", NowKeyword)
		}
	}
	return nil
}
