package digester

import (
	"context"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence"
	"github.com/forbiddencoding/reddit-post-notifier/common/temporalx"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type Worker struct {
	worker worker.Worker
}

func New(ctx context.Context, client client.Client, persistence persistence.Persistence, conf *config.Config) (*Worker, error) {
	options := worker.Options{}

	w := worker.New(client, "digest", options)
	worker.EnableVerboseLogging(false)

	activities, err := NewActivities(ctx, persistence, conf)
	if err != nil {
		return nil, err
	}

	w.RegisterWorkflowWithOptions(DigestWorkflow, workflow.RegisterOptions{Name: "digest"})
	w.RegisterActivityWithOptions(activities.LoadConfigurationAndState, activity.RegisterOptions{Name: LoadConfigurationAndStateActivityName})
	w.RegisterActivityWithOptions(activities.SendNotification, activity.RegisterOptions{Name: SendNotificationActivityName})
	w.RegisterActivityWithOptions(activities.UpdateState, activity.RegisterOptions{Name: UpdateStateActivityName})

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
