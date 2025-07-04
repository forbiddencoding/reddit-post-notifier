package app

import (
	"context"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence"
	"github.com/forbiddencoding/reddit-post-notifier/services/app/reddit"
	"github.com/go-playground/validator/v10"
	"github.com/sony/sonyflake/v2"
	"go.temporal.io/sdk/client"
)

type App struct {
	config      *config.Config
	temporal    client.Client
	persistence persistence.Persistence
	sonyflake   *sonyflake.Sonyflake
	validator   *validator.Validate
	// ---
	redditService reddit.Servicer
}

func New(
	config *config.Config,
	temporalClient client.Client,
	persistence persistence.Persistence,
	validator *validator.Validate,
) (*App, error) {
	var st sonyflake.Settings
	sf, err := sonyflake.New(st)
	if err != nil {
		return nil, err
	}

	redditService, err := reddit.NewService(persistence, temporalClient, sf, validator)
	if err != nil {
		return nil, err
	}

	return &App{
		config:        config,
		temporal:      temporalClient,
		persistence:   persistence,
		sonyflake:     sf,
		validator:     validator,
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

func (a *App) Validator() *validator.Validate {
	return a.validator
}
