package models

import (
	"database/sql"
)

type (
	LoadConfigurationAndState struct {
		ID                int64          `db:"id"`
		SubredditID       int64          `db:"subreddit_id"`
		Subreddit         string         `db:"subreddit"`
		IncludeNSFW       bool           `db:"include_nsfw"`
		Sort              string         `db:"sort"`
		RestrictSubreddit bool           `db:"restrict_subreddit"`
		Keyword           string         `db:"keyword"`
		Before            sql.NullString `db:"last_post"`
		RecipientID       sql.NullInt64  `db:"recipient_id"`
		Type              string         `db:"type"`
		Value             string         `db:"value"`
	}

	GetSchedule struct {
		ID                int64         `db:"id"`
		SubredditID       int64         `db:"subreddit_id"`
		Subreddit         string        `db:"subreddit"`
		IncludeNSFW       bool          `db:"include_nsfw"`
		Sort              string        `db:"sort"`
		RestrictSubreddit bool          `db:"restrict_subreddit"`
		Keyword           string        `db:"keyword"`
		Schedule          string        `db:"schedule"`
		RecipientID       sql.NullInt64 `db:"recipient_id"`
		Type              string        `db:"type"`
		Value             string        `db:"value"`
	}

	ListSchedulesModel struct {
		ID                int64         `db:"id"`
		OwnerID           int64         `db:"owner_id"`
		SubredditID       int64         `db:"subreddit_id"`
		Subreddit         string        `db:"subreddit"`
		IncludeNSFW       bool          `db:"include_nsfw"`
		Sort              string        `db:"sort"`
		RestrictSubreddit bool          `db:"restrict_subreddit"`
		Keyword           string        `db:"keyword"`
		Schedule          string        `db:"schedule"`
		RecipientID       sql.NullInt64 `db:"recipient_id"`
		Type              string        `db:"type"`
		Value             string        `db:"value"`
	}
)
