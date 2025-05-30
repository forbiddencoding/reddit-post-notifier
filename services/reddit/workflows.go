package reddit

import (
	"errors"
	"fmt"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"time"
)

type (
	PostsWorkflowInput struct {
		Keyword    string   `json:"keyword"`
		Subreddits []string `json:"subreddits,omitempty"`
	}

	PostsWorkflowOutput struct {
		Before string `json:"before"`
	}
)

func PostsWorkflow(ctx workflow.Context, in *PostsWorkflowInput) (*PostsWorkflowOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("PostsWorkflow started")

	var out PostsWorkflowOutput
	if workflow.HasLastCompletionResult(ctx) {
		if err := workflow.GetLastCompletionResult(ctx, &out); err != nil {
			logger.Error("Failed to get last completion result", "error", err)
			return nil, err
		}
	}

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	})

	maxRetries := 5
	retryCount := 0

	for retryCount < maxRetries {
		var result GetPostsOutput
		err := workflow.ExecuteActivity(ctx, (&Activities{}).GetPosts, &GetPostsInput{
			Keyword: in.Keyword,
			After:   out.Before,
		}).Get(ctx, &result)

		if err == nil {
			out.Before = result.Before
			break
		}

		var appErr *temporal.ApplicationError
		if errors.As(err, &appErr) {
			if !appErr.NonRetryable() {
				retryCount++
				logger.Warn("Rate limit exceeded, retrying", "attempt", retryCount, "error", err)
				_ = workflow.Sleep(ctx, appErr.NextRetryDelay())
				continue
			}

			return nil, appErr
		}

		return nil, err
	}

	if retryCount == maxRetries {
		return nil, fmt.Errorf("exceeded max rate limited retries")
	}

	if err := workflow.ExecuteActivity(ctx, (&Activities{}).SendNotification, &SendNotificationInput{}).Get(ctx, nil); err != nil {
		logger.Error("Failed to send notification", "error", err)
		return nil, err
	}

	return &PostsWorkflowOutput{}, nil
}
