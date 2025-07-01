package redditor

import (
	"context"
	"errors"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence/entity"
	"github.com/forbiddencoding/reddit-post-notifier/common/reddit"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"time"
)

type Activities struct {
	client *reddit.Client
}

func NewActivities(ctx context.Context, conf *config.Config) (*Activities, error) {
	client := reddit.New(ctx, conf.Reddit.ClientID, conf.Reddit.ClientSecret, conf.Reddit.UserAgent)

	return &Activities{
		client: client,
	}, nil
}

type (
	GetPostsInput struct {
		Keyword   string            `json:"keyword"`
		Subreddit *entity.Subreddit `json:"subreddit"`
	}

	GetPostsOutput struct {
		Posts  []reddit.Post `json:"posts"`
		Before string        `json:"before,omitzero"`
	}
)

const GetPostsActivityName = "get_posts"

func (a *Activities) GetPosts(ctx context.Context, in *GetPostsInput) (*GetPostsOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("GetPosts started", "subreddit", in.Subreddit.Name, "keyword", in.Keyword)

	res, err := a.client.GetPosts(
		ctx,
		&reddit.GetPostsInput{
			Keyword:           in.Keyword,
			Subreddit:         in.Subreddit.Name,
			Sort:              in.Subreddit.Sort,
			Before:            in.Subreddit.Before,
			IncludeNSFW:       in.Subreddit.IncludeNSFW,
			RestrictSubreddit: in.Subreddit.RestrictSubreddit,
		},
	)
	if err != nil {
		var rLErr reddit.RateLimitError
		if errors.As(err, &rLErr) {
			logger.Info("GetPosts hit rate limit")

			errOpts := temporal.ApplicationErrorOptions{
				Cause:        err,
				NonRetryable: false,
			}

			if delay := rLErr.GetReset(); delay > 0 {
				errOpts.NextRetryDelay = time.Duration(delay) * time.Second
			}

			return nil, temporal.NewApplicationErrorWithOptions("rate limit exceeded", "api", errOpts)
		}
		return nil, err
	}

	return &GetPostsOutput{
		Posts:  res.Posts,
		Before: res.Before,
	}, nil
}
