package exec

import (
	"bufio"
	"fmt"
	"io"
	execpkg "os/exec"
	"path/filepath"
	"strings"
)

// Runner defines the interface for executing commands.
type Runner interface {
	Run(args ...string) error
	RunQuiet(args ...string) (stdout, stderr string, err error)
	RunWithCallbacks(onStdout, onStderr func(string), args ...string) error
}

// CommandRunner executes actual commands.
type CommandRunner struct {
	path string
	name string
}

// NewCommandRunner creates a new CommandRunner with binary path.
func NewCommandRunner(path string) *CommandRunner {
	return &CommandRunner{path: path, name: filepath.Base(path)}
}

// Run executes the command with the given arguments and prints output to stdout/stderr.
func (r *CommandRunner) Run(args ...string) error {
	return r.RunWithCallbacks(r.PrintCallback, r.PrintCallback, args...)
}

// RunQuiet executes the command silently and returns captured stdout/stderr.
func (r *CommandRunner) RunQuiet(args ...string) (string, string, error) {
	var outBuf, errBuf strings.Builder
	err := r.RunWithCallbacks(
		func(line string) { outBuf.WriteString(line + "\n") },
		func(line string) { errBuf.WriteString(line + "\n") },
		args...,
	)
	return outBuf.String(), errBuf.String(), err
}

// RunWithCallbacks executes the command with custom handlers for stdout and
// stderr. Pass nil for either callback to discard that output stream.
func (r *CommandRunner) RunWithCallbacks(onStdout, onStderr func(string), args ...string) error {
	cmd := execpkg.Command(r.path, args...) // #nosec: G204

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting command: %w", err)
	}

	handle := func(p io.ReadCloser, h func(string)) {
		if h == nil {
			go func() { io.Copy(io.Discard, p) }()
		} else {
			go func() { streamPipe(p, h) }()
		}
	}
	handle(stdout, onStdout)
	handle(stderr, onStderr)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("running command: %w", err)
	}

	return nil
}

// PrintCallback is a default callback for stdout and stderr.
func (r *CommandRunner) PrintCallback(line string) {
	fmt.Printf("%s: %s\n", r.name, line)
}

// streamPipe handles streams with simple line-by-line reading.
func streamPipe(pipe io.ReadCloser, handler func(string)) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		handler(scanner.Text())
	}
}
