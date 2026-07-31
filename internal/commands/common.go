package commands

import (
	"fmt"
	osexec "os/exec"
	"time"

	"github.com/xymaxim/ypb/internal/actions"
	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/input"
)

type CommonFlags struct {
	Port int    `help:"Port to start playback on"  short:"p" default:"9000"`
	Now  string `help:"Pin now to a specific time"                          placeholder:"TIME" env:"YPB_NOW"`
}

type YtdlpOptionsFlag struct {
	YtdlpOptions []string `arg:"" help:"Options to pass to yt-dlp (use after --)" optional:"" passthrough:""` //nolint:lll
}

func ResolvePinnedTime(nowFlag string) (time.Time, error) {
	if nowFlag == "" {
		return time.Now().UTC(), nil
	}

	v, err := input.ParseIntervalPart(nowFlag, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing --now value: %w", err)
	}

	switch t := v.(type) {
	case time.Time:
		return t, nil
	case input.MomentKeyword:
		if t == input.NowKeyword {
			return time.Now().UTC(), nil
		}
		return time.Time{}, fmt.Errorf("unsupported keyword '%s' for --now", t)
	default:
		return time.Time{}, fmt.Errorf(
			"--now value must be a time, not a duration or expression",
		)
	}
}

func checkYtdlp() error {
	_, err := osexec.LookPath(apppkg.YtdlpBinaryPath)
	if err != nil {
		return fmt.Errorf("unable to find yt-dlp: %w", err)
	}
	return nil
}

func NormalizeYtdlpOptions(opts []string) []string {
	if len(opts) > 0 && opts[0] == "--" {
		return opts[1:]
	}
	return opts
}

func FormatDifference(diff time.Duration, showPlus bool) string {
	sign := ""
	if diff > 0 && showPlus {
		sign = "+"
	}
	return sign + actions.FormatDuration(diff)
}
