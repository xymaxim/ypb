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

func TestValidateLatencyWindow(t *testing.T) {
	t.Parallel()

	pastTime := time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC)

	testCases := []struct {
		name    string
		value   input.MomentValue
		latency time.Duration
		wantErr string
	}{
		{
			name:    "now keyword with latency",
			value:   input.NowKeyword,
			latency: 30 * time.Second,
			wantErr: "cannot locate 'now' with latency 30s: within the latency window, not yet ingested",
		},
		{
			name:    "now keyword without latency",
			value:   input.NowKeyword,
			latency: 0,
		},
		{
			name:    "time with latency",
			value:   pastTime,
			latency: 30 * time.Second,
		},
		{
			name:    "duration with latency",
			value:   time.Minute,
			latency: 30 * time.Second,
		},
		{
			name: "now minus duration expression with latency",
			value: input.MomentExpression{
				Operator: input.OpMinus,
				Left:     input.NowKeyword,
				Right:    time.Minute,
			},
			latency: 30 * time.Second,
		},
		{
			// 10s is inside the 30s latency window, so the content broadcast
			// 10s ago is not yet ingested.
			name: "now minus duration within the latency window",
			value: input.MomentExpression{
				Operator: input.OpMinus,
				Left:     input.NowKeyword,
				Right:    10 * time.Second,
			},
			latency: 30 * time.Second,
			wantErr: "cannot locate 'now - 10s' with latency 30s: within the latency window, not yet ingested",
		},
		{
			// 30s is exactly the latency window edge: the content broadcast
			// 30s ago is ingested right now.
			name: "now minus duration at the window edge",
			value: input.MomentExpression{
				Operator: input.OpMinus,
				Left:     input.NowKeyword,
				Right:    30 * time.Second,
			},
			latency: 30 * time.Second,
		},
		{
			name: "now minus duration without latency",
			value: input.MomentExpression{
				Operator: input.OpMinus,
				Left:     input.NowKeyword,
				Right:    10 * time.Second,
			},
			latency: 0,
		},
		{
			name:    "absolute time within the latency window",
			value:   time.Now().Add(-10 * time.Second),
			latency: 30 * time.Second,
			wantErr: "with latency 30s: within the latency window, not yet ingested",
		},
		{
			name:    "absolute time beyond the latency window",
			value:   time.Now().Add(-2 * time.Hour),
			latency: 30 * time.Second,
		},
		{
			name:    "absolute time at the window edge",
			value:   time.Now().Add(-30 * time.Second),
			latency: 30 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := input.ValidateLatencyWindow(tc.value, tc.latency)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
