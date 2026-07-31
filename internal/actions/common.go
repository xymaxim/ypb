package actions

import (
	"fmt"
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
	return t.Format("20060102T150405-07")
}

func FormatDuration(d time.Duration) string {
	s := d.Truncate(time.Second).String()
	s = strings.ReplaceAll(s, "m0s", "m")
	s = strings.ReplaceAll(s, "h0m", "h")
	return s
}

// BuildOutputStem builds the output file stem (without extension) for a
// located interval.
func BuildOutputStem(ctx *LocateOutputContext) string {
	return fmt.Sprintf(
		"%s_%s_%s_%s",
		AdjustForFilename(ctx.Title, 0),
		ctx.ID,
		FormatTime(ctx.InputStartTime),
		FormatDuration(ctx.InputDuration),
	)
}
