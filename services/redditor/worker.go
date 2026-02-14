package redditor

import (
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence"
	"github.com/forbiddencoding/reddit-post-notifier/common/reddit"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type Worker struct {
	worker worker.Worker
}

func New(client client.Client, persistence persistence.Persistence, reddit *reddit.Client) (*Worker, error) {
	options := worker.Options{
		// The only activities here is the GetPosts activity. Each activity does exactly one http request, with retries
		// handled by temporal. A value of 1.67 is equal to 100 req/min. We use 1.6 to try and avoid hitting the rate
		// limit proactively.
		TaskQueueActivitiesPerSecond: 1.6,
	}

	w := worker.New(client, "reddit", options)
	worker.EnableVerboseLogging(false)

	act, err := newActivities(persistence, reddit)
	if err != nil {
		return nil, err
	}

	w.RegisterWorkflowWithOptions(PostWorkflow, workflow.RegisterOptions{Name: "post"})
	w.RegisterActivityWithOptions(act.GetPosts, activity.RegisterOptions{Name: GetPostsActivityName})

	return &Worker{
		worker: w,
	}, nil
}

func (w *Worker) Start() error {
	return w.worker.Start()
}

func (w *Worker) Close() error {
	w.worker.Stop()
	return nil
}
