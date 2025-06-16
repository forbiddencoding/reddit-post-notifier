package entity

type (
	Recipient struct {
		ID    int64  `json:"id"`
		Type  string `json:"type"`
		Value string `json:"value"`
	}

	Subreddit struct {
		ID                int64  `json:"id"`
		Name              string `json:"name"`
		IncludeNSFW       bool   `json:"include_nsfw"`
		Sort              string `json:"sort"`
		RestrictSubreddit bool   `json:"restrict_subreddit"`
		After             string `json:"after,omitzero"`
	}

	LoadConfigurationAndStateInput struct {
		ID int64 `json:"id"`
	}

	LoadConfigurationAndStateOutput struct {
		Keyword    string       `json:"keyword"`
		Recipients []*Recipient `json:"recipients"`
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
