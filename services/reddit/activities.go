package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/forbiddencoding/reddit-post-notifier/common/reddit"
	"go.temporal.io/sdk/temporal"
	"net/http"
	"strconv"
	"time"
)

type Activities struct {
	client *reddit.Client
}

func NewActivities(ctx context.Context) *Activities {
	client := reddit.New(ctx, "", "", "")

	return &Activities{
		client: client,
	}
}

type (
	GetPostsInput struct {
		Keyword string `json:"keyword"`
		After   string `json:"after,omitempty"`
	}

	GetPostsOutput struct {
		Posts  []reddit.Post `json:"posts"`
		Before string        `json:"before"`
	}
)

func (a *Activities) GetPosts(ctx context.Context, in *GetPostsInput) (*GetPostsOutput, error) {
	url := "https://oauth.reddit.com/search?q=" + in.Keyword + "&sort=new&limit=100"
	if in.After != "" {
		url += "&after=" + in.After
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", a.client.UserAgent())

	resp, err := a.client.Client().Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		reset, err := strconv.ParseFloat(resp.Header.Get("X-RateLimit-Reset"), 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse rate limit reset: %w", err)
		}
		return nil, temporal.NewApplicationErrorWithOptions("rate limit exceeded", "api", temporal.ApplicationErrorOptions{
			NonRetryable:   false,
			Cause:          fmt.Errorf("rate limit exceeded"),
			NextRetryDelay: time.Second * time.Duration(reset),
		})
	case http.StatusOK:
		var res reddit.Response
		if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return nil, err
		}

		var posts []reddit.Post
		for _, child := range res.Data.Children {
			posts = append(posts, child.Data)
		}

		return &GetPostsOutput{
			Posts:  posts,
			Before: res.Data.After,
		}, nil
	default:
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

type (
	SendNotificationInput struct {
	}

	SendNotificationOutput struct {
	}
)

func (a *Activities) SendNotification(ctx context.Context, in *SendNotificationInput) (*SendNotificationOutput, error) {
	return nil, nil
}
