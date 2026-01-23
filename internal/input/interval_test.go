package input_test

import (
	"testing"
	"time"

	"github.com/xymaxim/ypb/internal/input"
)

func TestParseIntervalPart_Success(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name      string
		input     string
		wantErr   bool
		wantValue any
	}{
		{
			name:      "sequence number",
			input:     "123",
			wantErr:   true,
			wantValue: int(123),
		},
		{
			name:      "unix timestamp",
			input:     "@1767349230",
			wantErr:   true,
			wantValue: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
		},
		{
			name:    "only local date",
			input:   "2026-01-02",
			wantErr: true,
			//nolint:gosmopolitan
			wantValue: time.Date(2026, 1, 2, 0, 0, 0, 0, time.Local),
		},
		{
			name:    "local full date and time",
			input:   "2026-01-02T10:20:30",
			wantErr: true,
			//nolint:gosmopolitan
			wantValue: time.Date(2026, 1, 2, 10, 20, 30, 0, time.Local),
		},
		{
			name:      "zulu full date and time",
			input:     "2026-01-02T10:20:30Z",
			wantErr:   true,
			wantValue: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
		},
		{
			name:    "local date and time with hours and minutes",
			input:   "2026-01-02T10:20",
			wantErr: true,
			//nolint:gosmopolitan
			wantValue: time.Date(2026, 1, 2, 10, 20, 0, 0, time.Local),
		},
		{
			name:    "local date and time with +hh:mm offset",
			input:   "2026-01-02T10:20:30+01:00",
			wantErr: true,
			wantValue: time.Date(
				2026,
				1,
				2,
				10,
				20,
				30,
				0,
				time.FixedZone("+01:00", 3600),
			),
		},
		{
			name:    "date and time with -hh:mm offset",
			input:   "2026-01-02T10:20:30-01:00",
			wantErr: true,
			wantValue: time.Date(
				2026,
				1,
				2,
				10,
				20,
				30,
				0,
				time.FixedZone("-01:00", -3600),
			),
		},
		{
			name:    "date and time with +hhmm offset",
			input:   "2026-01-02T10:20:30+0100",
			wantErr: true,
			wantValue: time.Date(
				2026,
				1,
				2,
				10,
				20,
				30,
				0,
				time.FixedZone("+01:00", 3600),
			),
		},
		{
			name:    "date and time with +hh offset",
			input:   "2026-01-02T10:20:30+01",
			wantErr: true,
			wantValue: time.Date(
				2026,
				1,
				2,
				10,
				20,
				30,
				0,
				time.FixedZone("+01:00", 3600),
			),
		},
		{
			name:      "full duration",
			input:     "1d2h30m40s",
			wantErr:   true,
			wantValue: time.Duration(95440000000000),
		},
		{
			name:      "duration of hours and seconds",
			input:     "2h40s",
			wantErr:   true,
			wantValue: time.Duration(7240000000000),
		},
		{
			name:      "now keyword",
			input:     "now",
			wantErr:   true,
			wantValue: "now",
		},
		{
			name:      "earliest keyword",
			input:     "earliest",
			wantErr:   true,
			wantValue: "earliest",
		},
	}

	errorMessage := "mismatched\n want: %#v\n have: %#v"
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotValue, _ := input.ParseIntervalPart(tc.input)
			switch w := tc.wantValue.(type) {
			case time.Time:
				if !gotValue.(time.Time).Equal(w) {
					t.Errorf(errorMessage, w, gotValue)
				}
			default:
				if gotValue != tc.wantValue {
					t.Errorf(errorMessage, tc.wantValue, gotValue)
				}
			}
		})
	}
}

func TestParseInterval_Success(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name      string
		input     string
		wantErr   bool
		wantStart any
		wantEnd   any
	}{
		{
			name:      "two sequence numbers with slash",
			input:     "123/456",
			wantErr:   true,
			wantStart: 123,
			wantEnd:   456,
		},
		{
			name:      "sequence number and keyword with slash",
			input:     "123/now",
			wantErr:   true,
			wantStart: 123,
			wantEnd:   "now",
		},
		{
			name:      "two sequence numbers with two hyphens",
			input:     "123--456",
			wantErr:   true,
			wantStart: 123,
			wantEnd:   456,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotStart, gotEnd, gotErr := input.ParseInterval(tc.input)
			if gotErr != nil {
				t.Errorf("unexpected error\n %v", gotErr)
				return
			}
			if gotStart != tc.wantStart {
				t.Errorf(
					"mismatched start\n want: '%v'\n have: '%v'",
					tc.wantStart,
					gotStart,
				)
			}
			if gotEnd != tc.wantEnd {
				t.Errorf(
					"mismatched end\n want: '%v'\n have: '%v'",
					tc.wantEnd,
					gotEnd,
				)
			}
		})
	}
}

func TestParseInterval_Fail(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "now at start",
			input: "now/456",
		},
		{
			name:  "earliest at end",
			input: "123/earliest",
		},
		{
			name:  "two durations",
			input: "1h/2h",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			start, end, gotErr := input.ParseInterval(tc.input)
			if gotErr == nil {
				t.Errorf("should fail, but got value: %v, %v", start, end)
			}
		})
	}
}
