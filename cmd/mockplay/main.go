// mockplay runs a playback server backed by fixture data instead of a live
// YouTube stream, for local development and testing of the dash.js web
// player without hitting YouTube.

package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/alecthomas/kong"

	"github.com/xymaxim/ypb/internal/mockserver"
)

var cli struct {
	FixtureDir string        `help:"Path to fixture data" default:"playback/testdata/fixture"`
	Addr       string        `help:"Address to listen on"               default:":9000"`
	StreamAge  time.Duration `help:"How long the stream has been live"  default:"240h"`
}

func main() {
	kong.Parse(&cli,
		kong.Name("mockplay"),
		kong.Description("Run a mock playback server"),
	)

	actualStartTime := time.Now().Add(-cli.StreamAge)

	handler, err := mockserver.New(cli.FixtureDir, actualStartTime, cli.Addr, true)
	if err != nil {
		log.Fatalf("setting up mock server: %v", err)
	}

	server := &http.Server{
		Addr:    cli.Addr,
		Handler: handler,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Println("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	fmt.Printf("(<<) Mock playback started and listening on http://localhost%s\n", cli.Addr)

	slog.Info("serving fixture", slog.String("dir", cli.FixtureDir))
	slog.Info(
		"stream age",
		slog.Duration("age", cli.StreamAge),
		slog.String("started", actualStartTime.Format(time.RFC1123)),
	)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
