package app_test

import (
	"net"
	"testing"

	"github.com/xymaxim/ypb/internal/app"
)

const testHost = "127.0.0.1"

// maxPortAttempts mirrors app.maxPortAttempts.
const maxPortAttempts = 10

func portOf(ln net.Listener) int {
	return ln.Addr().(*net.TCPAddr).Port
}

// listen calls app.Listen and closes the listener when the test ends.
func listen(t *testing.T, port int) net.Listener {
	t.Helper()

	ln, err := app.Listen(testHost, port)
	if err != nil {
		t.Fatalf("Listen(%q, %d) = %v, want a listener", testHost, port, err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	return ln
}

func TestListenSkipsPortInUse(t *testing.T) {
	t.Parallel()

	busyPort := portOf(listen(t, 0))

	if got := portOf(listen(t, busyPort)); got <= busyPort || got > busyPort+maxPortAttempts-1 {
		t.Fatalf(
			"Listen(%q, %d) bound port %d, want a port in (%d, %d]",
			testHost,
			busyPort,
			got,
			busyPort,
			busyPort+maxPortAttempts-1,
		)
	}
}

// Not parallel, so no other test in this package can claim the port between
// probe and Listen. It retries because another process still could.
//
//nolint:paralleltest // Must stay serial so this package cannot claim the probed port.
func TestListenUsesRequestedPort(t *testing.T) {
	for range 3 {
		probe, err := net.Listen("tcp", testHost+":0")
		if err != nil {
			t.Fatalf("probing for a free port: %v", err)
		}
		port := portOf(probe)
		_ = probe.Close()

		if portOf(listen(t, port)) == port {
			return
		}
	}

	t.Fatal("Listen kept binding a different port for a free requested port")
}

func TestListenZeroPort(t *testing.T) {
	t.Parallel()

	if got := portOf(listen(t, 0)); got == 0 {
		t.Fatal("Listen bound port 0, want an ephemeral port")
	}
}

func TestListenInvalidPort(t *testing.T) {
	t.Parallel()

	for _, port := range []int{-1, 65536} {
		if ln, err := app.Listen(testHost, port); err == nil {
			_ = ln.Close()
			t.Fatalf("Listen(%q, %d) = nil error, want an error", testHost, port)
		}
	}
}
