package persistence

import (
	"context"
	"errors"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence/entity"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence/mysql"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence/postgres"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence/sqlite"
)

type Persistence interface {
	Close(ctx context.Context) error
	LoadConfigurationAndState(ctx context.Context, in *entity.LoadConfigurationAndStateInput) (*entity.LoadConfigurationAndStateOutput, error)
	UpdateState(ctx context.Context, in *entity.UpdateStateInput) (*entity.UpdateStateOutput, error)
	GetSchedule(ctx context.Context, in *entity.GetScheduleInput) (*entity.GetScheduleOutput, error)
	CreateSchedule(ctx context.Context, in *entity.CreateScheduleInput) (*entity.CreateScheduleOutput, error)
	DeleteSchedule(ctx context.Context, in *entity.DeleteScheduleInput) (*entity.DeleteScheduleOutput, error)
	ListSchedules(ctx context.Context, in *entity.ListSchedulesInput) (*entity.ListSchedulesOutput, error)
	UpdateSchedule(ctx context.Context, in *entity.UpdateScheduleInput) (*entity.UpdateScheduleOutput, error)
}

var ErrUnsupportedPersistenceDriver = errors.New("unsupported persistence driver")

func New(ctx context.Context, config *config.Persistence) (Persistence, error) {
	switch config.Driver {
	case "postgres":
		handle, err := postgres.NewHandle(ctx, config)
		if err != nil {
			return nil, err
		}
		return handle, nil
	case "mysql":
		handle, err := mysql.NewHandle(ctx, config)
		if err != nil {
			return nil, err
		}
		return handle, nil
	case "sqlite":
		handle, err := sqlite.NewHandle(ctx, config)
		if err != nil {
			return nil, err
		}
		return handle, nil
	default:
		return nil, ErrUnsupportedPersistenceDriver
	}
}
