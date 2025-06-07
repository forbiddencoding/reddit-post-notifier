package redditpostnotifier

import (
	"bytes"
	"context"
	"fmt"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence"
	"github.com/forbiddencoding/reddit-post-notifier/services/reddit"
	"github.com/go-playground/validator/v10"
	"github.com/urfave/cli/v3"
	"go.temporal.io/sdk/client"
	"go.uber.org/automaxprocs/maxprocs"
	"golang.org/x/sync/errgroup"
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
				Value: "./config/config.yml",
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

	validate := validator.New(validator.WithRequiredStructEnabled())

	conf, err := config.LoadConfig(ctx, cmd.String("config"), validate)
	if err != nil {
		slog.Error("failed to load config", slog.Any("error", err))
		return err
	}

	temporalClient, err := client.DialContext(ctx, client.Options{
		HostPort:  conf.Temporal.HostPort,
		Namespace: conf.Temporal.Namespace,
	})
	if err != nil {
		slog.Error("failed to connect to temporal", slog.Any("error", err))
		return fmt.Errorf("failed to connect to temporal: %w", err)
	}
	defer temporalClient.Close()

	db, err := persistence.New(ctx, &conf.Persistence)
	if err != nil {
		slog.Error("failed to create persistence handle", slog.Any("error", err))
		return fmt.Errorf("failed to create persistence handle: %w", err)
	}
	defer func() {
		_ = db.Close(ctx)
	}()

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return reddit.RunWorker(ctx, temporalClient, db, conf)
	})

	return g.Wait()
}
