package reddit

import (
	"context"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type Worker struct {
	worker worker.Worker
}

func New(ctx context.Context) (*Worker, error) {
	options := worker.Options{}

	w := worker.New(nil, "reddit", options)
	worker.EnableVerboseLogging(false)

	activities := NewActivities(ctx)

	w.RegisterWorkflowWithOptions(PostsWorkflow, workflow.RegisterOptions{Name: "posts"})
	w.RegisterActivityWithOptions(activities.GetPosts, activity.RegisterOptions{Name: "get_posts"})
	w.RegisterActivityWithOptions(activities.SendNotification, activity.RegisterOptions{Name: "send_notification"})

	return &Worker{
		worker: w,
	}, nil
}
