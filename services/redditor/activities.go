package redditor

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence"
	"github.com/forbiddencoding/reddit-post-notifier/common/reddit"
	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"time"
)

type activities struct {
	client      *reddit.Client
	persistence persistence.Persistence
}

func newActivities(persistence persistence.Persistence, client *reddit.Client) (*activities, error) {
	return &activities{
		persistence: persistence,
		client:      client,
	}, nil
}

type (
	GetPostsInput struct {
		ConfigurationID uuid.UUID              `json:"configuration_id"`
		Keyword         string                 `json:"keyword"`
		Subreddit       *persistence.Subreddit `json:"subreddit"`
	}

	GetPostsOutput struct {
		HasMore bool   `json:"has_more"`
		Before  string `json:"before,omitzero"`
	}
)

const GetPostsActivityName = "get_posts"

func (a *activities) GetPosts(ctx context.Context, in *GetPostsInput) (*GetPostsOutput, error) {
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
		if target, ok := errors.AsType[reddit.RateLimitError](err); ok {
			logger.Info("hit rate limit")

			opts := temporal.ApplicationErrorOptions{
				Cause:        err,
				NonRetryable: false,
			}

			if delay := target.GetReset(); delay > 0 {
				opts.NextRetryDelay = time.Duration(delay) * time.Second
			}

			return nil, temporal.NewApplicationErrorWithOptions("rate limit exceeded", "api", opts)
		}
		return nil, err
	}

	items := make([]persistence.QueueItem, 0, len(res.Posts))
	for _, p := range res.Posts {
		id, err := uuid.NewV7()
		if err != nil {
			return nil, err
		}

		createdTime := time.Unix(int64(p.CreatedUTC), 0)

		post, err := json.Marshal(persistence.Post{
			ID:        p.ID,
			Title:     p.Title,
			URL:       p.URL,
			Subreddit: p.Subreddit,
			NSFW:      p.NSFW,
			Spoiler:   p.Spoiler,
			Ups:       p.Ups,
			Downs:     p.Downs,
			Thumbnail: p.SanitizeThumbnail(),
			Created:   createdTime.Format(time.RFC822),
			Permalink: p.GetPermalink(),
		})
		if err != nil {
			return nil, err
		}

		items = append(items, persistence.QueueItem{
			ID:              id,
			ConfigurationID: in.ConfigurationID,
			Post:            post,
			CreatedTime:     createdTime,
		})
	}

	if len(items) == 0 {
		return &GetPostsOutput{}, nil
	}

	if _, err = a.persistence.QueuePosts(ctx, &persistence.QueuePostsInput{
		Items: items,
	}); err != nil {
		return nil, err
	}

	return &GetPostsOutput{
		HasMore: len(res.Posts) > 0,
		Before:  res.Before,
	}, nil
}
