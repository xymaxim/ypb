package download_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xymaxim/ypb/internal/commands"
	"github.com/xymaxim/ypb/internal/commands/download"
)

// Download has its own Validate, which hides CommonFlags.Validate.
func TestDownloadValidateChecksHost(t *testing.T) {
	t.Parallel()

	c := &download.Download{CommonFlags: commands.CommonFlags{Host: "127.0.0.1:9000"}}
	require.ErrorContains(t, c.Validate(), "use --port")
}
