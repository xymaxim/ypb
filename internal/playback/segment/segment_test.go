package segment_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xymaxim/ypb/internal/playback/segment"
)

func TestParseMetadata_Success(t *testing.T) {
	t.Parallel()
	b := `
                                       Sequence-Number: 7959120
Ingestion-Walltime-Us: 1679787234491176
Ingestion-Uncertainty-Us: 71
Target-Duration-Us: 2000000`
	expected := segment.Metadata{
		SequenceNumber:    7959120,
		IngestionWalltime: time.Unix(0, 1679787234491176*1e3).In(time.UTC),
		Duration:          2 * time.Second,
	}
	actual, _ := segment.ParseMetadata([]byte(b))
	assert.Equal(t, expected, *actual)
}

func TestParseMetadata_MissingField(t *testing.T) {
	t.Parallel()
	b := `
                                       Sequence-Number: 7959120
Ingestion-Uncertainty-Us: 71`
	_, err := segment.ParseMetadata([]byte(b))
	assert.Error(t, err, "should failed for missing 'Ingestion-Walltime-Us'")
}
