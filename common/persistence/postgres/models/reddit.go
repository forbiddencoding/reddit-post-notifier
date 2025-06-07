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
		FetchMode         string         `db:"fetch_mode"`
		FetchLimit        sql.NullInt64  `db:"fetch_limit"`
		Before            sql.NullString `db:"before"`
	}
)
