// mockplay runs a playback server backed by fixture data instead of a live
// YouTube stream, for local development and testing of the dash.js web
// player without hitting YouTube.

package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/alecthomas/kong"

	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/mockserver"
	"github.com/xymaxim/ypb/internal/urlutil"
)

// ListenAddr is a host:port pair parsed from a command-line flag.
type ListenAddr struct {
	Host string
	Port int
}

func (a *ListenAddr) UnmarshalText(text []byte) error {
	host, portStr, err := net.SplitHostPort(string(text))
	if err != nil {
		return fmt.Errorf("expected host:port: %w", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port %q: must be a number", portStr)
	}
	if port < 0 || port > 65535 {
		return fmt.Errorf("invalid port %d: must be between 0 and 65535", port)
	}
	a.Host, a.Port = host, port
	return nil
}

var cli struct {
	FixtureDir string        `help:"Path to fixture data" default:"playback/testdata/fixture"`
	Addr       ListenAddr    `help:"Address to listen on"               default:":9000"`
	StreamAge  time.Duration `help:"How long the stream has been live"  default:"240h"`
}

func main() {
	kctx := kong.Parse(&cli,
		kong.Name("mockplay"),
		kong.Description("Run a mock playback server"),
	)

	actualStartTime := time.Now().Add(-cli.StreamAge)

	ln, err := apppkg.Listen(cli.Addr.Host, cli.Addr.Port)
	kctx.FatalIfErrorf(err)
	addr := ln.Addr().String()

	handler, err := mockserver.New(cli.FixtureDir, actualStartTime, addr, true)
	if err != nil {
		log.Fatalf("setting up mock server: %v", err)
	}

	server := &http.Server{
		Addr:    addr,
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

	fmt.Printf(
		"(<<) Mock playback started and listening on %s\n",
		urlutil.FormatServerAddress(addr),
	)

	slog.Info("serving fixture", slog.String("dir", cli.FixtureDir))
	slog.Info(
		"stream age",
		slog.Duration("age", cli.StreamAge),
		slog.String("started", actualStartTime.Format(time.RFC1123)),
	)

	if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
