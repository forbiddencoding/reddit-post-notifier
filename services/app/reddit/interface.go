package reddit

import "time"

type (
	Recipient struct {
		ID      int64  `json:"id"`
		Address string `json:"address" validate:"required,email"`
	}

	Subreddit struct {
		ID                int64  `json:"id"`
		Subreddit         string `json:"subreddit" validate:"required"`
		IncludeNSFW       bool   `json:"includeNSFW"`
		Sort              string `json:"sort"`
		RestrictSubreddit bool   `json:"restrictSubreddit"`
	}

	CreateScheduleInput struct {
		Keyword    string       `json:"keyword" validate:"required"`
		Subreddits []*Subreddit `json:"subreddits" validate:"required,min=1,max=10"`
		Schedule   string       `json:"schedule" validate:"cron"` // Cron string
		Recipients []*Recipient `json:"recipients" validate:"required,min=1,max=10"`
		OwnerID    int64        `json:"ownerID"`
	}

	CreateScheduleOutput struct {
		Created    bool  `json:"created"`
		ScheduleID int64 `json:"scheduleID,omitempty"`
	}

	GetScheduleInput struct {
		ScheduleID int64 `json:"scheduleID"`
	}

	GetScheduleOutput struct {
		ID         int64        `json:"id"`
		Keyword    string       `json:"keyword"`
		Subreddits []*Subreddit `json:"subreddits"`
		Schedule   string       `json:"schedule"` // Cron string
		Recipients []*Recipient `json:"recipients"`
		OwnerID    int64        `json:"ownerID"`
		// --- Data from temporal
		NextActionTimes     []time.Time `json:"nextActionTimes"`
		Paused              bool        `json:"paused"`
		LastExecutionStatus string      `json:"lastExecutionStatus"`
	}

	UpdateScheduleInput struct {
		ID         int64        `json:"id"`
		Keyword    string       `json:"keyword" validate:"required"`
		Subreddits []*Subreddit `json:"subreddits" validate:"required,min=1,max=10"`
		Schedule   string       `json:"schedule"` // Cron string
		Recipients []*Recipient `json:"recipients" validate:"required,min=1,max=10"`
	}

	UpdateScheduleOutput struct {
	}

	DeleteScheduleInput struct {
		ID int64 `json:"id"`
	}

	DeleteScheduleOutput struct {
	}

	Schedule struct {
		ID         int64        `json:"id"`
		Keyword    string       `json:"keyword"`
		Subreddits []*Subreddit `json:"subreddits"`
		Schedule   string       `json:"schedule"` // Cron string
		Recipients []*Recipient `json:"recipients"`
		OwnerID    int64        `json:"ownerID"`
	}

	ListSchedulesInput struct {
		OwnerID int64 `json:"ownerID"`
	}

	ListSchedulesOutput struct {
		Schedules []*Schedule `json:"schedules"`
	}
)
