package reddit

import (
	"github.com/google/uuid"
	"time"
)

type (
	Recipient struct {
		ID      uuid.UUID `json:"id"`
		Address string    `json:"address" validate:"required,email"`
	}

	Subreddit struct {
		ID                uuid.UUID `json:"id"`
		Subreddit         string    `json:"subreddit" validate:"required"`
		IncludeNSFW       bool      `json:"includeNSFW"`
		Sort              string    `json:"sort"`
		RestrictSubreddit bool      `json:"restrictSubreddit"`
	}

	CreateScheduleInput struct {
		Keyword    string       `json:"keyword" validate:"required"`
		Subreddits []*Subreddit `json:"subreddits" validate:"required,min=1,max=10"`
		Schedule   string       `json:"schedule" validate:"cron"` // Cron string
		Recipients []*Recipient `json:"recipients" validate:"required,min=1,max=10"`
	}

	CreateScheduleOutput struct {
		Created    bool      `json:"created"`
		ScheduleID uuid.UUID `json:"scheduleID,omitempty"`
	}

	GetScheduleInput struct {
		ScheduleID uuid.UUID `json:"scheduleID"`
	}

	GetScheduleOutput struct {
		ID         uuid.UUID    `json:"id"`
		Keyword    string       `json:"keyword"`
		Subreddits []*Subreddit `json:"subreddits"`
		Schedule   string       `json:"schedule"` // Cron string
		Recipients []*Recipient `json:"recipients"`
		// --- Data from temporal
		NextActionTimes     []time.Time `json:"nextActionTimes"`
		Paused              bool        `json:"paused"`
		LastExecutionStatus string      `json:"lastExecutionStatus"`
	}

	UpdateScheduleInput struct {
		ID         uuid.UUID    `json:"id"`
		Keyword    string       `json:"keyword" validate:"required"`
		Subreddits []*Subreddit `json:"subreddits" validate:"required,min=1,max=10"`
		Schedule   string       `json:"schedule"` // Cron string
		Recipients []*Recipient `json:"recipients" validate:"required,min=1,max=10"`
	}

	UpdateScheduleOutput struct {
	}

	DeleteScheduleInput struct {
		ID uuid.UUID `json:"id"`
	}

	DeleteScheduleOutput struct {
	}

	Schedule struct {
		ID         uuid.UUID    `json:"id"`
		Keyword    string       `json:"keyword"`
		Subreddits []*Subreddit `json:"subreddits"`
		Schedule   string       `json:"schedule"` // Cron string
		Recipients []*Recipient `json:"recipients"`
	}

	ListSchedulesInput struct {
	}

	ListSchedulesOutput struct {
		Schedules []*Schedule `json:"schedules"`
	}
)
