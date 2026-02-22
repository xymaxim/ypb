package actions

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/internal/mpd"
	"github.com/xymaxim/ypb/internal/playback"
)

func ComposeStatic(
	pb playback.Playbacker,
	interval *playback.RewindInterval,
	baseURL string,
	runner exec.Runner,
) ([]byte, error) {
	startNumber := interval.Start.Metadata.SequenceNumber

	pts, err := probeSegmentPTS(pb, startNumber, runner)
	if err != nil {
		return nil, fmt.Errorf("extracting pts: %w", err)
	}

	out, err := mpd.ComposeStatic(mpd.StaticOptions{
		CommonOptions: mpd.CommonOptions{
			BaseURL:         baseURL,
			StartNumber:     startNumber,
			SegmentDuration: pb.Info().SegmentDuration,
			PTS:             pts,
		},
		MediaDuration: interval.Duration(),
		SegmentCount:  interval.End.Metadata.SequenceNumber - startNumber + 1,
	}, pb.Info())
	if err != nil {
		return nil, fmt.Errorf("composing mpd: %w", err)
	}

	return []byte(out), nil
}

func ComposeDynamic(
	pb playback.Playbacker,
	moment *playback.RewindMoment,
	baseURL string,
	runner exec.Runner,
) ([]byte, error) {
	startNumber := moment.Metadata.SequenceNumber

	pts, err := probeSegmentPTS(pb, startNumber, runner)
	if err != nil {
		return nil, fmt.Errorf("extracting pts: %w", err)
	}

	out, err := mpd.ComposeDynamic(mpd.DynamicOptions{
		CommonOptions: mpd.CommonOptions{
			BaseURL:         baseURL,
			StartNumber:     startNumber,
			SegmentDuration: pb.Info().SegmentDuration,
			PTS:             pts,
		},
		AvailabilityStartTime: moment.ActualTime,
	}, pb.Info())
	if err != nil {
		return nil, fmt.Errorf("composing mpd: %w", err)
	}
	return []byte(out), nil
}

func probeSegmentPTS(
	pb playback.Playbacker,
	sequenceNumber int,
	runner exec.Runner,
) (float64, error) {
	var buf bytes.Buffer
	if err := pb.StreamSegment(pb.ProbeItag(), sequenceNumber, &buf); err != nil {
		return 0, fmt.Errorf("downloading probe segment: %w", err)
	}

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

	ptsString := strings.TrimRight(string(result.Stdout), "\r\n")
	pts, err := strconv.ParseFloat(ptsString, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing pts: %w", err)
	}

	return pts, nil
}
