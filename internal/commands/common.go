package commands

import (
	"fmt"
	"os/exec"

	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/urlutil"
)

func checkYtdlp(path string) error {
	_, err := exec.LookPath(path)
	if err != nil {
		return fmt.Errorf("unable to find yt-dlp: %w", err)
	}
	return nil
}

func collectVideoInfo(id string, app *apppkg.App, port int) error {
	url := urlutil.BuildVideoLiveURL(id)

	fmt.Printf("(<<) Collecting info about %s...\n", url)
	cfg := &apppkg.Config{Port: port}
	if err := app.Initialize(id, cfg); err != nil {
		return fmt.Errorf("initializing app: %w", err)
	}

	fmt.Printf("Stream '%s' is alive!\n", app.Playback.Info().Title)

	return nil
}
