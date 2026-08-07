package mpd_test

import (
	"testing"
	"time"

	"github.com/xymaxim/ypb/internal/mpd"
	"github.com/xymaxim/ypb/internal/testutil"
)

func TestComposeDynamic(t *testing.T) {
	opts := mpd.DynamicOptions{
		CommonOptions:         testutil.SampleCommonOptions(),
		AvailabilityStartTime: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		TimeShiftBufferDepth:  30 * time.Second,
	}

	out, err := mpd.ComposeDynamic(opts, testutil.SampleVideoInfo())
	if err != nil {
		t.Fatalf("ComposeDynamic returned error: %v", err)
	}

	testutil.AssertGolden(t, out, "testdata/dynamic.golden.xml")
}
