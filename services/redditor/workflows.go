package redditor

import (
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence"
	"github.com/forbiddencoding/reddit-post-notifier/common/reddit"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"time"
)

type (
	PostWorkflowInput struct {
		Keyword   string
		Subreddit *persistence.Subreddit `json:"subreddit"`
		Posts     []reddit.Post          `json:"posts"`
	}

	PostWorkflowOutput struct {
		Subreddit *persistence.Subreddit `json:"subreddit"`
		Posts     []reddit.Post          `json:"posts"`
	}
)

func PostWorkflow(ctx workflow.Context, in *PostWorkflowInput) (*PostWorkflowOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("PostsWorkflow started")

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    30 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    5 * time.Minute,
			MaximumAttempts:    5,
		},
	})

	var result GetPostsOutput
	err := workflow.ExecuteActivity(ctx, GetPostsActivityName, &GetPostsInput{
		Keyword:   in.Keyword,
		Subreddit: in.Subreddit,
	}).Get(ctx, &result)
	if err != nil {
		logger.Error("Failed to get posts", "error", err, "subreddit", in.Subreddit.Name)
		return nil, err
	}

	input := PostWorkflowInput{
		Keyword:   in.Keyword,
		Subreddit: in.Subreddit,
		Posts:     in.Posts,
	}

	if len(result.Posts) == 0 {
		return &PostWorkflowOutput{
			Subreddit: in.Subreddit,
			Posts:     in.Posts,
		}, nil
	}

	input.Posts = append(in.Posts, result.Posts...)
	input.Subreddit.Before = result.Before

	return nil, workflow.NewContinueAsNewError(ctx, PostWorkflow, &input)
}
