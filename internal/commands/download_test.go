package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatDifference(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		diff     time.Duration
		showPlus bool
		expected string
	}{
		{
			name:     "+1s",
			diff:     time.Second,
			showPlus: true,
			expected: "+1s",
		},
		{
			name:     "1s",
			diff:     time.Second,
			showPlus: false,
			expected: "1s",
		},
		{
			name:     "-1s",
			diff:     -time.Second,
			showPlus: true,
			expected: "-1s",
		},
		{
			name:     "0s",
			diff:     time.Duration(0),
			showPlus: true,
			expected: "0s",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, FormatDifference(tc.diff, tc.showPlus))
		})
	}
}
