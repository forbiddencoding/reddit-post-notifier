package digester

import (
	"fmt"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence/entity"
	redditSDK "github.com/forbiddencoding/reddit-post-notifier/common/reddit"
	"github.com/forbiddencoding/reddit-post-notifier/services/redditor"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"time"
)

type (
	DigestWorkflowInput struct {
		ID int64 `json:"id"`
	}

	DigestWorkflowOutput struct {
	}
)

func DigestWorkflow(ctx workflow.Context, in *DigestWorkflowInput) (*DigestWorkflowOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("DigestWorkflow started")

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		TaskQueue:           "digest",
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    10 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    2 * time.Minute,
			MaximumAttempts:    5,
		},
	})

	var configuration LoadConfigurationAndStateOutput
	if err := workflow.ExecuteActivity(
		ctx,
		LoadConfigurationAndStateActivityName,
		&LoadConfigurationAndStateInput{
			ID: in.ID,
		},
	).Get(ctx, &configuration); err != nil {
		logger.Error("Failed to load configuration and state", "error", err)
		return nil, err
	}

	var (
		subreddits = make([]*entity.Subreddit, 0, len(configuration.Subreddits))
		posts      []redditSDK.Post
	)

	for i := 0; i < len(configuration.Subreddits); i++ {
		subreddit := configuration.Subreddits[i]

		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			TaskQueue:                "reddit",
			WorkflowID:               fmt.Sprintf("reddit_post_workflow::%s::%s", configuration.Keyword, subreddit.Name),
			ParentClosePolicy:        enums.PARENT_CLOSE_POLICY_TERMINATE,
			WorkflowExecutionTimeout: 15 * time.Minute,
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts:    5,
				InitialInterval:    time.Second,
				BackoffCoefficient: 2.0,
				MaximumInterval:    100 * time.Second,
			},
		})

		var result redditor.PostWorkflowOutput
		err := workflow.ExecuteChildWorkflow(childCtx, redditor.PostWorkflow, &redditor.PostWorkflowInput{
			Keyword:   configuration.Keyword,
			Subreddit: subreddit,
		}).Get(ctx, &result)
		if err != nil {
			logger.Error("Failed to execute PostWorkflow", "error", err, "subreddit", subreddit.Name)
			return nil, err
		}

		if len(result.Posts) > 0 {
			posts = append(posts, result.Posts...)
			subreddits = append(subreddits, &entity.Subreddit{
				ID:     subreddit.ID,
				Before: result.Subreddit.Before,
			})
		}
	}

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		TaskQueue:           "digest",
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	})

	if err := workflow.ExecuteActivity(ctx, SendNotificationActivityName, &SendNotificationInput{
		Posts:      posts,
		Recipients: configuration.Recipients,
	}).Get(ctx, nil); err != nil {
		logger.Error("Failed to send notification", "error", err)
		return nil, err
	}

	if err := workflow.ExecuteActivity(ctx, UpdateStateActivityName, &UpdateStateInput{
		Subreddits: subreddits,
	}).Get(ctx, nil); err != nil {
		logger.Error("Failed to update state", "error", err)
		return nil, err
	}

	return nil, nil
}
