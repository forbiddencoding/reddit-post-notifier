package models

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type (
	LoadConfigurationAndState struct {
		ID                int64       `db:"id"`
		SubredditID       int64       `db:"subreddit_id"`
		Subreddit         string      `db:"subreddit"`
		IncludeNSFW       bool        `db:"include_nsfw"`
		Sort              string      `db:"sort"`
		RestrictSubreddit bool        `db:"restrict_subreddit"`
		Keyword           string      `db:"keyword"`
		Before            pgtype.Text `db:"before"`
		RecipientID       pgtype.Int8 `db:"recipient_id"`
		Address           string      `db:"address"`
	}

	GetSchedule struct {
		ID                int64       `db:"id"`
		SubredditID       int64       `db:"subreddit_id"`
		Subreddit         string      `db:"subreddit"`
		IncludeNSFW       bool        `db:"include_nsfw"`
		Sort              string      `db:"sort"`
		RestrictSubreddit bool        `db:"restrict_subreddit"`
		Keyword           string      `db:"keyword"`
		Schedule          string      `db:"schedule"`
		RecipientID       pgtype.Int8 `db:"recipient_id"`
		Address           string      `db:"address"`
	}

	ListSchedulesModel struct {
		ID                int64       `db:"id"`
		OwnerID           int64       `db:"owner_id"`
		SubredditID       int64       `db:"subreddit_id"`
		Subreddit         string      `db:"subreddit"`
		IncludeNSFW       bool        `db:"include_nsfw"`
		Sort              string      `db:"sort"`
		RestrictSubreddit bool        `db:"restrict_subreddit"`
		Keyword           string      `db:"keyword"`
		Schedule          string      `db:"schedule"`
		RecipientID       pgtype.Int8 `db:"recipient_id"`
		Address           string      `db:"address"`
	}
)
