package ypb

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
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
	cancel context.CancelFunc
	done   chan struct{}
}

// StreamConfig holds options for creating a new stream.
type StreamConfig struct {
	Fetcher fetchers.Fetcher
	OnPrint func([]byte)
}

// NewStream creates a new playback server backed by a real YouTube stream.
func NewStream(ctx context.Context, videoID string, port int, cfg *StreamConfig) (*Stream, error) {
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

	return newStream(ctx, pb, port, ffprobeRunner)
}

// NewMockStream creates a playback server backed by fixture data instead of a
// live YouTube stream. See containers/ypb-mock on how to generate fixture data.
func NewMockStream(
	ctx context.Context,
	fixtureDir string,
	port int,
	actualStartTime time.Time,
) (*Stream, error) {
	pb, err := playback.NewMockPlayback(fixtureDir, actualStartTime)
	if err != nil {
		return nil, fmt.Errorf("creating mock playback: %w", err)
	}

	ffprobeRunner := exec.NewCommandRunner(apppkg.FFprobeBinaryPath)

	return newStream(ctx, pb, port, ffprobeRunner)
}

// newStream wires a Playbacker into a playback server.
func newStream(
	ctx context.Context,
	pb playback.Playbacker,
	port int,
	ffprobeRunner exec.Runner,
) (*Stream, error) {
	ctx, cancel := context.WithCancel(ctx)

	server := &http.Server{
		Addr: ":" + strconv.Itoa(port),
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
	server.Handler = apppkg.WithCORS(mux)

	stream := &Stream{
		server: server,
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

// Start begins serving requests. Blocks until the server is closed.
func (s *Stream) Start() error {
	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("starting stream server: %w", err)
	}
	return nil
}

// Stop gracefully shuts down the server.
func (s *Stream) Stop() {
	s.cancel()
	<-s.done
}
