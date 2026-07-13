package mpd_test

import (
	"testing"
	"time"

	"github.com/xymaxim/ypb/internal/mpd"
	"github.com/xymaxim/ypb/internal/testutil"
)

func TestComposeStatic(t *testing.T) {
	opts := mpd.StaticOptions{
		CommonOptions: testutil.SampleCommonOptions(),
		MediaDuration: 30*time.Minute + 5*time.Second,
		SegmentCount:  361,
	}

	out, err := mpd.ComposeStatic(opts, testutil.SampleVideoInfo())
	if err != nil {
		t.Fatalf("ComposeStatic returned error: %v", err)
	}

	testutil.AssertGolden(t, out, "testdata/static.golden.xml")
}
