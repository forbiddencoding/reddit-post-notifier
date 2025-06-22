package entity

type (
	Recipient struct {
		ID      int64  `json:"id"`
		Address string `json:"address"`
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

	CreateScheduleSubreddit struct {
		ID                int64  `json:"id"`
		Subreddit         string `json:"subreddit"`
		IncludeNSFW       bool   `json:"include_nsfw"`
		Sort              string `json:"sort"`
		RestrictSubreddit bool   `json:"restrict_subreddit"`
	}

	CreateScheduleRecipient struct {
		ID      int64  `json:"id"`
		Address string `json:"address"`
	}

	CreateScheduleInput struct {
		ID         int64                      `json:"id"`
		Keyword    string                     `json:"keyword"`
		Schedule   string                     `json:"schedule"`
		OwnerID    int64                      `json:"owner_id"`
		Recipients []*CreateScheduleRecipient `json:"recipients"`
		Subreddits []*CreateScheduleSubreddit `json:"subreddits"`
	}

	CreateScheduleOutput struct {
	}

	GetScheduleInput struct {
		ID int64 `json:"id"`
	}

	GetScheduleOutput struct {
		Keyword    string       `json:"keyword"`
		Schedule   string       `json:"schedule"`
		OwnerID    int64        `json:"owner_id"`
		Recipients []*Recipient `json:"recipients"`
		Subreddits []*Subreddit `json:"subreddits,omitempty"`
	}

	DeleteScheduleInput struct {
		ID int64 `json:"id"`
	}

	DeleteScheduleOutput struct{}

	ListSchedulesInput struct {
		OwnerID int64 `json:"owner_id,omitempty"`
	}

	Schedule struct {
		ID         int64        `json:"id"`
		Keyword    string       `json:"keyword"`
		Schedule   string       `json:"schedule"`
		OwnerID    int64        `json:"owner_id"`
		Recipients []*Recipient `json:"recipients"`
		Subreddits []*Subreddit `json:"subreddits,omitempty"`
	}

	ListSchedulesOutput struct {
		Schedules []*Schedule `json:"schedules"`
	}

	UpdateScheduleInput struct {
		ID         int64        `json:"id"`
		Keyword    string       `json:"keyword"`
		Schedule   string       `json:"schedule"`
		Recipients []*Recipient `json:"recipients"`
		Subreddits []*Subreddit `json:"subreddits"`
	}

	UpdateScheduleOutput struct{}
)
