package commands

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	osexec "os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/xymaxim/ypb/internal/actions"
	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/input"
)

// Host is a listen address with no port.
// Empty means all interfaces.
type Host string

func (h *Host) UnmarshalText(text []byte) error {
	s := string(text)
	host := s
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}
	if err := checkHost(host); err != nil {
		return fmt.Errorf("invalid host %q: %w", s, err)
	}
	*h = Host(host)
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

// Port is a TCP port number.
type Port int

func (p *Port) UnmarshalText(text []byte) error {
	n, err := strconv.Atoi(string(text))
	if err != nil {
		return fmt.Errorf("invalid port %q: must be a number", text)
	}
	if n < 0 || n > 65535 {
		return fmt.Errorf("invalid port %d: must be between 0 and 65535", n)
	}
	*p = Port(n)
	return nil
}

// PinnedTime is the raw --now value. Resolve turns it into a time.
type PinnedTime string

func (t *PinnedTime) UnmarshalText(text []byte) error {
	// Throwaway reference time, so bad input fails at parse time.
	if _, err := resolvePinnedTime(string(text), time.Now()); err != nil {
		return err
	}
	*t = PinnedTime(text)
	return nil
}

// Resolve returns the pinned time, or startup if --now wasn't set.
func (t PinnedTime) Resolve(startup time.Time) (time.Time, error) {
	return resolvePinnedTime(string(t), startup)
}

func resolvePinnedTime(s string, startup time.Time) (time.Time, error) {
	if s == "" {
		return startup, nil
	}

	v, err := input.ParseIntervalPart(s, &startup)
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
		return time.Time{}, errors.New(
			"--now value must be a time, not a duration or expression",
		)
	}
}

// Latency is streaming latency in seconds.
type Latency float64

func (l *Latency) UnmarshalText(text []byte) error {
	seconds, err := strconv.ParseFloat(string(text), 64)
	if err != nil {
		return fmt.Errorf("invalid seconds %q: %w", text, err)
	}
	if seconds < 0 {
		return fmt.Errorf("invalid seconds %q: must not be negative", text)
	}
	*l = Latency(seconds)
	return nil
}

// Duration returns the latency as a time.Duration.
func (l Latency) Duration() time.Duration {
	return time.Duration(float64(l) * float64(time.Second))
}

// ServerFlags are shared by commands that start a server.
type ServerFlags struct {
	Host Host `help:"Address to listen on (default: all interfaces)" placeholder:"ADDR" env:"YPB_HOST"`
	Port Port `help:"Port to listen on"                                                 env:"YPB_PORT" short:"p" default:"9000"` //nolint:lll
}

// NowFlag is shared by commands that depend on it.
type NowFlag struct {
	Now PinnedTime `help:"Pin now to a specific time" placeholder:"TIME" env:"YPB_NOW"`
}

type LatencyFlag struct {
	Latency Latency `help:"Streaming latency (in seconds)" short:"l" default:"0"`
}

type YtdlpOptionsFlag struct {
	YtdlpOptions []string `arg:"" help:"Options to pass to yt-dlp (use after --)" optional:"" passthrough:""`
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

// CheckYtdlp verifies that the yt-dlp binary is available.
func CheckYtdlp() error {
	_, err := osexec.LookPath(apppkg.YtdlpBinaryPath)
	if err != nil {
		return fmt.Errorf("unable to find yt-dlp: %w", err)
	}
	return nil
}
