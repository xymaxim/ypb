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

type LatencyFlag struct {
	Latency float64 `help:"Streaming latency (in seconds)" short:"l" default:"0"`
}

type YtdlpOptionsFlag struct {
	YtdlpOptions []string `arg:"" help:"Options to pass to yt-dlp (use after --)" optional:"" passthrough:""` //nolint:lll
}

// ResolvePinnedTime resolves the --now flag value.
func ResolvePinnedTime(nowFlag string, startup time.Time) (time.Time, error) {
	if nowFlag == "" {
		return startup, nil
	}

	v, err := input.ParseIntervalPart(nowFlag, &startup)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing --now value: %w", err)
	}

	switch t := v.(type) {
	case time.Time:
		return t, nil
	case input.MomentKeyword:
		if t == input.NowKeyword {
			return startup, nil
		}
		return time.Time{}, fmt.Errorf("unsupported keyword '%s' for --now", t)
	default:
		return time.Time{}, fmt.Errorf(
			"--now value must be a time, not a duration or expression",
		)
	}
}

func NormalizeYtdlpOptions(opts []string) []string {
	if len(opts) > 0 && opts[0] == "--" {
		return opts[1:]
	}
	return opts
}

func ToLatencyDuration(latency float64) time.Duration {
	return time.Duration(latency * float64(time.Second))
}

func ValidateLatency(latency float64) error {
	if latency < 0 {
		return fmt.Errorf(
			"latency must be a non-negative number of seconds, got %g",
			latency,
		)
	}
	return nil
}

func FormatDifference(diff time.Duration, showPlus bool) string {
	sign := ""
	if diff > 0 && showPlus {
		sign = "+"
	}
	return sign + actions.FormatDuration(diff)
}

func checkYtdlp() error {
	_, err := osexec.LookPath(apppkg.YtdlpBinaryPath)
	if err != nil {
		return fmt.Errorf("unable to find yt-dlp: %w", err)
	}
	return nil
}
