package testutil

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/xymaxim/ypb/internal/playback"
	"github.com/xymaxim/ypb/internal/playback/segment"
	"github.com/xymaxim/ypb/internal/urlutil"
)

type MetadataMap = map[playback.SequenceNumber]segment.Metadata

func GenerateFakeSegmentMetadata(count int, duration time.Duration) MetadataMap {
	t := time.Date(2026, 1, 2, 10, 20, 30, 0, time.UTC)
	out := make(MetadataMap, count)
	for i := range count {
		out[i] = segment.Metadata{
			SequenceNumber:    i,
			IngestionWalltime: t.Add(time.Duration(i) * duration),
			Duration:          duration,
		}
	}
	return out
}

func MakeSegmentMetadataHandler(
	t *testing.T,
	data MetadataMap,
) func(w http.ResponseWriter, r *http.Request) {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		sq, err := strconv.Atoi(urlutil.ExtractParameter(r.URL.EscapedPath(), "sq"))
		if err != nil {
			t.Fatalf("parsing sq in request URL: %v", err)
		}
		testMetadata, ok := data[sq]
		if !ok {
			t.Fatalf("no test metadata for sq=%d", sq)
		}
		w.Write(
			GenerateSegmentMetadataBytes(
				t,
				testMetadata.SequenceNumber,
				testMetadata.IngestionWalltime,
			),
		)
	}
}

func GenerateSegmentMetadataBytes(t *testing.T, sq int, ingestionWalltime time.Time) []byte {
	t.Helper()
	return fmt.Appendf(
		nil,
		`Sequence-Number: %d
Ingestion-Walltime-Us: %d
Target-Duration-Us: 2000000`,
		sq,
		ingestionWalltime.UnixMicro(),
	)
}
