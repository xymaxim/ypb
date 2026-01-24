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

func MakeSegmentMetadataHandler(
	t *testing.T,
	data map[playback.SequenceNumber]*segment.Metadata,
) func(w http.ResponseWriter, r *http.Request) {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		sq, err := strconv.Atoi(urlutil.ExtractParameter(r.URL.RawPath, "sq"))
		if err != nil {
			t.Fatalf("parsing sq in request URL: %v", err)
		}
		testMetadata := data[sq]
		if testMetadata == nil {
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
Ingestion-Walltime-Us: %d`,
		sq,
		ingestionWalltime.UnixMicro(),
	)
}
