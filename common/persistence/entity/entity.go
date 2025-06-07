package entity

type (
	Subreddit struct {
		ID                int64  `json:"id"`
		Name              string `json:"name"`
		IncludeNSFW       bool   `json:"include_nsfw"`
		Sort              string `json:"sort"`
		RestrictSubreddit bool   `json:"restrict_subreddit"`
		After             string `json:"after,omitzero"`
		FetchMode         string `json:"fetch_mode"`           // "limit" or "catch-up"
		FetchLimit        int64  `json:"fetch_limit,omitzero"` // Only used if FetchMode is "limit"
	}

	LoadConfigurationAndStateInput struct {
		ID int64 `json:"id"`
	}

	LoadConfigurationAndStateOutput struct {
		Keyword    string       `json:"keyword"`
		Subreddits []*Subreddit `json:"subreddits,omitempty"`
	}

	UpdateStateValue struct {
		SubredditConfigurationID int64  `json:"subreddit_configuration_id"`
		Before                   string `json:"before"`
	}
	UpdateStateInput struct {
		Values []*UpdateStateValue `json:"values"`
	}

	UpdateStateOutput struct {
	}
)
