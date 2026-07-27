package input_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xymaxim/ypb/internal/input"
)

func TestValidateMoments(t *testing.T) {
	t.Parallel()

	pastTime := time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC)
	futureTime := time.Date(2099, 1, 2, 10, 20, 30, 0, time.UTC)

	testCases := []struct {
		name    string
		start   input.MomentValue
		end     input.MomentValue
		wantErr string
	}{
		{
			name:  "start and end in the past",
			start: pastTime,
			end:   pastTime.Add(time.Hour),
		},
		{
			name:    "start in the future",
			start:   futureTime,
			end:     futureTime.Add(time.Hour),
			wantErr: "start time is in the future",
		},
		{
			name:    "end in the future",
			start:   pastTime,
			end:     futureTime,
			wantErr: "end time is in the future",
		},
		{
			name:    "start after end",
			start:   pastTime.Add(2 * time.Hour),
			end:     pastTime.Add(time.Hour),
			wantErr: "start time is after end time",
		},
		{
			name:    "duration end pushes past now",
			start:   time.Now().Add(-time.Hour),
			end:     2 * time.Hour,
			wantErr: "end time is in the future",
		},
		{
			name:  "duration end stays in the past",
			start: pastTime,
			end:   time.Hour,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := input.ValidateMoments(tc.start, tc.end)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
