package testutil

import (
	"flag"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// updateGolden: to update golden files run `go test ./... -update`.
var updateGolden = flag.Bool("update", false, "update golden test files")

func AssertGolden(t *testing.T, got, path string) {
	t.Helper()

	if *updateGolden {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("updating golden file %s: %v", path, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden file %s: %v (run with -update to create it)", path, err)
	}

	if diff := cmp.Diff(string(want), got); diff != "" {
		t.Errorf("output does not match %s (-want +got):\n%s", path, diff)
	}
}
