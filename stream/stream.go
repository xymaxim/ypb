package stream

import (
	"context"
	"fmt"
	"log"
	"net/http"

	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/playback"
)

type Streamer interface {
	Start() error
	Stop()
	Server() *http.Server
	Playback() playback.Playbacker
}

type Stream struct {
	app    *apppkg.App
	server *http.Server
	cancel context.CancelFunc
	done   chan struct{}
}

type StreamConfig struct {
	OnPrint func([]byte)
}

func NewStream(ctx context.Context, videoID string, port int, cfg *StreamConfig) (*Stream, error) {
	ctx, cancel := context.WithCancel(ctx)

	app := apppkg.NewApp()

	if err := app.Initialize(ctx, videoID, &apppkg.Config{
		Port:    port,
		OnPrint: cfg.OnPrint,
	}); err != nil {
		return nil, fmt.Errorf("initializing app: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(apppkg.InfoPath, apppkg.WithError(
		(&apppkg.InfoHandler{Info: app.Playback.Info()}).ServeHTTP),
	)
	mux.HandleFunc(apppkg.MPDPath, apppkg.WithError(
		(&apppkg.MPDHandler{
			Playback:      app.Playback,
			FFprobeRunner: app.FFprobeRunner,
			ServerAddr:    app.Server.Addr,
		}).ServeHTTP),
	)
	mux.HandleFunc(apppkg.SegmentPath, apppkg.WithError(
		(&apppkg.SegmentHandler{Playback: app.Playback}).ServeHTTP),
	)
	app.Server.Handler = mux

	stream := &Stream{
		app:    app,
		server: app.Server,
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

func (s *Stream) Server() *http.Server {
	return s.server
}

func (s *Stream) Playback() playback.Playbacker {
	return s.app.Playback
}

func (s *Stream) Start() error {
	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("starting stream server: %w", err)
	}
	return nil
}

func (s *Stream) Stop() {
	s.cancel()
	<-s.done
}
