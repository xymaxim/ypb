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
	RunWith(options []Option, args ...string) (*RunResult, error)
}

// RunResult contains the captured output from a command.
type RunResult struct {
	Stdout string
	Stderr string
}

// RunConfig configures command execution.
type RunConfig struct {
	Stdin         io.Reader
	OnStdout      func(string)
	OnStderr      func(string)
	CaptureOutput bool
	stdout        *strings.Builder
	stderr        *strings.Builder
}

// Option is a functional option for configuring RunConfig.
type Option func(*RunConfig)

// WithStdin sets an io.Reader as stdin for the command.
func WithStdin(r io.Reader) Option {
	return func(o *RunConfig) {
		o.Stdin = r
	}
}

// WithQuiet captures stdout and stderr instead of printing them.
func WithQuiet() Option {
	return func(o *RunConfig) {
		o.CaptureOutput = true
		o.stdout = &strings.Builder{}
		o.stderr = &strings.Builder{}
		o.OnStdout = func(line string) { o.stdout.WriteString(line + "\n") }
		o.OnStderr = func(line string) { o.stderr.WriteString(line + "\n") }
	}
}

// WithCallbacks sets custom handlers for both stdout and stderr lines.
func WithCallbacks(onStdout, onStderr func(string)) Option {
	return func(o *RunConfig) {
		o.OnStdout = onStdout
		o.OnStderr = onStderr
	}
}

// CommandRunner executes actual commands.
type CommandRunner struct {
	Path string
	Name string
}

// NewCommandRunner creates a new CommandRunner with binary path.
func NewCommandRunner(path string) *CommandRunner {
	return &CommandRunner{Path: path, Name: filepath.Base(path)}
}

// Run executes the command with the given arguments and prints output to stdout/stderr.
func (r *CommandRunner) Run(args ...string) error {
	_, err := r.RunWith(nil, args...)
	return err
}

// RunWith executes the command with functional options and returns captured output if requested.
func (r *CommandRunner) RunWith(options []Option, args ...string) (*RunResult, error) {
	config := RunConfig{
		OnStdout: r.PrintCallback,
		OnStderr: r.PrintCallback,
	}

	for _, o := range options {
		o(&config)
	}

	err := r.runWithConfig(config, args...)

	var result *RunResult
	if config.CaptureOutput {
		result = &RunResult{
			Stdout: config.stdout.String(),
			Stderr: config.stderr.String(),
		}
	}

	return result, err
}

// PrintCallback is a default callback for stdout and stderr.
func (r *CommandRunner) PrintCallback(line string) {
	fmt.Printf("%s: %s\n", r.Name, line)
}

// runWithOptions executes the command with the given config and arguments.
func (r *CommandRunner) runWithConfig(config RunConfig, args ...string) error {
	cmd := execpkg.Command(r.Path, args...) // #nosec: G204

	// Setup stdin if provided
	if config.Stdin != nil {
		cmd.Stdin = config.Stdin
	}

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
	handle(stdout, config.OnStdout)
	handle(stderr, config.OnStderr)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("running command: %w", err)
	}
	return nil
}

// streamPipe handles streams with simple line-by-line reading.
func streamPipe(pipe io.ReadCloser, handler func(string)) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		handler(scanner.Text())
	}
}
