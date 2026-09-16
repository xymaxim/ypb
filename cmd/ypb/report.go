package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kong"
)

const (
	reportTimeFormat = "20060102-150405"
	redactedIP       = "0.0.0.0"
)

// redactor replaces IP addresses with a safe placeholder.
type redactor struct {
	token *regexp.Regexp
}

func newRedactor() *redactor {
	return &redactor{
		token: regexp.MustCompile(`\[[0-9a-fA-F:.]+\]|[0-9a-fA-F:.]+`),
	}
}

func (r *redactor) redact(b []byte) []byte {
	matches := r.token.FindAllIndex(b, -1)
	if len(matches) == 0 {
		return b
	}

	out := make([]byte, 0, len(b))
	last := 0
	for _, m := range matches {
		start, end := m[0], m[1]
		out = append(out, b[last:start]...)
		if bounded(b, start, end) {
			if replaced, ok := redactIP(b[start:end]); ok {
				out = append(out, replaced...)
				last = end
				continue
			}
		}
		out = append(out, b[start:end]...)
		last = end
	}
	return append(out, b[last:]...)
}

func bounded(buf []byte, start, end int) bool {
	if start > 0 && isWord(buf[start-1]) {
		return false
	}
	if end < len(buf) && isWord(buf[end]) {
		return false
	}
	return true
}

func isWord(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

func redactIP(token []byte) ([]byte, bool) {
	cand := string(token)
	if len(cand) >= 2 && cand[0] == '[' && cand[len(cand)-1] == ']' {
		cand = cand[1 : len(cand)-1]
	}
	if net.ParseIP(cand) != nil {
		return []byte(redactedIP), true
	}
	// IPv4 address with a port suffix, e.g. "192.168.0.1:9000".
	if i := strings.LastIndexByte(cand, ':'); i > 0 {
		if net.ParseIP(cand[:i]) != nil {
			return append([]byte(redactedIP), cand[i:]...), true
		}
	}
	return token, false
}

// reportSink writes a redacted copy of the output to the report file. It
// buffers output until a newline so that IP addresses to be redacted split
// across write chunks are still detected, and holds any trailing partial line
// until flush.
type reportSink struct {
	file     io.Writer
	redactor *redactor
	buf      []byte
}

func (s *reportSink) Write(p []byte) (int, error) {
	s.buf = append(s.buf, p...)
	for {
		i := bytes.IndexByte(s.buf, '\n')
		if i < 0 {
			return len(p), nil
		}
		// A write error here must never propagate: returning an error
		// would stop the whole tee and eventually block the app's real
		// output. We simply drop the failed report line.
		_, _ = s.file.Write(s.redactor.redact(s.buf[:i+1]))
		s.buf = s.buf[i+1:]
	}
}

// flush writes any buffered partial line to the report file.
func (s *reportSink) flush() {
	if len(s.buf) == 0 {
		return
	}
	_, _ = s.file.Write(s.redactor.redact(s.buf))
	s.buf = s.buf[:0]
}

// forward tees the pipe to the screen and to the report sink (redacted).
func forward(pipe io.Reader, screen, sink io.Writer, wg *sync.WaitGroup) {
	defer wg.Done()
	_, _ = io.Copy(io.MultiWriter(screen, sink), pipe)
}

func enableReport(ctx *kong.Context) (func(), error) {
	name := "ypb-" + time.Now().Format(reportTimeFormat) + ".log"
	file, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("creating report file %s: %w", name, err)
	}

	sink := &reportSink{file: file, redactor: newRedactor()}

	realStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("creating output pipe: %w", err)
	}

	os.Stdout = writer
	os.Stderr = writer
	ctx.Stdout = writer
	ctx.Stderr = writer

	var wg sync.WaitGroup
	wg.Add(1)
	go forward(reader, realStdout, sink, &wg)

	var once sync.Once
	closeReport := func() {
		once.Do(func() {
			_ = writer.Close()
			wg.Wait()
			sink.flush()
			_ = file.Close()
		})
	}
	ctx.Exit = func(code int) {
		closeReport()
		os.Exit(code)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Report will be saved to %s\n", name)
	return closeReport, nil
}
