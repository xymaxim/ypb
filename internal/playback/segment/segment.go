package segment

import (
	"bytes"
	"fmt"
	"strconv"
	"time"
)

// MetadataLength is the size, in bytes, used for a segment file's metadata.
const MetadataLength int64 = 2000

// Metadata contains metadata for a segment.
type Metadata struct {
	SequenceNumber    int
	IngestionWalltime time.Time
	Duration          time.Duration
}

type parser[T any] func(string) (T, error)

// Time returns a timestamp associated with a segment.
func (m *Metadata) Time() time.Time {
	return m.IngestionWalltime
}

func ParseMetadata(b []byte) (*Metadata, error) {
	var m Metadata
	var err error

	m.SequenceNumber, err = extractAndParse(b, "Sequence-Number",
		func(s string) (int, error) { return strconv.Atoi(s) })
	if err != nil {
		return nil, err
	}

	ingestionWalltimeUs, err := extractAndParse(b, "Ingestion-Walltime-Us",
		func(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) })
	if err != nil {
		return nil, err
	}
	m.IngestionWalltime = time.Unix(0, ingestionWalltimeUs*1e3).In(time.UTC)

	durationUs, err := extractAndParse(b, "Target-Duration-Us",
		func(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) })
	if err != nil {
		return nil, err
	}
	m.Duration = time.Duration(durationUs) * time.Microsecond

	return &m, nil
}

func extractAndParse[T any](b []byte, field string, parse parser[T]) (T, error) {
	var zero T
	raw, err := extractMetadataField(b, field)
	if err != nil {
		return zero, err
	}
	value, err := parse(raw)
	if err != nil {
		return zero, fmt.Errorf("converting '%s': %w", field, err)
	}
	return value, nil
}

// extractMetadataField extracts the value of a metadata field from b. Accepts
// both CRLF (\r\n) and LF (\n) line endings.
func extractMetadataField(b []byte, field string) (string, error) {
	token := []byte(field + ": ")
	index := bytes.Index(b, token)
	if index == -1 {
		return "", fmt.Errorf("field '%s' not present", field)
	}

	var valueBytes []byte
	valueStart := index + len(token)
	lineEndRel := bytes.IndexByte(b[valueStart:], '\n')
	if lineEndRel == -1 {
		valueBytes = b[valueStart:]
	} else {
		valueBytes = b[valueStart : valueStart+lineEndRel]
	}

	return string(bytes.TrimRight(valueBytes, "\r")), nil
}
