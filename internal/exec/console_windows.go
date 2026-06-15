//go:build windows

package exec

import (
	"os/exec"
	"syscall"
)

// SuppressConsole hides the window for any command on Windows.
func SuppressConsole(cmd *exec.Cmd) *exec.Cmd {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	return cmd
}
