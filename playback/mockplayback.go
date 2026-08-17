package playback

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Eyevinn/mp4ff/mp4"

	"github.com/xymaxim/ypb/info"
	"github.com/xymaxim/ypb/segment"
)

var _ Playbacker = (*MockPlayback)(nil)

// MockPlayback is a fake Playbacker used by internal/mockserver and, via that,
// cmd/mockplay to drive the web player without hitting a live YouTube stream.
type MockPlayback struct {
	dir                string
	info               info.VideoInformation
	baseURLs           map[string]string
	timescales         map[string]uint64
	segmentCount       int
	availabilityWindow time.Duration
	patchedCache       sync.Map
}

const defaultAvailabilityWindow = 7 * 24 * time.Hour

func NewMockPlayback(dir string, actualStartTime time.Time) (*MockPlayback, error) {
	infoPath := filepath.Join(dir, "info.json")
	raw, err := os.ReadFile(infoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(
				"fixture info not found at %s",
				infoPath,
			)
		}
		return nil, fmt.Errorf("reading fixture info: %w", err)
	}

	var vi info.VideoInformation
	if err := json.Unmarshal(raw, &vi); err != nil {
		return nil, fmt.Errorf("parsing fixture info: %w", err)
	}

	if !actualStartTime.IsZero() {
		vi.ActualStartTime = actualStartTime
	}

	segmentsDir := filepath.Join(dir, "segments")
	if _, err := os.Stat(segmentsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf(
			"fixture segments not found at %s",
			segmentsDir,
		)
	}

	baseURLs := make(map[string]string)
	for _, s := range vi.AudioStreams {
		baseURLs[s.Itag] = "mock://" + s.Itag
	}
	for _, s := range vi.VideoStreams {
		baseURLs[s.Itag] = "mock://" + s.Itag
	}

	m := &MockPlayback{
		dir:                dir,
		info:               vi,
		baseURLs:           baseURLs,
		timescales:         make(map[string]uint64),
		availabilityWindow: defaultAvailabilityWindow,
	}

	n, err := m.countPhysicalSegments(m.ProbeItag())
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, fmt.Errorf("no fixture segments for itag %q", m.ProbeItag())
	}
	m.segmentCount = n

	for itag := range baseURLs {
		ts, err := m.readTimescale(itag)
		if err != nil {
			return nil, fmt.Errorf("reading timescale for itag %q: %w", itag, err)
		}
		m.timescales[itag] = ts
	}

	return m, nil
}

func (m *MockPlayback) Info() info.VideoInformation {
	return m.info
}

func (m *MockPlayback) ProbeItag() string {
	return m.info.VideoStreams[0].Itag
}

func (m *MockPlayback) BaseURLs() map[string]string {
	return m.baseURLs
}

func (m *MockPlayback) RefreshBaseURLs() error {
	return nil
}

func (m *MockPlayback) RequestHeadSeqNum() (int, error) {
	elapsed := time.Since(m.info.ActualStartTime)
	sq := int(elapsed/m.info.SegmentDuration) + 1
	if sq < 1 {
		sq = 1
	}
	return sq, nil
}

func (m *MockPlayback) StreamSegment(itag string, sq SequenceNumber, w io.Writer) error {
	if err := m.checkAvailability(sq); err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("%s:%d", itag, sq)
	if cached, ok := m.patchedCache.Load(cacheKey); ok {
		_, err := w.Write(cached.([]byte))
		return err
	}

	idx := m.physicalIndex(sq)
	raw, err := os.ReadFile(m.physicalSegmentPath(itag, idx))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no fixture segment for itag=%s sq=%d", itag, sq)
		}
		return fmt.Errorf("reading fixture segment: %w", err)
	}

	targetTicks := uint64(sq-1) * m.segmentDurationTicks(itag)
	patched, err := patchSegmentTfdt(raw, targetTicks)
	if err != nil {
		return fmt.Errorf("patching segment pts: %w", err)
	}

	m.patchedCache.Store(cacheKey, patched)

	_, err = w.Write(patched)
	return err
}

func (m *MockPlayback) FetchSegmentMetadata(
	itag string,
	sq SequenceNumber,
) (*segment.Metadata, error) {
	if err := m.checkAvailability(sq); err != nil {
		return nil, err
	}

	idx := m.physicalIndex(sq)
	if _, err := os.Stat(m.physicalSegmentPath(itag, idx)); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no fixture segment for itag=%s sq=%d", itag, sq)
		}
		return nil, fmt.Errorf("checking fixture segment: %w", err)
	}

	return &segment.Metadata{
		SequenceNumber:    sq,
		IngestionWalltime: m.walltimeFor(sq),
		Duration:          m.info.SegmentDuration,
	}, nil
}

func (m *MockPlayback) LocateMoment(
	t time.Time,
	sm segment.Metadata,
	isEnd bool,
) (*RewindMoment, error) {
	return LocateMomentFor(m, t, sm, isEnd)
}

func (m *MockPlayback) physicalIndex(sq SequenceNumber) SequenceNumber {
	walltime := m.walltimeFor(sq)
	secondsIntoHour := walltime.Minute()*60 + walltime.Second()
	return secondsIntoHour/int(m.info.SegmentDuration.Seconds()) + 1
}

func (m *MockPlayback) physicalSegmentPath(itag string, idx SequenceNumber) string {
	return filepath.Join(m.dir, "segments", itag, fmt.Sprintf("%d.mp4", idx))
}

func (m *MockPlayback) countPhysicalSegments(itag string) (int, error) {
	entries, err := os.ReadDir(filepath.Join(m.dir, "segments", itag))
	if err != nil {
		return 0, fmt.Errorf("reading fixture segments dir: %w", err)
	}
	return len(entries), nil
}

func (m *MockPlayback) walltimeFor(sq SequenceNumber) time.Time {
	alignedStart := m.info.ActualStartTime.Truncate(m.info.SegmentDuration)
	alignedElapsed := time.Duration(sq-1) * m.info.SegmentDuration
	return alignedStart.Add(alignedElapsed)
}

func (m *MockPlayback) checkAvailability(sq SequenceNumber) error {
	walltime := m.walltimeFor(sq)
	cutoff := time.Now().Add(-m.availabilityWindow)
	if walltime.Before(cutoff) {
		return fmt.Errorf(
			"segment sq=%d (%s) is outside the %s availability window",
			sq,
			walltime.Format(time.RFC3339),
			m.availabilityWindow,
		)
	}
	return nil
}

func (m *MockPlayback) readTimescale(itag string) (uint64, error) {
	raw, err := os.ReadFile(m.physicalSegmentPath(itag, 1))
	if err != nil {
		return 0, fmt.Errorf("reading reference segment: %w", err)
	}
	f, err := mp4.DecodeFile(bytes.NewReader(raw))
	if err != nil {
		return 0, fmt.Errorf("decoding reference segment: %w", err)
	}
	trak := f.Moov.Traks[0]
	return uint64(trak.Mdia.Mdhd.Timescale), nil
}

func (m *MockPlayback) segmentDurationTicks(itag string) uint64 {
	return uint64(m.info.SegmentDuration) * m.timescales[itag] / uint64(time.Second)
}

func patchSegmentTfdt(raw []byte, targetTicks uint64) ([]byte, error) {
	patched := make([]byte, len(raw))
	copy(patched, raw)

	offset := bytes.Index(patched, []byte("tfdt"))
	if offset < 0 {
		return nil, errors.New("tfdt box not found")
	}

	// offset points at the 4-byte "tfdt" fourcc. The box's size field is
	// the 4 bytes immediately before it. Version/flags is the 4 bytes
	// immediately after.
	versionByte := patched[offset+4]

	if versionByte == 0 {
		// 4-byte baseMediaDecodeTime, starts after fourcc(4) + version/flags(4)
		binary.BigEndian.PutUint32(patched[offset+8:], uint32(targetTicks))
	} else {
		// 8-byte baseMediaDecodeTime
		binary.BigEndian.PutUint64(patched[offset+8:], targetTicks)
	}

	return patched, nil
}
