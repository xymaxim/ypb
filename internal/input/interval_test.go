package input_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/xymaxim/ypb/internal/input"
)

func TestParseIntervalPart(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name      string
		input     string
		wantErr   bool
		wantValue any
	}{
		// Sequence number
		{
			name:      "sequence number",
			input:     "123",
			wantErr:   false,
			wantValue: 123,
		},

		// Unix timestamp
		{
			name:      "unix timestamp",
			input:     "@1767349230",
			wantErr:   false,
			wantValue: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
		},

		// Date and time
		{
			name:    "only local date",
			input:   "2026-01-02",
			wantErr: false,
			//nolint:gosmopolitan
			wantValue: time.Date(2026, 1, 2, 0, 0, 0, 0, time.Local),
		},
		{
			name:    "local full date and time",
			input:   "2026-01-02T10:20:30",
			wantErr: false,
			//nolint:gosmopolitan
			wantValue: time.Date(2026, 1, 2, 10, 20, 30, 0, time.Local),
		},
		{
			name:      "zulu full date and time",
			input:     "2026-01-02T10:20:30Z",
			wantErr:   false,
			wantValue: time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
		},
		{
			name:    "local date and time with hours and minutes",
			input:   "2026-01-02T10:20",
			wantErr: false,
			//nolint:gosmopolitan
			wantValue: time.Date(2026, 1, 2, 10, 20, 0, 0, time.Local),
		},
		{
			name:    "local date and time with +hh:mm offset",
			input:   "2026-01-02T10:20:30+01:00",
			wantErr: false,
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
			wantErr: false,
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
			wantErr: false,
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
			wantErr: false,
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

		// Only time
		{
			name:    "only local time",
			input:   "10:20:30",
			wantErr: false,
			wantValue: func() time.Time {
				now := time.Now()
				return time.Date(
					now.Year(),
					now.Month(),
					now.Day(),
					10,
					20,
					30,
					0,
					time.Local, //nolint:gosmopolitan
				)
			}(),
		},
		{
			name:    "only time with time zone",
			input:   "10:20:30+00",
			wantErr: false,
			wantValue: func() time.Time {
				now := time.Now()
				return time.Date(
					now.Year(),
					now.Month(),
					now.Day(),
					10,
					20,
					30,
					0,
					time.UTC,
				)
			}(),
		},

		// Duration
		{
			name:      "full duration",
			input:     "1d2h30m40s",
			wantErr:   false,
			wantValue: time.Duration(95440000000000),
		},
		{
			name:      "duration of hours and seconds",
			input:     "2h40s",
			wantErr:   false,
			wantValue: time.Duration(7240000000000),
		},

		// Keywords
		{
			name:      "now keyword",
			input:     "now",
			wantErr:   false,
			wantValue: input.NowKeyword,
		},
		{
			name:      "earliest keyword",
			input:     "earliest",
			wantErr:   false,
			wantValue: input.MomentKeyword("earliest"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value, err := input.ParseIntervalPart(tc.input)
			if err == nil && tc.wantErr {
				t.Fatalf("should fail, got: %v", value)
			}
			if err != nil && !tc.wantErr {
				t.Fatalf("should not fail, got %v", err)
			}
			if diff := cmp.Diff(tc.wantValue, value); diff != "" {
				t.Fatalf("Mismatch (- want, + have):\n%s", diff)
			}
		})
	}
}

func TestParseIntervalPart_Expressions(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name      string
		input     string
		wantErr   bool
		wantValue any
	}{
		{
			name:    "date and time plus duration",
			input:   "2026-01-02T10:20:30+00 + 1h",
			wantErr: false,
			wantValue: input.MomentExpression{
				Left:     time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
				Operator: input.OpPlus,
				Right:    time.Hour,
			},
		},
		{
			name:    "date and time minus duration",
			input:   "2026-01-02T10:20:30+00 - 1h",
			wantErr: false,
			wantValue: input.MomentExpression{
				Left:     time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
				Operator: input.OpMinus,
				Right:    time.Hour,
			},
		},
		{
			name:    "only local time plus duration",
			input:   "10:20:30 + 1h",
			wantErr: false,
			wantValue: input.MomentExpression{
				Left: func() time.Time {
					now := time.Now()
					return time.Date(
						now.Year(),
						now.Month(),
						now.Day(),
						10,
						20,
						30,
						0,
						time.Local, //nolint:gosmopolitan
					)
				}(),
				Operator: input.OpPlus,
				Right:    time.Hour,
			},
		},
		{
			name:    "only local time plus duration",
			input:   "10:20:30 + 1h",
			wantErr: false,
			wantValue: input.MomentExpression{
				Left: func() time.Time {
					now := time.Now()
					return time.Date(
						now.Year(),
						now.Month(),
						now.Day(),
						10,
						20,
						30,
						0,
						time.Local, //nolint:gosmopolitan
					)
				}(),
				Operator: input.OpPlus,
				Right:    time.Hour,
			},
		},
		{
			name:    "unix timestamp plus duration",
			input:   "@1767349230 + 1h",
			wantErr: false,
			wantValue: input.MomentExpression{
				Left:     time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC),
				Operator: input.OpPlus,
				Right:    time.Hour,
			},
		},
		{
			name:    "sequence number plus duration",
			input:   "123 + 1h",
			wantErr: false,
			wantValue: input.MomentExpression{
				Left:     123,
				Operator: input.OpPlus,
				Right:    time.Hour,
			},
		},
		{
			name:    "now minus duration",
			input:   "now - 1h",
			wantErr: false,
			wantValue: input.MomentExpression{
				Left:     input.NowKeyword,
				Operator: input.OpMinus,
				Right:    time.Hour,
			},
		},
	}

	// Expand test cases to include "without spaces" variants
	expandedTestCases := make([]struct {
		name      string
		input     string
		wantErr   bool
		wantValue any
	}, 0, 2*len(testCases))

	for _, tc := range testCases {
		expandedTestCases = append(expandedTestCases, tc)
		expandedTestCases = append(expandedTestCases, struct {
			name      string
			input     string
			wantErr   bool
			wantValue any
		}{
			name:      tc.name + " without spaces",
			input:     strings.ReplaceAll(tc.input, " ", ""),
			wantErr:   tc.wantErr,
			wantValue: tc.wantValue,
		})
	}

	for _, tc := range expandedTestCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value, err := input.ParseIntervalPart(tc.input)
			if err == nil && tc.wantErr {
				t.Fatalf("should fail, got: %v", value)
			}
			if err != nil && !tc.wantErr {
				t.Fatalf("should not fail, got %v", err)
			}
			if diff := cmp.Diff(tc.wantValue, value); diff != "" {
				t.Fatalf("Mismatch (- want, + have):\n%s", diff)
			}
		})
	}
}

func TestParseInterval(t *testing.T) {
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
			wantErr:   false,
			wantStart: 123,
			wantEnd:   456,
		},
		{
			name:      "sequence number and keyword with slash",
			input:     "123/now",
			wantErr:   false,
			wantStart: 123,
			wantEnd:   input.NowKeyword,
		},
		{
			name:      "two sequence numbers with two hyphens",
			input:     "123--456",
			wantErr:   false,
			wantStart: 123,
			wantEnd:   456,
		},
		{
			name:    "now at start",
			input:   "now/456",
			wantErr: true,
		},
		{
			name:    "earliest at end",
			input:   "123/earliest",
			wantErr: true,
		},
		{
			name:    "two durations",
			input:   "1h/2h",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			start, end, err := input.ParseInterval(tc.input)
			if err == nil && tc.wantErr {
				t.Fatalf("should fail, got: start '%v', end '%v'", start, end)
			}
			if err != nil && !tc.wantErr {
				t.Fatalf("should not fail, got %v", err)
			}
			if diff := cmp.Diff(tc.wantStart, start); diff != "" {
				t.Fatalf("mismatch (- want, + have):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantEnd, end); diff != "" {
				t.Fatalf("mismatch (- want, + have):\n%s", diff)
			}
		})
	}
}
