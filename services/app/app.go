package app

import (
	"context"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence"
	"github.com/forbiddencoding/reddit-post-notifier/services/app/reddit"
	"github.com/sony/sonyflake/v2"
	"go.temporal.io/sdk/client"
)

type App struct {
	config      *config.Config
	temporal    client.Client
	persistence persistence.Persistence
	sonyflake   *sonyflake.Sonyflake
	// ---
	redditService reddit.Servicer
}

func New(
	config *config.Config,
	temporalClient client.Client,
	persistence persistence.Persistence,
) (*App, error) {
	var st sonyflake.Settings
	sf, err := sonyflake.New(st)
	if err != nil {
		return nil, err
	}

	redditService, err := reddit.NewService(persistence, temporalClient, sf)
	if err != nil {
		return nil, err
	}

	return &App{
		config:        config,
		temporal:      temporalClient,
		persistence:   persistence,
		sonyflake:     sf,
		redditService: redditService,
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	return nil
}

func (a *App) Close(ctx context.Context) error {
	return nil
}

func (a *App) ScheduleService() reddit.Servicer {
	return a.redditService
}
