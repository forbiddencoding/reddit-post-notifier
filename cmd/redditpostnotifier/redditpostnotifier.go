package redditpostnotifier

import (
	"bytes"
	"context"
	"github.com/urfave/cli/v3"
	"go.uber.org/automaxprocs/maxprocs"
	"log/slog"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
)

func BuildCLI() *cli.Command {
	return &cli.Command{
		Name:  "reddit post notifier",
		Usage: "rpn",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Value: "./config/config.yaml",
				Usage: "config path is a path relative to root, or an absolute path",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "start reddit post notifier services",
				Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
					if _, err := maxprocs.Set(); err != nil {
						slog.Warn("could not set GOMAXPROCS", slog.Any("error", err))
					}

					return ctx, nil
				},
				Action: start,
			},
		},
	}
}

func start(ctx context.Context, cmd *cli.Command) error {
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()

		if r := recover(); r != nil {
			var buf bytes.Buffer
			if err := pprof.Lookup("goroutine").WriteTo(&buf, 2); err != nil {
				slog.Debug("failed to write goroutine stack trace", slog.Any("error", err))
			}
			slog.Debug("application panic", slog.Any("panic", r), slog.String("goroutines", buf.String()))
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("received shutdown signal, shutting down...")
		cancel()
	}()

	return nil
}
