package ypb

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/xymaxim/ypb/fetchers"
	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/playback"
)

// Streamer controls the playback server lifecycle.
type Streamer interface {
	Start() error
	Stop()
	Server() *http.Server
}

// Stream is the playback server implementation.
type Stream struct {
	server *http.Server
	host   string
	port   int
	cancel context.CancelFunc
	done   chan struct{}
}

// StreamConfig holds options for creating a new stream.
type StreamConfig struct {
	Fetcher fetchers.Fetcher
	OnPrint func([]byte)

	// AllowedOrigins lists the browser origins allowed to read responses,
        // e.g. "http://localhost:9000". Matching is exact, so "localhost" and
	// "127.0.0.1" are different origins. Empty disables CORS.
	AllowedOrigins []string
}

// NewStream creates a new playback server backed by a real YouTube stream. It
// listens on host (empty for all interfaces, "127.0.0.1" for this computer
// only) and port.
func NewStream(
	ctx context.Context,
	videoID, host string,
	port int,
	cfg *StreamConfig,
) (*Stream, error) {
	ytdlpRunner := exec.NewCommandRunner(apppkg.YtdlpBinaryPath)
	ffprobeRunner := exec.NewCommandRunner(apppkg.FFprobeBinaryPath)

	fetcher := cfg.Fetcher
	if fetcher == nil {
		fetcher = &fetchers.YtdlpFetcher{
			VideoID: videoID,
			Runner:  ytdlpRunner,
			OnPrint: cfg.OnPrint,
		}
	}

	pb, err := playback.NewPlayback(ctx, videoID, fetcher, nil)
	if err != nil {
		return nil, fmt.Errorf("creating playback: %w", err)
	}

	return newStream(ctx, pb, host, port, ffprobeRunner, cfg.AllowedOrigins)
}

// NewMockStream creates a playback server backed by fixture data instead of a
// live YouTube stream. See containers/ypb-mock on how to generate fixture data.
func NewMockStream(
	ctx context.Context,
	fixtureDir, host string,
	port int,
	actualStartTime time.Time,
) (*Stream, error) {
	pb, err := playback.NewMockPlayback(fixtureDir, actualStartTime)
	if err != nil {
		return nil, fmt.Errorf("creating mock playback: %w", err)
	}

	ffprobeRunner := exec.NewCommandRunner(apppkg.FFprobeBinaryPath)

	return newStream(ctx, pb, host, port, ffprobeRunner, nil)
}

// newStream wires a Playbacker into a playback server.
func newStream(
	ctx context.Context,
	pb playback.Playbacker,
	host string,
	port int,
	ffprobeRunner exec.Runner,
	allowedOrigins []string,
) (*Stream, error) {
	ctx, cancel := context.WithCancel(ctx)

	server := &http.Server{
		Addr: apppkg.ServerAddr(host, port),
	}

	app := &apppkg.App{
		Playback:      pb,
		FFprobeRunner: ffprobeRunner,
		Server:        server,
	}

	mux := http.NewServeMux()
	apppkg.RegisterInfoRoute(mux, app)
	apppkg.RegisterMPDRoute(mux, app)
	apppkg.RegisterSegmentRoute(mux, app)

	allowed := make(map[string]bool, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if origin != "" {
			allowed[origin] = true
		}
	}
	server.Handler = apppkg.WithCORS(allowed, mux)

	stream := &Stream{
		server: server,
		host:   host,
		port:   port,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	go func() {
		<-ctx.Done()
		if err := stream.server.Close(); err != nil {
			log.Println("failed to close stream server")
		}
		close(stream.done)
	}()

	return stream, nil
}

// Server returns the underlying HTTP server.
func (s *Stream) Server() *http.Server {
	return s.server
}

// Start begins serving requests. Blocks until the server is closed. If the
// configured port is in use, the next free port is used and the server address
// is updated accordingly.
func (s *Stream) Start() error {
	ln, err := apppkg.Listen(s.host, s.port)
	if err != nil {
		return fmt.Errorf("starting stream server: %w", err)
	}
	s.server.Addr = ln.Addr().String()

	if err := s.server.Serve(ln); err != http.ErrServerClosed {
		return fmt.Errorf("starting stream server: %w", err)
	}
	return nil
}

// Stop gracefully shuts down the server.
func (s *Stream) Stop() {
	s.cancel()
	<-s.done
}
