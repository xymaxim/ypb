package commands

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/gosimple/slug"

	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/urlutil"
)

type CommonFlags struct {
	Port int `help:"Port to start playback on" short:"p" default:"8080"`
}

func checkYtdlp() error {
	_, err := exec.LookPath(apppkg.YtdlpBinaryPath)
	if err != nil {
		return fmt.Errorf("unable to find yt-dlp: %w", err)
	}
	return nil
}

func CollectVideoInfo(id string, app *apppkg.App, port int) error {
	url := urlutil.BuildVideoLiveURL(id)

	fmt.Printf("(<<) Collecting info about %s...\n", url)
	cfg := &apppkg.Config{Port: port}
	if err := app.Initialize(id, cfg); err != nil {
		return fmt.Errorf("initializing app: %w", err)
	}

	fmt.Printf("Stream '%s' is alive!\n", app.Playback.Info().Title)

	return nil
}

func AdjustForFilename(s string, length int) string {
	const maxAdjustedLength = 30

	if length == 0 {
		length = maxAdjustedLength
	}

	slug.MaxLength = length
	slug.Lowercase = false

	return slug.Make(s)
}

func FormatTime(t time.Time) string {
	return t.Format("20060102T150405-07")
}

func FormatDuration(d time.Duration) string {
	s := d.Truncate(time.Second).String()
	s = strings.ReplaceAll(s, "m0s", "m")
	s = strings.ReplaceAll(s, "h0m", "h")
	return s
}

func FormatDifference(diff time.Duration, showPlus bool) string {
	sign := ""
	if diff > 0 && showPlus {
		sign = "+"
	}
	return sign + FormatDuration(diff)
}
