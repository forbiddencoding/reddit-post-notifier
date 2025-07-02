package reddit

import (
	"context"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"net/http"
	"time"
)

type (
	Client struct {
		httpClient *http.Client
	}

	userAgentRoundTripper struct {
		userAgent string
		next      http.RoundTripper
	}
)

func (urt *userAgentRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", urt.userAgent)
	}
	return urt.next.RoundTrip(req)
}

func New(ctx context.Context, clientID, clientSecret, userAgent string) (*Client, error) {
	conf := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     "https://www.reddit.com/api/v1/access_token",
		AuthStyle:    oauth2.AuthStyleInHeader,
	}

	oauth2HttpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &userAgentRoundTripper{
			userAgent: userAgent,
			next:      http.DefaultTransport,
		},
	}

	tokenCtx := context.WithValue(ctx, oauth2.HTTPClient, oauth2HttpClient)
	tokenSrc := conf.TokenSource(tokenCtx)

	_, err := tokenSrc.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain initial OAuth2 token: %w", err)
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &oauth2.Transport{
				Source: tokenSrc,
				Base: &userAgentRoundTripper{
					userAgent: userAgent,
					next:      http.DefaultTransport,
				},
			},
		},
	}, nil
}
