package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	"github.com/forbiddencoding/reddit-post-notifier/common/mail"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence/entity"
	"github.com/forbiddencoding/reddit-post-notifier/common/reddit"
	"go.temporal.io/sdk/temporal"
	"net/http"
	"strings"
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
		Keyword    string       `json:"keyword"`
		Subreddits []*Subreddit `json:"subreddits,omitempty"`
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
	url := fmt.Sprintf("https://oauth.reddit.com/r/%s/search", in.Subreddit.Name)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", a.client.UserAgent())
	req.URL.Query().Add("limit", "10")

	r := req.URL.Query()
	r.Add("q", in.Keyword)

	if in.Subreddit.RestrictSubreddit {
		r.Add("restrict_sr", "1")
	}

	if in.Subreddit.IncludeNSFW {
		r.Add("include_over_18", "on")
	}

	if in.Subreddit.Before != "" {
		r.Add("before", in.Subreddit.Before)
	}

	if in.Subreddit.Sort != "" {
		r.Add("sort", in.Subreddit.Sort)
	} else {
		r.Add("sort", "new")
	}
	req.URL.RawQuery = r.Encode()

	resp, err := a.client.Client().Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		// Reddit does not return any header or body when the rate limit is reached.
		// Therefore, the below code sadly does not work as intended.
		//reset, err := strconv.ParseFloat(resp.Header.Get("X-RateLimit-Reset"), 64)
		//if err != nil {
		//	return nil, fmt.Errorf("failed to parse rate limit reset: %w", err)
		//}
		//return nil, temporal.NewApplicationErrorWithOptions("rate limit exceeded", "api", temporal.ApplicationErrorOptions{
		//	NonRetryable:   false,
		//	Cause:          fmt.Errorf("rate limit exceeded"),
		//	NextRetryDelay: time.Second * time.Duration(reset),
		//})
		return nil, temporal.NewApplicationErrorWithOptions("rate limit exceeded", "api", temporal.ApplicationErrorOptions{
			Cause:        fmt.Errorf("rate limit exceeded"),
			NonRetryable: false,
		})
	case http.StatusOK:
		var res reddit.Response
		if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return nil, err
		}

		posts := make([]reddit.Post, 0, len(res.Data.Children))
		for _, child := range res.Data.Children {
			posts = append(posts, child.Data)
		}

		var before string
		if len(res.Data.Children) > 0 {
			before = res.Data.Children[0].Data.Name
		}

		return &GetPostsOutput{
			Posts:  posts,
			Before: before,
		}, nil
	default:
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

type (
	SendNotificationInput struct {
		Posts []reddit.Post `json:"posts"`
	}

	SendNotificationOutput struct {
	}
)

const SendNotificationActivityName = "send_notification"

func (a *Activities) SendNotification(ctx context.Context, in *SendNotificationInput) (*SendNotificationOutput, error) {
	sb := strings.Builder{}

	for _, post := range in.Posts {
		sb.WriteString(fmt.Sprintf("Title: %s\nURL: %s\n\n", post.Title, post.URL))
	}

	if err := a.mailer.SendMail(
		ctx,
		[]string{""},
		"New Reddit Posts Notification",
		sb.String(),
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
