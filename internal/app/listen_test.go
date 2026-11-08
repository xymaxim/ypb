package app_test

import (
	"net"
	"testing"

	"github.com/xymaxim/ypb/internal/app"
)

// maxPortAttempts mirrors app.maxPortAttempts.
const maxPortAttempts = 10

func TestListenSkipsPortInUse(t *testing.T) {
	t.Parallel()

	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("occupying a port: %v", err)
	}
	t.Cleanup(func() { _ = busy.Close() })

	busyPort := busy.Addr().(*net.TCPAddr).Port

	ln, err := app.Listen("127.0.0.1", busyPort)
	if err != nil {
		t.Fatalf("Listen(%q, %d) = %v, want a free port", "127.0.0.1", busyPort, err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	gotPort := ln.Addr().(*net.TCPAddr).Port
	if gotPort <= busyPort || gotPort > busyPort+maxPortAttempts-1 {
		t.Fatalf(
			"Listen(%q, %d) bound port %d, want a port in (%d, %d]",
			"127.0.0.1",
			busyPort,
			gotPort,
			busyPort,
			busyPort+maxPortAttempts-1,
		)
	}
}

func TestListenUsesRequestedPort(t *testing.T) {
	t.Parallel()

	// Find a port that is free by binding it and releasing it again. Retry a
	// few times in case another process grabs the port in between.
	for range 5 {
		probe, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("probing for a free port: %v", err)
		}
		port := probe.Addr().(*net.TCPAddr).Port
		_ = probe.Close()

		ln, err := app.Listen("127.0.0.1", port)
		if err != nil {
			t.Fatalf("Listen(%q, %d) = %v, want a listener", "127.0.0.1", port, err)
		}

		gotPort := ln.Addr().(*net.TCPAddr).Port
		_ = ln.Close()
		if gotPort == port {
			return
		}
	}

	t.Fatal("Listen kept binding a different port for a free requested port")
}

func TestListenZeroPort(t *testing.T) {
	t.Parallel()

	ln, err := app.Listen("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("Listen(%q, 0) = %v, want a listener", "127.0.0.1", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	if gotPort := ln.Addr().(*net.TCPAddr).Port; gotPort == 0 {
		t.Fatal("Listen bound port 0, want an ephemeral port")
	}
}

func TestListenInvalidPort(t *testing.T) {
	t.Parallel()

	for _, port := range []int{-1, 65536} {
		if ln, err := app.Listen("127.0.0.1", port); err == nil {
			_ = ln.Close()
			t.Fatalf("Listen(%q, %d) = nil error, want an error", "127.0.0.1", port)
		}
	}
}
