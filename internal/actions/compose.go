package actions

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/internal/mpd"
	"github.com/xymaxim/ypb/playback"
)

func ComposeStatic(
	pb playback.Playbacker,
	interval *playback.RewindInterval,
	baseURL string,
	runner exec.Runner,
) ([]byte, error) {
	startNumber := interval.Start.Metadata.SequenceNumber

	audioPTS, videoPTS, err := probeAudioVideoPTS(pb, startNumber, runner)
	if err != nil {
		return nil, err
	}

	out, err := mpd.ComposeStatic(mpd.StaticOptions{
		CommonOptions: mpd.CommonOptions{
			BaseURL:         baseURL,
			StartNumber:     startNumber,
			SegmentDuration: pb.Info().SegmentDuration,
			AudioPTS:        audioPTS,
			VideoPTS:        videoPTS,
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

	audioPTS, videoPTS, err := probeAudioVideoPTS(pb, startNumber, runner)
	if err != nil {
		return nil, err
	}

	out, err := mpd.ComposeDynamic(mpd.DynamicOptions{
		CommonOptions: mpd.CommonOptions{
			BaseURL:         baseURL,
			StartNumber:     startNumber,
			SegmentDuration: pb.Info().SegmentDuration,
			AudioPTS:        audioPTS,
			VideoPTS:        videoPTS,
		},
		AvailabilityStartTime:      time.Now().UTC(),
		TimeShiftBufferDepth:       1 * time.Hour,
		SuggestedPresentationDelay: 10 * time.Second,
	}, pb.Info())
	if err != nil {
		return nil, fmt.Errorf("composing mpd: %w", err)
	}
	return []byte(out), nil
}

func probeAudioVideoPTS(
	pb playback.Playbacker,
	sequenceNumber int,
	runner exec.Runner,
) (int64, int64, error) {
	info := pb.Info()
	if len(info.AudioStreams) == 0 {
		return 0, 0, errors.New("no audio streams available")
	}
	if len(info.VideoStreams) == 0 {
		return 0, 0, errors.New("no video streams available")
	}

	audioPTS, err := probeSegmentPTS(pb, info.AudioStreams[0].Itag, sequenceNumber, runner, mpd.ManifestTimescale)
	if err != nil {
		return 0, 0, fmt.Errorf("probing audio pts: %w", err)
	}

	videoPTS, err := probeSegmentPTS(pb, info.VideoStreams[0].Itag, sequenceNumber, runner, mpd.ManifestTimescale)
	if err != nil {
		return 0, 0, fmt.Errorf("probing video pts: %w", err)
	}

	return audioPTS, videoPTS, nil
}

func probeSegmentPTS(
	pb playback.Playbacker,
	itag string,
	sequenceNumber int,
	runner exec.Runner,
	manifestTimescale int64,
) (int64, error) {
	var buf bytes.Buffer
	if err := pb.StreamSegment(itag, sequenceNumber, &buf); err != nil {
		return 0, fmt.Errorf("downloading probe segment: %w", err)
	}

	result, err := runner.RunWith(context.Background(), []exec.Option{
		exec.WithQuiet(),
		exec.WithStdin(bytes.NewReader(buf.Bytes())),
	},
		"-v", "quiet",
		"-i", "pipe:0",
		"-show_entries", "stream=time_base:packet=pts",
		"-read_intervals", "%+#1",
		"-of", "csv=p=0",
	)
	if err != nil {
		return 0, fmt.Errorf("ffprobe: probing segment: %w (stderr: %s)", err, result.Stderr)
	}

	var pts, tbNum, tbDen int64
	var havePTS, haveTimeBase bool

	for _, line := range strings.Split(strings.TrimSpace(string(result.Stdout)), "\n") {
		line = strings.TrimSpace(line)

		if tb, ok := strings.CutSuffix(line, ""); ok && strings.Contains(tb, "/") {
			parts := strings.SplitN(tb, "/", 2)
			tbNum, err = strconv.ParseInt(parts[0], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("parsing time_base numerator %q: %w", line, err)
			}
			tbDen, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("parsing time_base denominator %q: %w", line, err)
			}
			haveTimeBase = true
			continue
		}

		pts, err = strconv.ParseInt(line, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parsing pts %q: %w", line, err)
		}
		havePTS = true
	}

	if !haveTimeBase || !havePTS {
		return 0, fmt.Errorf("incomplete ffprobe output: %q", result.Stdout)
	}
	if tbDen == 0 {
		return 0, fmt.Errorf("time_base denominator is zero: %q", result.Stdout)
	}

	// Floor division: pts * tbNum * manifestTimescale / tbDen.
	// Never rounds up, so PTO never lands after the true sample time.
	return pts * tbNum * manifestTimescale / tbDen, nil
}
