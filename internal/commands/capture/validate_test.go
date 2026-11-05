package capture_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xymaxim/ypb/internal/commands"
	"github.com/xymaxim/ypb/internal/commands/capture"
)

// Frame and Timelapse have their own Validate, which hides CommonFlags.Validate.
func TestCaptureValidateChecksHost(t *testing.T) {
	t.Parallel()

	flags := commands.CommonFlags{Host: "127.0.0.1:9000"}
	require.ErrorContains(t, (&capture.Frame{CommonFlags: flags}).Validate(), "use --port")
	require.ErrorContains(t, (&capture.Timelapse{CommonFlags: flags}).Validate(), "use --port")
}
