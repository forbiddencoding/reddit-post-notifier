package reddit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	"github.com/forbiddencoding/reddit-post-notifier/common/mail"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence/entity"
	"github.com/forbiddencoding/reddit-post-notifier/common/reddit"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"html/template"
	"time"
)

type Activities struct {
	client      *reddit.Client
	mailer      mail.Mailer
	persistence persistence.Persistence
}

func NewActivities(ctx context.Context, persistence persistence.Persistence, conf *config.Config) (*Activities, error) {
	client := reddit.New(ctx, conf.Reddit.ClientID, conf.Reddit.ClientSecret, conf.Reddit.UserAgent)

	mailer, err := mail.New(ctx, &conf.Mailer)
	if err != nil {
		return nil, err
	}

	return &Activities{
		client:      client,
		mailer:      mailer,
		persistence: persistence,
	}, nil
}

type (
	LoadConfigurationAndStateInput struct {
		ID int64 `json:"id"`
	}

	LoadConfigurationAndStateOutput struct {
		Keyword    string              `json:"keyword"`
		Recipients []*entity.Recipient `json:"recipients"`
		Subreddits []*Subreddit        `json:"subreddits,omitempty"`
	}
)

const LoadConfigurationAndStateActivityName = "load_configuration_and_state"

func (a *Activities) LoadConfigurationAndState(ctx context.Context, in *LoadConfigurationAndStateInput) (*LoadConfigurationAndStateOutput, error) {
	state, err := a.persistence.LoadConfigurationAndState(ctx, &entity.LoadConfigurationAndStateInput{
		ID: in.ID,
	})
	if err != nil {
		return nil, err
	}

	subreddits := make([]*Subreddit, 0, len(state.Subreddits))
	for _, sr := range state.Subreddits {
		subreddits = append(subreddits, &Subreddit{
			SubredditID:       sr.ID,
			Name:              sr.Name,
			IncludeNSFW:       sr.IncludeNSFW,
			Sort:              sr.Sort,
			RestrictSubreddit: sr.RestrictSubreddit,
			Before:            sr.After,
		})
	}

	return &LoadConfigurationAndStateOutput{
		Keyword:    state.Keyword,
		Subreddits: subreddits,
		Recipients: state.Recipients,
	}, nil
}

type (
	Subreddit struct {
		SubredditID       int64  `json:"subreddit_id"`
		Name              string `json:"name"`
		IncludeNSFW       bool   `json:"include_nsfw"`
		Sort              string `json:"sort"`
		RestrictSubreddit bool   `json:"restrict_subreddit"`
		Before            string `json:"before,omitzero"`
	}

	GetPostsInput struct {
		Keyword   string     `json:"keyword"`
		Subreddit *Subreddit `json:"subreddit"`
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
		if errors.Is(err, reddit.RateLimitErr) {
			logger.Info("GetPosts hit rate limit")

			return nil, temporal.NewApplicationErrorWithOptions("rate limit exceeded", "api", temporal.ApplicationErrorOptions{
				Cause:        err,
				NonRetryable: false,
			})
		}
		return nil, err
	}

	return &GetPostsOutput{
		Posts:  res.Posts,
		Before: res.Before,
	}, nil
}

type (
	SendNotificationInput struct {
		Posts      []reddit.Post       `json:"posts"`
		Recipients []*entity.Recipient `json:"recipients"`
	}

	SendNotificationOutput struct {
	}
)

const SendNotificationActivityName = "send_notification"

func (a *Activities) SendNotification(ctx context.Context, in *SendNotificationInput) (*SendNotificationOutput, error) {
	type (
		PostView struct {
			ID         string `json:"id"`
			Title      string `json:"title"`
			URL        string `json:"url"`
			Subreddit  string `json:"subreddit"`
			NSFW       bool   `json:"nsfw"`
			Spoiler    bool   `json:"spoiler"`
			Ups        int    `json:"ups"`
			Downs      int    `json:"downs"`
			Thumbnail  string `json:"thumbnail"`
			CreatedStr string `json:"created_str"`
			Permalink  string `json:"permalink"`
		}
	)

	var addresses []string
	for _, recipient := range in.Recipients {
		addresses = append(addresses, recipient.Address)
	}

	postViews := make([]PostView, 0, len(in.Posts))
	for _, p := range in.Posts {
		postViews = append(postViews, PostView{
			ID:         p.ID,
			Title:      p.Title,
			URL:        p.URL,
			Subreddit:  p.Subreddit,
			NSFW:       p.NSFW,
			Spoiler:    p.Spoiler,
			Ups:        p.Ups,
			Downs:      p.Downs,
			Thumbnail:  p.SanitizeThumbnail(),
			CreatedStr: time.Unix(int64(p.CreatedUTC), 0).Format(time.RFC822),
			Permalink:  p.GetPermalink(),
		})
	}

	data := map[string]any{
		"Title": "New Reddit Posts Notification",
		"Posts": postViews,
	}

	tmpl, err := template.ParseGlob("templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse email template: %w", err)
	}

	var body bytes.Buffer
	if err = tmpl.ExecuteTemplate(&body, "email", data); err != nil {
		return nil, fmt.Errorf("failed to execute email template: %w", err)
	}

	if err = a.mailer.SendMail(
		ctx,
		addresses,
		"New Reddit Posts Notification",
		body.String(),
	); err != nil {
		return nil, fmt.Errorf("failed to send mail: %w", err)
	}
	return nil, nil
}

type (
	UpdateStateInput struct {
		Subreddits []*Subreddit `json:"subreddits"`
	}

	UpdateStateOutput struct {
	}
)

const UpdateStateActivityName = "update_state"

func (a *Activities) UpdateState(ctx context.Context, in *UpdateStateInput) (*UpdateStateOutput, error) {
	values := make([]*entity.UpdateStateValue, 0, len(in.Subreddits))
	for _, sr := range in.Subreddits {
		values = append(values, &entity.UpdateStateValue{
			SubredditConfigurationID: sr.SubredditID,
			Before:                   sr.Before,
		})
	}
	_, err := a.persistence.UpdateState(ctx, &entity.UpdateStateInput{
		Values: values,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update state: %w", err)
	}

	return nil, nil
}
