package version

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// GitVersion is a version as described by Git (passed in at buid via -ldflags).
var GitVersion string

const revisionLength = 7

func GetFull() string {
	info, ok := debug.ReadBuildInfo()
	if ok { //nolint:nestif
		var sb strings.Builder

		var (
			arch     string
			platform string
			revision string
			modified bool
		)

		for _, kv := range info.Settings {
			switch kv.Key {
			case "GOOS":
				platform = kv.Value
			case "GOARCH":
				arch = kv.Value
			case "vcs.revision":
				revision = kv.Value[:revisionLength]
			case "vcs.modified":
				modified = kv.Value == "true"
			}
		}

		// Write Git version
		var version = "(untagged)"
		if GitVersion != "" {
			version = buildVersionNumber(GitVersion, modified)
		}
		sb.WriteString("ypb version " + version)

		// Write revision
		if revision != "" {
			sb.WriteString(" from " + revision)
		}

		// Write Go version
		sb.WriteString(" with " + info.GoVersion)

		// Write platform and arch
		sb.WriteString(fmt.Sprintf(" on %s/%s", platform, arch))

		return sb.String()
	}

	return ""
}

func GetShort() string {
	info, ok := debug.ReadBuildInfo()
	if ok {
		var modified bool
		for _, kv := range info.Settings {
			switch kv.Key {
			case "vcs.modified":
				modified = kv.Value == "true"
				break
			}
		}
		return buildVersionNumber(GitVersion, modified)
	}
	return ""
}

func buildVersionNumber(v string, dirty bool) string {
	if v != "" && dirty {
		return v + "+dirty"
	}
	return v
}
