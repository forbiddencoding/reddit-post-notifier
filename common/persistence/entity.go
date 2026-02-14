package persistence

import (
	"github.com/google/uuid"
	"time"
)

type (
	Recipient struct {
		ID      uuid.UUID `json:"id"`
		Address string    `json:"address"`
	}

	Subreddit struct {
		ID                uuid.UUID `json:"id"`
		Name              string    `json:"name"`
		IncludeNSFW       bool      `json:"include_nsfw"`
		Sort              string    `json:"sort"`
		RestrictSubreddit bool      `json:"restrict_subreddit"`
		Before            string    `json:"before,omitzero"`
	}

	LoadConfigurationAndStateInput struct {
		ID uuid.UUID `json:"id"`
	}

	LoadConfigurationAndStateOutput struct {
		Keyword    string       `json:"keyword"`
		Recipients []*Recipient `json:"recipients"`
		Subreddits []*Subreddit `json:"subreddits,omitempty"`
	}

	UpdateStateValue struct {
		SubredditConfigurationID uuid.UUID `json:"subreddit_configuration_id"`
		Before                   string    `json:"before"`
	}
	UpdateStateInput struct {
		Values []*UpdateStateValue `json:"values"`
	}

	UpdateStateOutput struct {
	}

	CreateScheduleSubreddit struct {
		ID                uuid.UUID `json:"id"`
		Subreddit         string    `json:"subreddit"`
		IncludeNSFW       bool      `json:"include_nsfw"`
		Sort              string    `json:"sort"`
		RestrictSubreddit bool      `json:"restrict_subreddit"`
	}

	CreateScheduleRecipient struct {
		ID      uuid.UUID `json:"id"`
		Address string    `json:"address"`
	}

	CreateScheduleInput struct {
		ID         uuid.UUID                  `json:"id"`
		Keyword    string                     `json:"keyword"`
		Schedule   string                     `json:"schedule"`
		Recipients []*CreateScheduleRecipient `json:"recipients"`
		Subreddits []*CreateScheduleSubreddit `json:"subreddits"`
	}

	CreateScheduleOutput struct {
	}

	GetScheduleInput struct {
		ID uuid.UUID `json:"id"`
	}

	GetScheduleOutput struct {
		ID         uuid.UUID    `json:"id"`
		Keyword    string       `json:"keyword"`
		Schedule   string       `json:"schedule"`
		Recipients []*Recipient `json:"recipients"`
		Subreddits []*Subreddit `json:"subreddits,omitempty"`
	}

	DeleteScheduleInput struct {
		ID uuid.UUID `json:"id"`
	}

	DeleteScheduleOutput struct{}

	ListSchedulesInput struct {
	}

	Schedule struct {
		ID         uuid.UUID    `json:"id"`
		Keyword    string       `json:"keyword"`
		Schedule   string       `json:"schedule"`
		Recipients []*Recipient `json:"recipients"`
		Subreddits []*Subreddit `json:"subreddits,omitempty"`
	}

	ListSchedulesOutput struct {
		Schedules []*Schedule `json:"schedules"`
	}

	UpdateScheduleInput struct {
		ID         uuid.UUID    `json:"id"`
		Keyword    string       `json:"keyword"`
		Schedule   string       `json:"schedule"`
		Recipients []*Recipient `json:"recipients"`
		Subreddits []*Subreddit `json:"subreddits"`
	}

	UpdateScheduleOutput struct{}

	Post struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		URL       string `json:"url"`
		Subreddit string `json:"subreddit"`
		NSFW      bool   `json:"nsfw"`
		Spoiler   bool   `json:"spoiler"`
		Ups       int    `json:"ups"`
		Downs     int    `json:"downs"`
		Thumbnail string `json:"thumbnail"`
		Created   string `json:"created"`
		Permalink string `json:"permalink"`
	}

	QueueItem struct {
		ID              uuid.UUID
		ConfigurationID uuid.UUID
		Post            []byte
		CreatedTime     time.Time
	}

	QueuePostsInput struct {
		Items []QueueItem
	}

	QueuePostsOutput struct{}

	GetPostsInput struct {
		ConfigurationID uuid.UUID
	}

	GetPostsOutput struct {
		Items []QueueItem
	}

	PopPostsInput struct {
		ConfigurationID uuid.UUID
	}

	PopPostsOutput struct{}
)
