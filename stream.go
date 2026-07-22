package ypb

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/exec"
	"github.com/xymaxim/ypb/playback"
	"github.com/xymaxim/ypb/playback/fetchers"
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

// NewStream creates a new playback server.
func NewStream(ctx context.Context, videoID string, port int, cfg *StreamConfig) (*Stream, error) {
	ctx, cancel := context.WithCancel(ctx)

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

	server := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		ReadHeaderTimeout: 20 * time.Second,
	}

	mux := http.NewServeMux()
	mux.HandleFunc(apppkg.InfoPath, apppkg.WithError(
		(&apppkg.InfoHandler{Info: pb.Info()}).ServeHTTP),
	)
	mux.HandleFunc(apppkg.MPDPath, apppkg.WithError(
		(&apppkg.MPDHandler{
			Playback:      pb,
			FFprobeRunner: ffprobeRunner,
			ServerAddr:    server.Addr,
		}).ServeHTTP),
	)
	mux.HandleFunc(apppkg.SegmentPath, apppkg.WithError(
		(&apppkg.SegmentHandler{Playback: pb}).ServeHTTP),
	)
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
