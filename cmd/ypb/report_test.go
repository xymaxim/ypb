package main

import (
	"bytes"
	"errors"
	"testing"
)

func TestRedactor_Redact(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"ipv4 plain", "connect to 192.168.1.10 now", "connect to 0.0.0.0 now"},
		{"ipv4 with port", "listening on 127.0.0.1:8080", "listening on 0.0.0.0:8080"},
		{"ipv6 plain", "addr fe80::1 seen", "addr 0.0.0.0 seen"},
		{"ipv6 bracketed with port", "dial [::1]:9000 failed", "dial 0.0.0.0:9000 failed"},
		{"ipv4-mapped ipv6", "peer ::ffff:192.0.2.1 connected", "peer 0.0.0.0 connected"},
		{"multiple ips one line", "from 10.0.0.1 to 10.0.0.2", "from 0.0.0.0 to 0.0.0.0"},
		{"version string untouched", "running v1.2.3.4 build", "running v1.2.3.4 build"},
		{"non-ip numeric token untouched", "date 20060102 ok", "date 20060102 ok"},
		{"no ip present", "hello world", "hello world"},
	}

	r := newRedactor()
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := string(r.redact([]byte(tt.in)))
			if got != tt.want {
				t.Errorf("redact(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestReportSink_BuffersUntilNewline(t *testing.T) {
	var out bytes.Buffer
	sink := &reportSink{file: &out, redactor: newRedactor()}

	partial := "partial line no newline"
	n, err := sink.Write([]byte(partial))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(partial) {
		t.Fatalf("n = %d, want %d", n, len(partial))
	}
	if out.Len() != 0 {
		t.Fatalf("expected nothing written yet, got %q", out.String())
	}

	if _, err := sink.Write([]byte(" continues\n")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "partial line no newline continues\n"
	if out.String() != want {
		t.Fatalf("out = %q, want %q", out.String(), want)
	}
}

func TestReportSink_RedactsIPSplitAcrossWrites(t *testing.T) {
	var out bytes.Buffer
	sink := &reportSink{file: &out, redactor: newRedactor()}

	if _, err := sink.Write([]byte("client 192.168.")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := sink.Write([]byte("1.10 connected\n")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "client 0.0.0.0 connected\n"
	if out.String() != want {
		t.Fatalf("out = %q, want %q", out.String(), want)
	}
}

func TestReportSink_MultipleLinesInOneWrite(t *testing.T) {
	var out bytes.Buffer
	sink := &reportSink{file: &out, redactor: newRedactor()}

	if _, err := sink.Write(
		[]byte("first 10.0.0.1\nsecond 10.0.0.2\nthird partial"),
	); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "first 0.0.0.0\nsecond 0.0.0.0\n"
	if out.String() != want {
		t.Fatalf("out = %q, want %q", out.String(), want)
	}

	sink.flush()
	want += "third partial"
	if out.String() != want {
		t.Fatalf("after flush, out = %q, want %q", out.String(), want)
	}
}

func TestReportSink_FlushNoOpWhenEmpty(t *testing.T) {
	var out bytes.Buffer
	sink := &reportSink{file: &out, redactor: newRedactor()}

	sink.flush()
	if out.Len() != 0 {
		t.Fatalf("expected no writes, got %q", out.String())
	}
}

type failingWriter struct{}

func (failingWriter) Write(p []byte) (int, error) {
	return 0, errors.New("I/O error")
}

func TestReportSink_WriteErrorsDoNotPropagate(t *testing.T) {
	sink := &reportSink{file: failingWriter{}, redactor: newRedactor()}

	line := "192.168.1.1 boom\n"
	n, err := sink.Write([]byte(line))
	if err != nil {
		t.Fatalf("expected no error from sink.Write despite writing error, got %v", err)
	}
	if n != len(line) {
		t.Fatalf("n = %d, want %d", n, len(line))
	}
}
