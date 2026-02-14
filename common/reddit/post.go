package reddit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

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
			Children []struct {
				Data Post `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}

	GetPostsInput struct {
		Keyword           string
		Subreddit         string
		Sort              string
		Before            string
		IncludeNSFW       bool
		RestrictSubreddit bool
	}

	GetPostsOutput struct {
		Posts  []Post
		Before string
	}
)

type RateLimitError struct {
	Message           string
	SecondsUntilReset int
}

func (e RateLimitError) Error() string {
	return e.Message
}

func (e RateLimitError) GetReset() int {
	return e.SecondsUntilReset
}

var RateLimitErr = errors.New("rate limit exceeded")

func (c *Client) GetPosts(ctx context.Context, in *GetPostsInput) (*GetPostsOutput, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("https://oauth.reddit.com/r/%s/search", in.Subreddit),
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.URL.Query().Add("limit", "10")

	r := req.URL.Query()
	r.Add("q", in.Keyword)

	if in.RestrictSubreddit {
		r.Add("restrict_sr", "1")
	}

	if in.IncludeNSFW {
		r.Add("include_over_18", "on")
	}

	if in.Before != "" {
		r.Add("before", in.Before)
	}

	r.Add("sort", in.Sort)
	req.URL.RawQuery = r.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		rLErr := &RateLimitError{
			Message: "rate limit exceeded",
		}

		if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
			seconds, err := strconv.Atoi(reset)
			if err == nil {
				rLErr.SecondsUntilReset = seconds
			}
		}

		return nil, RateLimitErr
	case http.StatusOK:
		var response Response

		if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, err
		}

		posts := make([]Post, 0, len(response.Data.Children))
		for _, child := range response.Data.Children {
			posts = append(posts, child.Data)
		}

		out := &GetPostsOutput{
			Posts: posts,
		}

		if len(response.Data.Children) > 0 {
			out.Before = response.Data.Children[0].Data.Name
		}

		return out, nil
	default:
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func (p *Post) SanitizeThumbnail() string {
	if strings.HasPrefix(p.Thumbnail, "https://www.reddit.com/media?url=") {
		parsed, err := url.Parse(p.Thumbnail)
		if err == nil {
			decoded := parsed.Query().Get("url")
			if decoded != "" {
				return decoded
			}
		}
	}

	raw := html.UnescapeString(p.Thumbnail)

	if strings.HasPrefix(raw, "http://") {
		raw = strings.Replace(raw, "http://", "https://", 1)
	}

	u := strings.ToLower(raw)

	if strings.HasSuffix(u, ".jpg") ||
		strings.HasSuffix(u, ".jpeg") ||
		strings.HasSuffix(u, ".png") ||
		strings.HasSuffix(u, ".gif") ||
		strings.HasPrefix(u, "https://i.redd.it/") {
		return raw
	}

	return ""
}

func (p *Post) GetPermalink() string {
	return fmt.Sprintf("https://www.reddit.com%s", p.Permalink)
}
