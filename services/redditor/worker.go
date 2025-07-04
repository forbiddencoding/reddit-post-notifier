package redditor

import (
	"context"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	"github.com/forbiddencoding/reddit-post-notifier/common/temporalx"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type Worker struct {
	worker worker.Worker
}

func New(ctx context.Context, client client.Client, conf *config.Config) (*Worker, error) {
	options := worker.Options{
		// The only activities here is the GetPosts activity. Each activity does exactly one http request, with retries
		// handled by temporal. A value of 1.67 is equal to 100 req/min. We use 1.6 to try and avoid hitting the rate
		// limit proactively.
		WorkerActivitiesPerSecond: 1.6,
	}

	w := worker.New(client, "reddit", options)
	worker.EnableVerboseLogging(false)

	activities, err := NewActivities(ctx, conf)
	if err != nil {
		return nil, err
	}

	w.RegisterWorkflowWithOptions(PostWorkflow, workflow.RegisterOptions{Name: "post"})
	w.RegisterActivityWithOptions(activities.GetPosts, activity.RegisterOptions{Name: GetPostsActivityName})

	return &Worker{
		worker: w,
	}, nil
}

func (w *Worker) Start(ctx context.Context) error {
	return w.worker.Run(temporalx.WorkerInterruptFromCtxChan(ctx))
}

func (w *Worker) Close(ctx context.Context) error {
	w.worker.Stop()
	return nil
}
