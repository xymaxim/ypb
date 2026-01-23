package pathutil

import (
	"strings"
	"time"

	"github.com/gosimple/slug"
)

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
	return t.Format("20060102T030405-07")
}

func FormatDuration(d time.Duration) string {
	s := d.Truncate(time.Second).String()
	s = strings.ReplaceAll(s, "m0s", "m")
	s = strings.ReplaceAll(s, "h0m", "h")
	return s
}
