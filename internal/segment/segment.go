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
}

// Time returns a timestamp associated with a segment.
func (m *Metadata) Time() time.Time {
	return m.IngestionWalltime
}

// ParseMetadata parses metadata bytes to a Metadata.
func ParseMetadata(b []byte) (*Metadata, error) {
	rawSequenceNumber, err := extractMetadataField(b, "Sequence-Number")
	if err != nil {
		return nil, err
	}
	sequenceNumber, err := strconv.Atoi(rawSequenceNumber)
	if err != nil {
		return nil, fmt.Errorf("converting 'Sequence-Number': %w", err)
	}

	rawIngestionWalltimeUs, err := extractMetadataField(b, "Ingestion-Walltime-Us")
	if err != nil {
		return nil, err
	}
	ingestionWalltimeUs, err := strconv.ParseInt(rawIngestionWalltimeUs, 10, 64)
	if err != nil {
		return nil, fmt.Errorf(
			"converting 'Ingestion-Walltime-Us': %w",
			err,
		)
	}

	return &Metadata{
		SequenceNumber:    sequenceNumber,
		IngestionWalltime: time.Unix(0, ingestionWalltimeUs*1e3).In(time.UTC),
	}, nil
}

func extractMetadataField(b []byte, key string) (string, error) {
	index := bytes.Index(b, []byte(key))
	if index == -1 {
		return "", fmt.Errorf("field '%s' not present", key)
	}

	valueStart := index + len(key) + 2
	lineEndRel := bytes.IndexAny(b[valueStart:], "\r\n")

	var valueBytes []byte
	if lineEndRel == -1 {
		valueBytes = b[valueStart:]
	} else {
		valueBytes = b[valueStart : valueStart+lineEndRel]
	}
	return string(valueBytes), nil
}
