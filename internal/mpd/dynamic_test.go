package mpd_test

import (
	"testing"
	"time"

	"github.com/xymaxim/ypb/internal/mpd"
	"github.com/xymaxim/ypb/internal/testutil"
)

func TestComposeDynamic(t *testing.T) {
	opts := mpd.DynamicOptions{
		CommonOptions:              testutil.SampleCommonOptions(),
		AvailabilityStartTime:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		TimeShiftBufferDepth:       30 * time.Second,
		SuggestedPresentationDelay: 10 * time.Second,
		MinimumUpdatePeriod:        30 * time.Second,
		PublishTime:                time.Date(2026, 1, 2, 3, 4, 6, 0, time.UTC),
		SegmentRepeatCount:         4,
	}

	out, err := mpd.ComposeDynamic(opts, testutil.SampleVideoInfo())
	if err != nil {
		t.Fatalf("ComposeDynamic returned error: %v", err)
	}

	testutil.AssertGolden(t, out, "testdata/dynamic.golden.xml")
}

func TestComposeDynamicLocation(t *testing.T) {
	opts := mpd.DynamicOptions{
		CommonOptions:              testutil.SampleCommonOptions(),
		Location:                   "http://localhost:9000/mpd/123",
		AvailabilityStartTime:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		TimeShiftBufferDepth:       30 * time.Second,
		SuggestedPresentationDelay: 10 * time.Second,
		MinimumUpdatePeriod:        30 * time.Second,
		PublishTime:                time.Date(2026, 1, 2, 3, 4, 6, 0, time.UTC),
		SegmentRepeatCount:         4,
	}

	out, err := mpd.ComposeDynamic(opts, testutil.SampleVideoInfo())
	if err != nil {
		t.Fatalf("ComposeDynamic returned error: %v", err)
	}

	testutil.AssertGolden(t, out, "testdata/dynamic.location.golden.xml")
}
