package redditor

import (
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence"
	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"time"
)

type (
	PostWorkflowInput struct {
		ConfigurationID uuid.UUID              `json:"configuration_id"`
		Keyword         string                 `json:"keyword"`
		Subreddit       *persistence.Subreddit `json:"subreddit"`
	}

	PostWorkflowOutput struct {
		Before string `json:"before"`
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
		ConfigurationID: in.ConfigurationID,
		Keyword:         in.Keyword,
		Subreddit:       in.Subreddit,
	}).Get(ctx, &result)
	if err != nil {
		return nil, err
	}

	if !result.HasMore {
		return &PostWorkflowOutput{
			Before: in.Subreddit.Before,
		}, nil
	}

	input := PostWorkflowInput{
		ConfigurationID: in.ConfigurationID,
		Keyword:         in.Keyword,
		Subreddit:       in.Subreddit,
	}

	input.Subreddit.Before = result.Before

	return nil, workflow.NewContinueAsNewError(ctx, PostWorkflow, &input)
}
