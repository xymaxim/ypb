package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xymaxim/ypb/internal/input"
)

func TestResolvePinnedTime(t *testing.T) {
	t.Parallel()
	before := time.Now().UTC()
	testCases := []struct {
		name      string
		nowFlag   string
		wantErr   bool
		checkFunc func(t *testing.T, got time.Time)
	}{
		{
			name:    "empty string returns current time",
			nowFlag: "",
			checkFunc: func(t *testing.T, got time.Time) {
				after := time.Now().UTC()
				assert.True(
					t,
					!got.Before(before) && !got.After(after),
					"expected time between %v and %v, got %v",
					before,
					after,
					got,
				)
			},
		},
		{
			name:    "datetime with UTC offset",
			nowFlag: "2026-01-02T10:20:30+00",
			checkFunc: func(t *testing.T, got time.Time) {
				assert.True(
					t,
					got.Equal(time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC)),
					"expected 2026-01-02T10:20:30 UTC, got %v",
					got,
				)
			},
		},
		{
			name:    "datetime with positive offset",
			nowFlag: "2026-01-02T10:20:30+02",
			checkFunc: func(t *testing.T, got time.Time) {
				assert.Equal(t, 2026, got.Year())
				assert.Equal(t, time.January, got.Month())
				assert.Equal(t, 2, got.Day())
				assert.Equal(t, 10, got.Hour())
				assert.Equal(t, 20, got.Minute())
				assert.Equal(t, 30, got.Second())
			},
		},
		{
			name:    "date only",
			nowFlag: "2026-01-02",
			checkFunc: func(t *testing.T, got time.Time) {
				assert.Equal(t, 2026, got.Year())
				assert.Equal(t, time.January, got.Month())
				assert.Equal(t, 2, got.Day())
				assert.Equal(t, 0, got.Hour())
				assert.Equal(t, 0, got.Minute())
				assert.Equal(t, 0, got.Second())
			},
		},
		{
			name:    "now keyword",
			nowFlag: "now",
			checkFunc: func(t *testing.T, got time.Time) {
				after := time.Now().UTC()
				assert.True(
					t,
					!got.Before(before) && !got.After(after),
					"expected time between %v and %v, got %v",
					before,
					after,
					got,
				)
			},
		},
		{
			name:    "time only",
			nowFlag: "10:20:30",
			checkFunc: func(t *testing.T, got time.Time) {
				assert.Equal(t, 10, got.Hour())
				assert.Equal(t, 20, got.Minute())
				assert.Equal(t, 30, got.Second())
			},
		},
		{
			name:    "duration is rejected",
			nowFlag: "1h",
			wantErr: true,
		},
		{
			name:    "sequence number is rejected",
			nowFlag: "123",
			wantErr: true,
		},
		{
			name:    "earliest keyword is rejected",
			nowFlag: "earliest",
			wantErr: true,
		},
		{
			name:    "invalid string is rejected",
			nowFlag: "invalid",
			wantErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ResolvePinnedTime(tc.nowFlag)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tc.checkFunc(t, got)
		})
	}
}

func TestNowFlagIntegration(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		nowFlag string
		input   string
		check   func(t *testing.T, v input.MomentValue)
	}{
		{
			name:    "datetime input with datetime --now",
			nowFlag: "2026-06-15T12:00:00+00",
			input:   "2026-01-02T10:20:30+00",
			check: func(t *testing.T, v input.MomentValue) {
				tt, ok := v.(time.Time)
				require.True(t, ok, "expected time.Time, got %T", v)
				assert.True(
					t,
					tt.Equal(time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC)),
					"expected 2026-01-02T10:20:30 UTC, got %v",
					tt,
				)
			},
		},
		{
			name:    "now resolves to pinned time",
			nowFlag: "2026-01-02T10:20:30+00",
			input:   "now",
			check: func(t *testing.T, v input.MomentValue) {
				tt, ok := v.(time.Time)
				require.True(t, ok, "expected time.Time, got %T", v)
				assert.True(
					t,
					tt.Equal(time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC)),
					"expected 2026-01-02T10:20:30 UTC, got %v",
					tt,
				)
			},
		},
		{
			name:    "now expression resolves left operand to pinned time",
			nowFlag: "2026-01-02T10:20:30+00",
			input:   "now - 10m",
			check: func(t *testing.T, v input.MomentValue) {
				expr, ok := v.(input.MomentExpression)
				require.True(t, ok, "expected MomentExpression, got %T", v)
				tt, ok := expr.Left.(time.Time)
				require.True(t, ok, "expected time.Time left, got %T", expr.Left)
				assert.True(
					t,
					tt.Equal(time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC)),
					"expected 2026-01-02T10:20:30 UTC, got %v",
					tt,
				)
				assert.Equal(t, input.OpMinus, expr.Operator)
				assert.Equal(t, 10*time.Minute, expr.Right)
			},
		},
		{
			name:    "now plus expression resolves left operand to pinned time",
			nowFlag: "2026-01-02",
			input:   "now + 1h",
			check: func(t *testing.T, v input.MomentValue) {
				expr, ok := v.(input.MomentExpression)
				require.True(t, ok, "expected MomentExpression, got %T", v)
				tt, ok := expr.Left.(time.Time)
				require.True(t, ok, "expected time.Time left, got %T", expr.Left)
				assert.Equal(t, 2026, tt.Year())
				assert.Equal(t, time.January, tt.Month())
				assert.Equal(t, 2, tt.Day())
				assert.Equal(t, input.OpPlus, expr.Operator)
				assert.Equal(t, time.Hour, expr.Right)
			},
		},
		{
			name:    "now keyword stays when --now is not set",
			nowFlag: "",
			input:   "now",
			check: func(t *testing.T, v input.MomentValue) {
				assert.Equal(t, input.NowKeyword, v)
			},
		},
		{
			name:    "time-only uses date from date --now",
			nowFlag: "2026-01-02",
			input:   "10:20",
			check: func(t *testing.T, v input.MomentValue) {
				tt, ok := v.(time.Time)
				require.True(t, ok, "expected time.Time, got %T", v)
				assert.Equal(t, 2026, tt.Year())
				assert.Equal(t, time.January, tt.Month())
				assert.Equal(t, 2, tt.Day())
				assert.Equal(t, 10, tt.Hour())
				assert.Equal(t, 20, tt.Minute())
			},
		},
		{
			name:    "time-only uses date from datetime --now",
			nowFlag: "2026-01-02T10:20:30+00",
			input:   "10:20",
			check: func(t *testing.T, v input.MomentValue) {
				tt, ok := v.(time.Time)
				require.True(t, ok, "expected time.Time, got %T", v)
				assert.Equal(t, 2026, tt.Year())
				assert.Equal(t, time.January, tt.Month())
				assert.Equal(t, 2, tt.Day())
				assert.Equal(t, 10, tt.Hour())
				assert.Equal(t, 20, tt.Minute())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pinnedTime, err := ResolvePinnedTime(tc.nowFlag)
			require.NoError(t, err)

			var refTime *time.Time
			if tc.nowFlag != "" {
				refTime = &pinnedTime
			}
			value, err := input.ParseIntervalPart(tc.input, refTime)
			require.NoError(t, err)

			tc.check(t, value)
		})
	}
}

func TestNowFlagIntervalIntegration(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name      string
		nowFlag   string
		input     string
		wantStart func(t *testing.T, v input.MomentValue)
		wantEnd   func(t *testing.T, v input.MomentValue)
		wantErr   bool
	}{
		{
			name:    "now at start with --now set resolves to pinned time",
			nowFlag: "2026-01-02T10:20:30+00",
			input:   "now/1h",
			wantStart: func(t *testing.T, v input.MomentValue) {
				tt, ok := v.(time.Time)
				require.True(t, ok, "expected time.Time, got %T", v)
				assert.True(
					t,
					tt.Equal(time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC)),
					"expected 2026-01-02T10:20:30 UTC, got %v",
					tt,
				)
			},
			wantEnd: func(t *testing.T, v input.MomentValue) {
				d, ok := v.(time.Duration)
				require.True(t, ok, "expected time.Duration, got %T", v)
				assert.Equal(t, time.Hour, d)
			},
		},
		{
			name:    "now at start without --now is rejected",
			nowFlag: "",
			input:   "now/1h",
			wantErr: true,
		},
		{
			name:    "now plus expression with --now set resolves left to pinned time",
			nowFlag: "2026-01-02T10:20:30+00",
			input:   "now + 1h/2h",
			wantStart: func(t *testing.T, v input.MomentValue) {
				expr, ok := v.(input.MomentExpression)
				require.True(t, ok, "expected MomentExpression, got %T", v)
				tt, ok := expr.Left.(time.Time)
				require.True(t, ok, "expected time.Time left, got %T", expr.Left)
				assert.True(
					t,
					tt.Equal(time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC)),
					"expected 2026-01-02T10:20:30 UTC, got %v",
					tt,
				)
				assert.Equal(t, input.OpPlus, expr.Operator)
				assert.Equal(t, time.Hour, expr.Right)
			},
			wantEnd: func(t *testing.T, v input.MomentValue) {
				d, ok := v.(time.Duration)
				require.True(t, ok, "expected time.Duration, got %T", v)
				assert.Equal(t, 2*time.Hour, d)
			},
		},
		{
			name:    "now plus expression without --now is rejected",
			nowFlag: "",
			input:   "now + 1h/2h",
			wantStart: func(t *testing.T, v input.MomentValue) {
				expr, ok := v.(input.MomentExpression)
				require.True(t, ok, "expected MomentExpression, got %T", v)
				assert.Equal(t, input.NowKeyword, expr.Left)
				assert.Equal(t, input.OpPlus, expr.Operator)
				assert.Equal(t, time.Hour, expr.Right)
			},
			wantEnd: func(t *testing.T, v input.MomentValue) {
				d, ok := v.(time.Duration)
				require.True(t, ok, "expected time.Duration, got %T", v)
				assert.Equal(t, 2*time.Hour, d)
			},
		},
		{
			name:    "now minus start and now end with --now set resolves to pinned time",
			nowFlag: "2026-01-02T10:20:30+00",
			input:   "now - 1h/now",
			wantStart: func(t *testing.T, v input.MomentValue) {
				expr, ok := v.(input.MomentExpression)
				require.True(t, ok, "expected MomentExpression, got %T", v)
				tt, ok := expr.Left.(time.Time)
				require.True(t, ok, "expected time.Time left, got %T", expr.Left)
				assert.True(
					t,
					tt.Equal(time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC)),
					"expected 2026-01-02T10:20:30 UTC, got %v",
					tt,
				)
				assert.Equal(t, input.OpMinus, expr.Operator)
				assert.Equal(t, time.Hour, expr.Right)
			},
			wantEnd: func(t *testing.T, v input.MomentValue) {
				tt, ok := v.(time.Time)
				require.True(t, ok, "expected time.Time, got %T", v)
				assert.True(
					t,
					tt.Equal(time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC)),
					"expected 2026-01-02T10:20:30 UTC, got %v",
					tt,
				)
			},
		},
		{
			name:    "now minus start without --now passes parsing",
			nowFlag: "",
			input:   "now - 1h/now",
			wantStart: func(t *testing.T, v input.MomentValue) {
				expr, ok := v.(input.MomentExpression)
				require.True(t, ok, "expected MomentExpression, got %T", v)
				assert.Equal(t, input.NowKeyword, expr.Left)
				assert.Equal(t, input.OpMinus, expr.Operator)
				assert.Equal(t, time.Hour, expr.Right)
			},
			wantEnd: func(t *testing.T, v input.MomentValue) {
				assert.Equal(t, input.NowKeyword, v)
			},
		},
		{
			name:    "time only start with now end and --now set",
			nowFlag: "2026-01-02T10:20:30+00",
			input:   "10:20/now",
			wantStart: func(t *testing.T, v input.MomentValue) {
				tt, ok := v.(time.Time)
				require.True(t, ok, "expected time.Time, got %T", v)
				assert.Equal(t, 2026, tt.Year())
				assert.Equal(t, time.January, tt.Month())
				assert.Equal(t, 2, tt.Day())
				assert.Equal(t, 10, tt.Hour())
				assert.Equal(t, 20, tt.Minute())
			},
			wantEnd: func(t *testing.T, v input.MomentValue) {
				tt, ok := v.(time.Time)
				require.True(t, ok, "expected time.Time, got %T", v)
				assert.True(
					t,
					tt.Equal(time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC)),
					"expected 2026-01-02T10:20:30 UTC, got %v",
					tt,
				)
			},
		},
		{
			name:    "time only start with now end without --now",
			nowFlag: "",
			input:   "10:20/now",
			wantStart: func(t *testing.T, v input.MomentValue) {
				tt, ok := v.(time.Time)
				require.True(t, ok, "expected time.Time, got %T", v)
				now := time.Now()
				assert.Equal(t, now.Year(), tt.Year())
				assert.Equal(t, now.Month(), tt.Month())
				assert.Equal(t, now.Day(), tt.Day())
				assert.Equal(t, 10, tt.Hour())
				assert.Equal(t, 20, tt.Minute())
			},
			wantEnd: func(t *testing.T, v input.MomentValue) {
				assert.Equal(t, input.NowKeyword, v)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pinnedTime, err := ResolvePinnedTime(tc.nowFlag)
			require.NoError(t, err)

			var refTime *time.Time
			if tc.nowFlag != "" {
				refTime = &pinnedTime
			}
			start, end, err := input.ParseInterval(tc.input, refTime)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tc.wantStart(t, start)
			tc.wantEnd(t, end)
		})
	}
}
