package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence"
	"github.com/forbiddencoding/reddit-post-notifier/common/reddit"
	"github.com/forbiddencoding/reddit-post-notifier/common/server"
	"github.com/forbiddencoding/reddit-post-notifier/services/app"
	"github.com/forbiddencoding/reddit-post-notifier/services/app/api"
	"github.com/forbiddencoding/reddit-post-notifier/services/digester"
	"github.com/forbiddencoding/reddit-post-notifier/services/redditor"
	"github.com/go-playground/validator/v10"
	"github.com/urfave/cli/v3"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
	"golang.org/x/sync/errgroup"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"
)

const (
	ServiceApp    = "app"
	ServiceDigest = "digest"
	ServiceReddit = "reddit"
)

type ServiceFactory func(ctx context.Context, infra *infrastructure, conf *config.Config, v *validator.Validate) (Service, error)

type Service interface {
	io.Closer
	Start() error
}

func buildCLI() *cli.Command {
	return &cli.Command{
		Name:  "reddit-post-notifier",
		Usage: "rpn",
		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "start reddit post notifier services",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "services",
						Aliases: []string{"s"},
						Usage:   "comma-separated list of services (app, digest, reddit)",
						Value:   strings.Join([]string{ServiceApp, ServiceDigest, ServiceReddit}, ","),
					},
				},
				Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
					if cmd.String("services") == "" {
						return ctx, fmt.Errorf("no services provided")
					}
					return ctx, nil
				},
				Action: start,
			},
		},
	}
}

func start(ctx context.Context, cmd *cli.Command) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	defer func() {
		if r := recover(); r != nil {
			var buf bytes.Buffer
			if err := pprof.Lookup("goroutine").WriteTo(&buf, 2); err != nil {
				slog.Error("failed to write goroutine stack trace", slog.Any("error", err))
			}
			slog.Error("application panic", slog.Any("panic", r), slog.String("goroutines", buf.String()))
			os.Exit(1)
		}
	}()

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	validate := validator.New(validator.WithRequiredStructEnabled())
	conf, err := config.LoadConfig(ctx, validate)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	infra, err := bootstrapInfrastructure(ctx, conf)
	if err != nil {
		return err
	}
	defer infra.Close()

	registry := map[string]ServiceFactory{
		ServiceApp:    startAppService,
		ServiceDigest: startDigestService,
		ServiceReddit: startRedditService,
	}

	g, ctx := errgroup.WithContext(ctx)

	for serviceName := range strings.SplitSeq(cmd.String("services"), ",") {
		name := strings.TrimSpace(serviceName)
		if name == "" {
			continue
		}

		factory, ok := registry[name]
		if !ok {
			return fmt.Errorf("unknown service: %s", name)
		}

		shutdownComplete := make(chan struct{})

		g.Go(func() error {
			svc, err := factory(ctx, infra, conf, validate)
			if err != nil {
				return fmt.Errorf("failed to init %s: %w", name, err)
			}

			slog.Info("starting service", slog.String("service", name))

			stop := context.AfterFunc(ctx, func() {
				slog.Info("closing service", slog.String("service", name))
				if err = svc.Close(); err != nil {
					slog.Error("closing server", slog.String("service", name), slog.Any("error", err))
				}
				close(shutdownComplete)
			})
			defer stop()

			err = svc.Start()
			if ctx.Err() != nil {
				<-shutdownComplete
			}

			return err
		})
	}

	if err = g.Wait(); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}

	return nil
}

type infrastructure struct {
	temporal client.Client
	db       persistence.Persistence
}

func (i *infrastructure) Close() {
	if i.db != nil {
		if err := i.db.Close(); err != nil {
			slog.Error("failed to close database", slog.Any("error", err))
		}
	}
	if i.temporal != nil {
		i.temporal.Close()
	}
}

func bootstrapInfrastructure(ctx context.Context, conf *config.Config) (*infrastructure, error) {
	temporalClient, err := client.DialContext(ctx, client.Options{
		HostPort:  conf.Temporal.HostPort,
		Namespace: conf.Temporal.Namespace,
		Logger:    log.NewStructuredLogger(slog.Default()),
	})
	if err != nil {
		return nil, fmt.Errorf("connect to temporal: %w", err)
	}

	db, err := persistence.New(ctx, &conf.Persistence)
	if err != nil {
		temporalClient.Close()
		return nil, fmt.Errorf("create persistence handle: %w", err)
	}

	return &infrastructure{
		temporal: temporalClient,
		db:       db,
	}, nil
}

func startAppService(ctx context.Context, infra *infrastructure, conf *config.Config, v *validator.Validate) (Service, error) {
	appInstance, err := app.New(conf, infra.temporal, infra.db, v)
	if err != nil {
		return nil, err
	}
	return server.New(api.NewRouter(appInstance), conf), nil
}

func startRedditService(ctx context.Context, infra *infrastructure, conf *config.Config, _ *validator.Validate) (Service, error) {
	redditClient, err := reddit.New(ctx, conf.Reddit.ClientID, conf.Reddit.ClientSecret, conf.Reddit.UserAgent)
	if err != nil {
		return nil, err
	}
	return redditor.New(infra.temporal, infra.db, redditClient)
}

func startDigestService(ctx context.Context, infra *infrastructure, conf *config.Config, _ *validator.Validate) (Service, error) {
	return digester.New(ctx, infra.temporal, infra.db, conf)
}
