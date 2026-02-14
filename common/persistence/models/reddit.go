package models

import (
	"encoding/json"
	"github.com/google/uuid"
)

type (
	LoadConfigurationAndState struct {
		Keyword    string          `db:"keyword"`
		Subreddits json.RawMessage `db:"subreddits"`
		Recipients json.RawMessage `db:"recipients"`
	}

	GetSchedule struct {
		ID                uuid.UUID `db:"id"`
		SubredditID       uuid.UUID `db:"subreddit_id"`
		Subreddit         string    `db:"subreddit"`
		IncludeNSFW       bool      `db:"include_nsfw"`
		Sort              string    `db:"sort"`
		RestrictSubreddit bool      `db:"restrict_subreddit"`
		Keyword           string    `db:"keyword"`
		Schedule          string    `db:"schedule"`
		RecipientID       uuid.UUID `db:"recipient_id"`
		Address           string    `db:"address"`
	}

	ListSchedulesModel struct {
		ID         uuid.UUID       `db:"id"`
		Keyword    string          `db:"keyword"`
		Schedule   string          `db:"schedule"`
		Subreddits json.RawMessage `db:"subreddits"`
		Recipients json.RawMessage `db:"recipients"`
	}
)
