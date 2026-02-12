package main

import (
	"log/slog"
	"os"

	"github.com/alecthomas/kong"

	"github.com/xymaxim/ypb/internal/commands"
	"github.com/xymaxim/ypb/internal/commands/capture"
)

type CLI struct {
	Verbose int `help:"Show verbose output." short:"v" type:"counter"`

	Capture  CaptureCommands   `cmd:"" help:"Capture single frame or time-lapse sequence"`
	Download commands.Download `cmd:"" help:"Download stream excerpts"`
	Serve    commands.Serve    `cmd:"" help:"Start playback server"`
	Version  commands.Version  `cmd:"" help:"Show version info and exit"`
}

type CaptureCommands struct {
	Frame capture.Frame `cmd:"" help:"Capture a single frame"`
}

type VersionFlag string

func main() {
	var cli CLI

	kongCtx := kong.Parse(&cli,
		kong.Name("ypb"),
		kong.Description("A playback for YouTube live streams"),
		kong.UsageOnError(),
	)

	setupLogging(cli.Verbose)

	err := kongCtx.Run()
	kongCtx.FatalIfErrorf(err)
}

func setupLogging(verbose int) {
	var level slog.Level

	switch verbose {
	case 0:
		level = slog.LevelWarn
	case 1:
		level = slog.LevelInfo
	case 2:
		level = slog.LevelDebug
	default:
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: level,
		},
	)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
