package models

import "database/sql"

type (
	LoadConfigurationAndState struct {
		ID                int64          `db:"id"`
		SubredditID       int64          `db:"subreddit_id"`
		Subreddit         string         `db:"subreddit"`
		IncludeNSFW       bool           `db:"include_nsfw"`
		Sort              string         `db:"sort"`
		RestrictSubreddit bool           `db:"restrict_subreddit"`
		Keyword           string         `db:"keyword"`
		Before            sql.NullString `db:"before"`
		RecipientID       sql.NullInt64  `db:"recipient_id"`
		Type              string         `db:"type"`
		Value             string         `db:"value"`
	}
)
