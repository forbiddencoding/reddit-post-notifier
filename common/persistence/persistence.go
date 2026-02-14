package persistence

import (
	"context"
	"io"
)

type Persistence interface {
	io.Closer
	LoadConfigurationAndState(ctx context.Context, in *LoadConfigurationAndStateInput) (*LoadConfigurationAndStateOutput, error)
	UpdateState(ctx context.Context, in *UpdateStateInput) (*UpdateStateOutput, error)
	GetSchedule(ctx context.Context, in *GetScheduleInput) (*GetScheduleOutput, error)
	CreateSchedule(ctx context.Context, in *CreateScheduleInput) (*CreateScheduleOutput, error)
	DeleteSchedule(ctx context.Context, in *DeleteScheduleInput) (*DeleteScheduleOutput, error)
	ListSchedules(ctx context.Context, in *ListSchedulesInput) (*ListSchedulesOutput, error)
	UpdateSchedule(ctx context.Context, in *UpdateScheduleInput) (*UpdateScheduleOutput, error)
	QueuePosts(ctx context.Context, in *QueuePostsInput) (*QueuePostsOutput, error)
	GetPosts(ctx context.Context, in *GetPostsInput) (*GetPostsOutput, error)
	PopPosts(ctx context.Context, in *PopPostsInput) (*PopPostsOutput, error)
}
