package reddit

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

func RunWorker(ctx context.Context, client client.Client, persistence persistence.Persistence, conf *config.Config) error {
	options := worker.Options{}

	w := worker.New(client, "reddit", options)
	worker.EnableVerboseLogging(false)

	activities := NewActivities(ctx, persistence, conf)

	w.RegisterWorkflowWithOptions(DigestWorkflow, workflow.RegisterOptions{Name: "digest"})
	w.RegisterActivityWithOptions(activities.LoadConfigurationAndState, activity.RegisterOptions{Name: LoadConfigurationAndStateActivityName})
	w.RegisterActivityWithOptions(activities.SendNotification, activity.RegisterOptions{Name: SendNotificationActivityName})
	w.RegisterActivityWithOptions(activities.UpdateState, activity.RegisterOptions{Name: UpdateStateActivityName})

	w.RegisterWorkflowWithOptions(PostWorkflow, workflow.RegisterOptions{Name: "post"})
	w.RegisterActivityWithOptions(activities.GetPosts, activity.RegisterOptions{Name: GetPostsActivityName})

	return w.Run(temporalx.WorkerInterruptFromCtxChan(ctx))
}
