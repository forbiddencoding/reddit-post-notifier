package reddit

import (
	"context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"net/http"
)

type Client struct {
	httpClient *http.Client
	userAgent  string
}

func New(ctx context.Context, clientID, clientSecret, userAgent string) *Client {
	conf := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     "https://www.reddit.com/api/v1/access_token",
	}

	tokenSrc := conf.TokenSource(ctx)
	client := oauth2.NewClient(ctx, tokenSrc)

	return &Client{
		httpClient: client,
		userAgent:  userAgent,
	}
}

func (c *Client) Client() *http.Client {
	return c.httpClient
}

func (c *Client) UserAgent() string {
	return c.userAgent
}

type (
	Post struct {
		ID         string  `json:"id"`
		Title      string  `json:"title"`
		URL        string  `json:"url"`
		CreatedUTC float64 `json:"created_utc"`
		Subreddit  string  `json:"subreddit"`
		Name       string  `json:"name"`
		NSFW       bool    `json:"over_18"`
		Spoiler    bool    `json:"spoiler"`
		Ups        int     `json:"ups"`
		Downs      int     `json:"downs"`
		Thumbnail  string  `json:"thumbnail"` // "self" or a URL to an image
		Permalink  string  `json:"permalink"`
	}

	Response struct {
		Data struct {
			After    string `json:"after"`
			Children []struct {
				Data Post `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
)
