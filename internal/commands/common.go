package commands

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	osexec "os/exec"
	"strings"
	"time"

	"github.com/xymaxim/ypb/internal/actions"
	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/input"
)

type CommonFlags struct {
	Host string `help:"Address to listen on (default: all interfaces)" placeholder:"ADDR" env:"YPB_HOST"`
	Port int    `help:"Port to start playback on"                                                        short:"p" default:"9000"` //nolint:lll
	Now  string `help:"Pin now to a specific time"                     placeholder:"TIME" env:"YPB_NOW"`
}

// Validate normalizes --host (accepts "[::1]" as "::1") and rejects hosts
// that include a port or are not a valid IP address.
func (c *CommonFlags) Validate() error {
	host := c.Host
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}
	if err := checkHost(host); err != nil {
		return fmt.Errorf("invalid --host %q: %w", c.Host, err)
	}
	c.Host = host
	return nil
}

func checkHost(host string) error {
	if !strings.Contains(host, ":") {
		return nil
	}
	if _, err := netip.ParseAddr(host); err == nil {
		return nil
	}
	if _, _, err := net.SplitHostPort(host); err == nil {
		return errors.New("did you include a port? use --port for that")
	}
	return errors.New("not a valid address")
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

// CheckYtdlp verifies that the yt-dlp binary is available.
func CheckYtdlp() error {
	_, err := osexec.LookPath(apppkg.YtdlpBinaryPath)
	if err != nil {
		return fmt.Errorf("unable to find yt-dlp: %w", err)
	}
	return nil
}
