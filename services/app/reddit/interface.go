package reddit

type (
	Recipient struct {
		ID            int64  `json:"id"`
		Type          string `json:"type"`
		Configuration string `json:"configuration"`
	}

	Subreddit struct {
		ID                int64  `json:"id"`
		Subreddit         string `json:"subreddit"`
		IncludeNSFW       bool   `json:"include_nsfw"`
		Sort              string `json:"sort"`
		RestrictSubreddit bool   `json:"restrict_subreddit"`
	}

	CreateScheduleInput struct {
		Keyword    string       `json:"keyword"`
		Subreddits []*Subreddit `json:"subreddits"`
		Schedule   string       `json:"schedule"` // Cron string
		Recipients []*Recipient `json:"recipients"`
		OwnerID    int64        `json:"owner_id"`
	}

	CreateScheduleOutput struct {
		Created    bool  `json:"created"`
		ScheduleID int64 `json:"schedule_id,omitempty"`
	}

	GetScheduleInput struct {
		ScheduleID int64 `json:"schedule_id"`
	}

	GetScheduleOutput struct {
		Keyword    string       `json:"keyword"`
		Subreddits []*Subreddit `json:"subreddits"`
		Schedule   string       `json:"schedule"` // Cron string
		Recipients []*Recipient `json:"recipients"`
		OwnerID    int64        `json:"owner_id"`
	}

	UpdateScheduleInput struct {
		ID         int64        `json:"id"`
		Keyword    string       `json:"keyword"`
		Subreddits []*Subreddit `json:"subreddits"`
		Schedule   string       `json:"schedule"` // Cron string
		Recipients []*Recipient `json:"recipients"`
	}

	UpdateScheduleOutput struct {
	}

	DeleteScheduleInput struct {
		ID int64 `json:"id"`
	}

	DeleteScheduleOutput struct {
	}

	Schedule struct {
		Status     string       `json:"status"`
		Keyword    string       `json:"keyword"`
		Subreddits []*Subreddit `json:"subreddits"`
		Schedule   string       `json:"schedule"` // Cron string
		Recipients []*Recipient `json:"recipients"`
		OwnerID    int64        `json:"owner_id"`
	}

	ListSchedulesInput struct {
		OwnerID int64 `json:"owner_id"`
	}

	ListSchedulesOutput struct {
		Schedules []*Schedule `json:"schedules"`
	}
)
