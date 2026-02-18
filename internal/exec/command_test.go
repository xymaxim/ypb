package exec_test

import (
	"bytes"
	"io"
	"os"
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/internal/testutil"
)

func getShellCommand(script string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd.exe", []string{"/c", script}
	}
	return "sh", []string{"-c", script}
}

// captureConsoleOutput captures stdout/stderr during test.
func captureConsoleOutput(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	t.Cleanup(func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	})

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	os.Stdout = wOut
	os.Stderr = wErr

	fn()

	wOut.Close()
	wErr.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	io.Copy(&stdoutBuf, rOut)
	io.Copy(&stderrBuf, rErr)

	return stdoutBuf.String(), stderrBuf.String()
}

func TestCommandRunner_Run(t *testing.T) {
	shell, args := getShellCommand(`printf "test"`)
	runner := exec.NewCommandRunner(shell)

	gotConsoleStdout, _ := captureConsoleOutput(t, func() {
		if err := runner.Run(args...); err != nil {
			t.Errorf("Run() error = %v, want nil", err)
		}
	})

	wantConsoleStdout := "test"
	if diff := cmp.Diff(wantConsoleStdout, gotConsoleStdout); diff != "" {
		t.Errorf("console stdout mismatch %s", testutil.PrintWantGot(diff))
	}
}

// func TestCommandRunner_Run(t *testing.T) {
// 	shell, args := getShellCommand("echo test")
// 	runner := exec.NewCommandRunner(shell)

// 	gotConsoleStdout, _ := captureConsoleOutput(t, func() {
// 		if err := runner.Run(args...); err != nil {
// 			t.Errorf("Run() error = %v, want nil", err)
// 		}
// 	})
// 	wantConsoleStdout := shell + ": test\n"
// 	if diff := cmp.Diff(wantConsoleStdout, gotConsoleStdout); diff != "" {
// 		t.Errorf("console stdout mismatch %s", testutil.PrintWantGot(diff))
// 	}
// }

func TestCommandRunner_Name(t *testing.T) {
	runner := exec.NewCommandRunner("/usr/bin/test-binary")

	if runner.Name != "test-binary" {
		t.Errorf("expected name 'test-binary', got: %q", runner.Name)
	}

	if runtime.GOOS == "windows" {
		runner = exec.NewCommandRunner(`C:\Program Files\test-binary.exe`)
		if runner.Name != "test-binary.exe" {
			t.Errorf("expected name 'test-binary.exe', got: %q", runner.Name)
		}
	}
}

func TestCommandRunner_RunWith_Quiet(t *testing.T) {
	shell, args := getShellCommand(
		`printf "stdout line 1\nstdout line 2\n" && printf "stderr line 1\nstderr line 2\n" 1>&2`,
	)
	runner := exec.NewCommandRunner(shell)

	gotConsoleStdout, gotConsoleStderr := captureConsoleOutput(t, func() {
		got, err := runner.RunWith([]exec.Option{exec.WithQuiet()}, args...)
		if err != nil {
			t.Fatalf("RunWith() error = %v, want nil", err)
		}
		if diff := cmp.Diff(
			[]byte("stdout line 1\nstdout line 2\n"),
			got.Stdout,
		); diff != "" {
			t.Errorf("captured stdout mismatch %s", testutil.PrintWantGot(diff))
		}
		if diff := cmp.Diff(
			[]byte("stderr line 1\nstderr line 2\n"),
			got.Stderr,
		); diff != "" {
			t.Errorf("captured stderr mismatch %s", testutil.PrintWantGot(diff))
		}
	})

	if gotConsoleStdout != "" {
		t.Errorf("expected no console stdout, got: %q", gotConsoleStdout)
	}
	if gotConsoleStderr != "" {
		t.Errorf("expected no console stderr, got: %q", gotConsoleStderr)
	}
}

func TestCommandRunner_RunWith_Callbacks(t *testing.T) {
	shell, args := getShellCommand(`printf "line 1\nline 2\n"`)
	runner := exec.NewCommandRunner(shell)

	var buf bytes.Buffer
	gotConsoleStdout, _ := captureConsoleOutput(t, func() {
		_, err := runner.RunWith(
			[]exec.Option{
				exec.WithCallbacks(
					func(b []byte) { buf.Write(b) },
					runner.PrintCallback,
				),
			},
			args...,
		)
		if err != nil {
			t.Errorf("RunWith() error = %v, want nil", err)
		}
	})

	if diff := cmp.Diff([]byte("line 1\nline 2\n"), buf.Bytes()); diff != "" {
		t.Errorf("captured stdout mismatch %s", testutil.PrintWantGot(diff))
	}
	if gotConsoleStdout != "" {
		t.Errorf("expected no console stdout, got: %q", gotConsoleStdout)
	}
}
