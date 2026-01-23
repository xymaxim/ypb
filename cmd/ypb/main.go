package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"

	apppkg "github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/commands"
)

func main() {
	app := apppkg.NewApp()

	cliApp := &cli.Command{
		Name:  "ypb",
		Usage: "A playback for YouTube live streams",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "port",
				Value: 8080,
				Usage: "port to start playback on",
			},
		},
		Commands: []*cli.Command{
			commands.NewDownloadCommand(app),
			commands.NewServeCommand(app),
		},
	}

	if err := cliApp.Run(context.Background(), os.Args); err != nil {
		log.Fatalf("error: %v", err)
	}
}
