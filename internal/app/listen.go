package app

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"runtime"
	"syscall"
)

// maxPortAttempts is how many consecutive ports Listen tries.
const maxPortAttempts = 10

// Winsock codes for a busy or blocked port. They don't match syscall.EADDRINUSE.
const (
	errWSAEACCES     = syscall.Errno(10013)
	errWSAEADDRINUSE = syscall.Errno(10048)
)

// isPortUnavailable reports whether err means the port can't be bound.
func isPortUnavailable(err error) bool {
	if errors.Is(err, syscall.EADDRINUSE) {
		return true
	}
	return runtime.GOOS == "windows" &&
		(errors.Is(err, errWSAEADDRINUSE) || errors.Is(err, errWSAEACCES))
}

// Listen binds a TCP listener on host:port. If the port is taken, it tries
// the next ports in sequence, up to maxPortAttempts. Use ln.Addr() to get
// the address actually bound.
func Listen(host string, port int) (net.Listener, error) {
	if port < 0 || port > 65535 {
		return nil, fmt.Errorf("invalid port %d: must be between 0 and 65535", port)
	}

	var lastErr error
	last := min(port+maxPortAttempts-1, 65535)
	for p := port; p <= last; p++ {
		addr := ServerAddr(host, p)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			return ln, nil
		}
		if !isPortUnavailable(err) {
			return nil, fmt.Errorf("listening on %s: %w", addr, err)
		}
		lastErr = err
		slog.Warn("port unavailable, trying next", "port", p, "error", err)
	}

	return nil, fmt.Errorf(
		"no free port found in range %d-%d: %w",
		port,
		last,
		lastErr,
	)
}
