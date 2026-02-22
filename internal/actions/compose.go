package actions

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/internal/mpd"
	"github.com/xymaxim/ypb/internal/playback"
)

const segmentMediaURL = "videoplayback/itag/$RepresentationID$/sq/$Number$"

func ComposeStatic(
	pb playback.Playbacker,
	interval *playback.RewindInterval,
	baseURL string,
	runner exec.Runner,
) ([]byte, error) {
	segmentDuration := pb.Info().SegmentDuration

	var buf bytes.Buffer
	err := pb.StreamSegment(pb.ProbeItag(), interval.Start.Metadata.SequenceNumber, &buf)
	if err != nil {
		return nil, fmt.Errorf("downloading probe segment: %w", err)
	}

	presentationTimestamp, err := extractPresentationTimestamp(&buf, runner)
	if err != nil {
		return nil, fmt.Errorf("extracting presentation timestamp: %w", err)
	}
	presentationTimeOffset := presentationTimestamp * float64(time.Second.Milliseconds())

	mpdInfo := mpd.Information{
		MediaPresentationDuration: interval.Duration(),
		RepresentationBaseURL:     baseURL,
		SegmentTemplate: &mpd.SegmentTemplate{
			Media:                  segmentMediaURL,
			StartNumber:            interval.Start.Metadata.SequenceNumber,
			Timescale:              fmt.Sprintf("%d", time.Second.Milliseconds()),
			Duration:               fmt.Sprintf("%d", segmentDuration.Milliseconds()),
			PresentationTimeOffset: fmt.Sprintf("%.0f", presentationTimeOffset),
		},
	}

	out := mpd.ComposeStaticMPD(mpdInfo, pb.Info())

	return []byte(out), nil
}

func extractPresentationTimestamp(buf *bytes.Buffer, runner exec.Runner) (float64, error) {
	result, err := runner.RunWith(
		[]exec.Option{
			exec.WithQuiet(),
			exec.WithStdin(bytes.NewReader(buf.Bytes())),
		},
		"-v", "quiet",
		"-i", "pipe:0",
		"-show_entries", "packet=pts_time",
		"-read_intervals", "%+#1",
		"-of", "default=noprint_wrappers=1:nokey=1",
	)
	if err != nil {
		return 0, fmt.Errorf(
			"ffprobe: probing segment: %w (stderr: %s)",
			err,
			result.Stderr,
		)
	}

	timestampString := strings.TrimRight(string(result.Stdout), "\r\n")
	timestamp, err := strconv.ParseFloat(timestampString, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing timestamp: %w", err)
	}

	return timestamp, nil
}
