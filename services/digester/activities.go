package digester

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	"github.com/forbiddencoding/reddit-post-notifier/common/mail"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence"
	"github.com/google/uuid"
	"html/template"
	"sort"
	"time"
)

type Activities struct {
	mailer      mail.Mailer
	persistence persistence.Persistence
}

func NewActivities(ctx context.Context, persistence persistence.Persistence, conf *config.Config) (*Activities, error) {
	mailer, err := mail.New(ctx, &conf.Mailer)
	if err != nil {
		return nil, err
	}

	return &Activities{
		mailer:      mailer,
		persistence: persistence,
	}, nil
}

type (
	LoadConfigurationAndStateInput struct {
		ID uuid.UUID `json:"id"`
	}

	LoadConfigurationAndStateOutput struct {
		Keyword    string                   `json:"keyword"`
		Recipients []*persistence.Recipient `json:"recipients"`
		Subreddits []*persistence.Subreddit `json:"subreddits,omitempty"`
	}
)

const LoadConfigurationAndStateActivityName = "load_configuration_and_state"

func (a *Activities) LoadConfigurationAndState(ctx context.Context, in *LoadConfigurationAndStateInput) (*LoadConfigurationAndStateOutput, error) {
	state, err := a.persistence.LoadConfigurationAndState(ctx, &persistence.LoadConfigurationAndStateInput{
		ID: in.ID,
	})
	if err != nil {
		return nil, err
	}

	subreddits := make([]*persistence.Subreddit, 0, len(state.Subreddits))
	for _, sr := range state.Subreddits {
		subreddits = append(subreddits, &persistence.Subreddit{
			ID:                sr.ID,
			Name:              sr.Name,
			IncludeNSFW:       sr.IncludeNSFW,
			Sort:              sr.Sort,
			RestrictSubreddit: sr.RestrictSubreddit,
			Before:            sr.Before,
		})
	}

	return &LoadConfigurationAndStateOutput{
		Keyword:    state.Keyword,
		Subreddits: subreddits,
		Recipients: state.Recipients,
	}, nil
}

type (
	SendNotificationInput struct {
		ConfigurationID uuid.UUID                `json:"configuration_id"`
		Keyword         string                   `json:"keyword"`
		Recipients      []*persistence.Recipient `json:"recipients"`
	}

	SendNotificationOutput struct {
	}
)

const SendNotificationActivityName = "send_notification"

func (a *Activities) SendNotification(ctx context.Context, in *SendNotificationInput) (*SendNotificationOutput, error) {
	var addresses = make([]string, 0, len(in.Recipients))
	for _, recipient := range in.Recipients {
		addresses = append(addresses, recipient.Address)
	}

	items, err := a.persistence.GetPosts(ctx, &persistence.GetPostsInput{
		ConfigurationID: in.ConfigurationID,
	})
	if err != nil {
		return nil, fmt.Errorf("get posts from queue: %w", err)
	}

	var posts = make([]persistence.Post, 0, len(items.Items))
	for _, item := range items.Items {
		var post persistence.Post
		if err = json.Unmarshal(item.Post, &post); err != nil {
			return nil, fmt.Errorf("unmarshal post: %w", err)
		}

		posts = append(posts, post)
	}

	sort.SliceStable(posts, func(i, j int) bool {
		iCreated, _ := time.Parse(posts[i].Created, time.RFC822)
		jCreated, _ := time.Parse(posts[i].Created, time.RFC822)

		return iCreated.Before(jCreated)
	})

	data := map[string]any{
		"Title": "New Reddit Posts Notification",
		"Posts": posts,
	}

	tmpl, err := template.ParseGlob("templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse email template: %w", err)
	}

	var body bytes.Buffer
	if err = tmpl.ExecuteTemplate(&body, "email", data); err != nil {
		return nil, fmt.Errorf("execute email template: %w", err)
	}

	if err = a.mailer.SendMail(
		ctx,
		addresses,
		in.Keyword,
		body.String(),
	); err != nil {
		return nil, fmt.Errorf("send mail: %w", err)
	}

	if _, err = a.persistence.PopPosts(ctx, &persistence.PopPostsInput{
		ConfigurationID: in.ConfigurationID,
	}); err != nil {
		return nil, fmt.Errorf("pop posts from queue: %w", err)
	}

	return nil, nil
}

type (
	UpdateStateInput struct {
		Subreddits []*persistence.Subreddit `json:"subreddits"`
	}

	UpdateStateOutput struct {
	}
)

const UpdateStateActivityName = "update_state"

func (a *Activities) UpdateState(ctx context.Context, in *UpdateStateInput) (*UpdateStateOutput, error) {
	values := make([]*persistence.UpdateStateValue, 0, len(in.Subreddits))
	for _, sr := range in.Subreddits {
		values = append(values, &persistence.UpdateStateValue{
			SubredditConfigurationID: sr.ID,
			Before:                   sr.Before,
		})
	}
	_, err := a.persistence.UpdateState(ctx, &persistence.UpdateStateInput{
		Values: values,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update state: %w", err)
	}

	return nil, nil
}
