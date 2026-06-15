//go:build unix

package exec

import "os/exec"

// SuppressConsole is a no-op on Unix-like systems (Linux, macOS, etc.).
// Console suppression is only needed on Windows.
func SuppressConsole(cmd *exec.Cmd) *exec.Cmd {
	return cmd
}
