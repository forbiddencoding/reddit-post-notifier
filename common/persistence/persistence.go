package persistence

import (
	"context"
)

type Persistence interface {
	LoadConfigurationAndState(ctx context.Context, in *LoadConfigurationAndStateInput) (*LoadConfigurationAndStateOutput, error)
	UpdateState(ctx context.Context, in *UpdateStateInput) (*UpdateStateOutput, error)
	GetSchedule(ctx context.Context, in *GetScheduleInput) (*GetScheduleOutput, error)
	CreateSchedule(ctx context.Context, in *CreateScheduleInput) (*CreateScheduleOutput, error)
	DeleteSchedule(ctx context.Context, in *DeleteScheduleInput) (*DeleteScheduleOutput, error)
	ListSchedules(ctx context.Context, in *ListSchedulesInput) (*ListSchedulesOutput, error)
	UpdateSchedule(ctx context.Context, in *UpdateScheduleInput) (*UpdateScheduleOutput, error)
}
